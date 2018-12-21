/*
Copyright 2018 Graham Lee Bevan <graham.bevan@ntlworld.com>

This file is part of gostint.

gostint is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

gostint is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with gostint.  If not, see <https://www.gnu.org/licenses/>.
*/

package jobqueues

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gbevan/gostint/approle"
	"github.com/gbevan/gostint/cleanup"
	"github.com/gbevan/gostint/logmsg"
	"github.com/gbevan/gostint/state"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/hashicorp/vault/api"
	. "github.com/visionmedia/go-debug" // nolint
	yaml "gopkg.in/yaml.v2"
)

const gostintUID = 2001
const gostintGID = 2001

var debug = Debug("jobqueues")

// AppRole holds Vault App Role details
type AppRole struct {
	ID   string
	Name string
}

// JobQueues holds jobqueue settings and state
type JobQueues struct {
	Db       *mgo.Database
	AppRole  *AppRole
	NodeUUID string
}

var jobQueues JobQueues

// Job structure to represent a job submission request
type Job struct {
	ID       bson.ObjectId `json:"_id"               bson:"_id,omitempty"`
	NodeUUID string        `json:"node_uuid"         bson:"node_uuid" description:"gostint node's unique identifier"`

	// These fields are passed from requestor in POSTed request:
	Qname        string `    json:"qname"             bson:"qname"`
	CubbyToken   string `    json:"cubby_token"       bson:"cubby_token"`
	CubbyPath    string `    json:"cubby_path"        bson:"cubby_path"`
	WrapSecretID string `    json:"wrap_secret_id"    bson:"wrap_secret_id" description:"Wrapping Token for the SecretID"`

	Payload string `         json:"payload"           bson:"payload" description:"Encrypted payload for the job from requestor, populated temporarily from the cubbyhole"`

	// These are populated from the decrypted payload
	// NOTE: ContainerImage: this may be passed in the content itself as meta data
	ContainerImage  string   `json:"container_image"   bson:"container_image"`
	ImagePullPolicy string   `json:"image_pull_policy" bson:"image_pull_policy"`
	Content         string   `json:"content"           bson:"content"`
	EntryPoint      []string `json:"entrypoint"        bson:"entrypoint"`
	Run             []string `json:"run"               bson:"run"`
	WorkingDir      string   `json:"working_directory" bson:"working_directory"`
	EnvVars         []string `json:"env_vars"          bson:"env_vars"`
	SecretRefs      []string `json:"secret_refs"       bson:"secret_refs"`
	SecretFileType  string   `json:"secret_file_type"  bson:"secret_file_type"`
	ContOnWarnings  bool     `json:"cont_on_warnings"  bson:"cont_on_warnings"`

	// These are returned
	Status        string    `json:"status"            bson:"status"`
	ReturnCode    int       `json:"return_code"       bson:"return_code"`
	Submitted     time.Time `json:"submitted"         bson:"submitted"`
	Started       time.Time `json:"started"           bson:"started,omitempty"`
	Ended         time.Time `json:"ended"             bson:"ended,omitempty"`
	Output        string    `json:"output"            bson:"output"`
	ContainerID   string    `json:"container_id"      bson:"container_id"`
	KillRequested bool      `json:"kill_requested"    bson:"kill_requested"`

	// Internal:
	contentRdr io.Reader
	secretsRdr io.Reader
}

func (job *Job) String() string {
	return fmt.Sprintf("ID: %s, Qname: %s, Submitted: %s, Status: %s, Started: %s, Image: %s, Content: %d", job.ID, job.Qname, job.Submitted, job.Status, job.Started, job.ContainerImage, len(job.Content))
}

// Init Initialises the job queues loop
func Init(db *mgo.Database, appRole *AppRole, nodeUUID string) {
	jobQueues.Db = db
	jobQueues.AppRole = appRole
	jobQueues.NodeUUID = nodeUUID

	clientAPIVer, dockerInfo, err := GetDockerInfo()
	if err != nil {
		logmsg.Error("Failed to get docker info: %v", err)
		panic(err)
	}
	logmsg.Info(
		"Starting job queue. Docker client api: v%v, server: v%s",
		clientAPIVer,
		dockerInfo.ServerVersion,
	)

	// start go routine to loop on the queues collection for new work
	// Qname defines the FIFO queue.
	go requestHandler()

	go killHandler()

	// Cleanup unused docker images
	go cleanup.Images()
}

// GetDockerInfo retrieves details of the docker client api and server info.
func GetDockerInfo() (string, *types.Info, error) {
	vctx, vcli, err := getDockerClient()
	if err != nil {
		return "", nil, err
	}
	vInfo, err := vcli.Info(*vctx)
	if err != nil {
		return vcli.ClientVersion(), nil, err
	}
	return vcli.ClientVersion(), &vInfo, nil
}

func requestHandler() {
	// TODO: Provide a Wake channel for immediate pull ???
	db := jobQueues.Db
	c := db.C("queues")

	for {
		if state.GetState() == "active" {
			var queues []string
			err := c.Find(bson.M{}).Distinct("qname", &queues)
			if err != nil {
				logmsg.Error("Error: Find queues failed: %s\n", err)
			}

			for _, q := range queues {
				job := Job{}

				chg := mgo.Change{
					Update: bson.M{"$set": bson.M{
						"status": "running",
					}},
					ReturnNew: false,
				}

				statusCond := []bson.M{}
				statusCond = append(statusCond, bson.M{"status": "queued"})
				statusCond = append(statusCond, bson.M{"status": "running"})

				_, err := c.Find(bson.M{"qname": q, "$or": statusCond}).Sort("submitted").Limit(1).Apply(chg, &job)
				if err != nil {
					if err.Error() == "not found" {
						continue
					}
					logmsg.Error("Pop from queue %s failed: %v\n", q, err)
				}

				// NOTE: if the returned job has status = "queued", then this has just
				// been atomically pop'd from the FIFO stack
				if job.Status != "queued" {
					break
				}

				// set node uuid that we are running on
				chg2 := mgo.Change{
					Update: bson.M{"$set": bson.M{
						"node_uuid": jobQueues.NodeUUID,
						"started":   time.Now(),
					}},
					ReturnNew: true,
				}
				_, err = c.FindId(job.ID).Apply(chg2, &job)
				if err != nil {
					logmsg.Error("Update to node uuid on queue %s failed: %v\n", q, err)
				}

				go job.runRequest()
			}
		} // if state active

		time.Sleep(1000 * time.Millisecond)
	}
}

func killHandler() {
	db := jobQueues.Db
	c := db.C("queues")

	for {
		var queues []Job
		err := c.Find(bson.M{
			"node_uuid":      jobQueues.NodeUUID,
			"kill_requested": true,
			"status": bson.M{
				"$nin": []string{
					"stopping",
					"failed",
					"success",
					"unkown",
				},
			},
		}).All(&queues)
		if err != nil {
			logmsg.Error("killHandler Find queues failed: %s\n", err)
			continue
		}

		for _, job := range queues {
			job.kill()
		}

		time.Sleep(5000 * time.Millisecond)
	}
}

// UpdateJob Atomically update a job in MongoDB
func (job *Job) UpdateJob(u bson.M) (*Job, error) {
	db := jobQueues.Db
	c := db.C("queues")

	chg := mgo.Change{
		Update:    bson.M{"$set": u},
		ReturnNew: true,
	}

	var resJob Job
	_, err := c.FindId(job.ID).Apply(chg, &resJob)
	if err != nil {
		logmsg.Error("update queue failed: %s\n", err)
		return nil, err
	}
	return &resJob, nil
}

func getDockerClient() (*context.Context, *client.Client, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	return &ctx, cli, err
}

func (job *Job) jobFailed(status string, err error) {
	job.UpdateJob(bson.M{
		"status": status,
		"ended":  time.Now(),
		"output": err.Error(),
	})
}

func (job *Job) resolveContentMeta() (*Meta, error) {
	// Handle Content
	meta := Meta{}
	if job.Content != "" {
		parts := strings.Split(job.Content, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Failed to parse invalid content")
		}
		// decode content base64
		data, err2 := base64.StdEncoding.DecodeString(parts[1])
		if err2 != nil {
			return nil, fmt.Errorf("Failed to decode content base64: %s", err2)
		}
		switch parts[0] {
		case "targz":
			// Put content reader in job for later copy into container as tar reader
			rdr, err := gzip.NewReader(strings.NewReader(string(data)))
			job.contentRdr = rdr
			if err != nil {
				return nil, fmt.Errorf("Failed to unzip content: %s", err)
			}
			break
		default:
			return nil, fmt.Errorf("Failed to extract content, unsupported archive format: %s", parts[0])
		}

		// Look for gostint.yml in content
		tempRdr, err2 := gzip.NewReader(strings.NewReader(string(data)))
		if err2 != nil {
			return nil, fmt.Errorf("Failed to unzip content to rdr for gostint.yaml: %s", err2)
		}
		tr := tar.NewReader(tempRdr)
		var bufMeta bytes.Buffer
		for {
			hdr, err2 := tr.Next()
			if err2 == io.EOF {
				break // End of archive
			}
			if err2 != nil {
				return nil, fmt.Errorf("Failed content tar: %s", err2)
			}
			if hdr.Name == "./gostint.yml" {
				if _, err := io.Copy(&bufMeta, tr); err != nil {
					return nil, fmt.Errorf("Failed extracting gostint.yml from tar: %s", err)
				}
			}

			// parse gostint.yml
			err := yaml.Unmarshal(bufMeta.Bytes(), &meta)
			if err != nil {
				return nil, fmt.Errorf("Failed parsing yaml in gostint.yml: %s", err)
			}
		} // for

		job.ContainerImage = meta.ContainerImage
	}
	return &meta, nil
}

func (job *Job) pullDockerImage(ctx *context.Context, cli *client.Client) (string, error) {

	if job.ContainerImage == "" {
		errmsg := "ContainerImage is empty"
		logmsg.Error(errmsg)
		return "", errors.New(errmsg)
	}

	if strings.Index(job.ContainerImage, ":") == -1 {
		job.ContainerImage = fmt.Sprintf("%s:latest", job.ContainerImage)
	}

	var imgRef string
	if strings.Contains(job.ContainerImage, "/") {
		// TODO: Support logins
		imgRef = job.ContainerImage
	} else {
		imgRef = fmt.Sprintf("docker.io/%s", job.ContainerImage)
	}

	// Get list of images on host
	imgList, err := cli.ImageList(*ctx, types.ImageListOptions{
		All: true,
	})
	if err != nil {
		return "", err
	}
	imgAlreadyPulled := false
	imgID := ""
	for _, img := range imgList {
		if len(img.RepoTags) > 0 && img.RepoTags[0] == job.ContainerImage {
			imgAlreadyPulled = true
			imgID = img.ID
		}
	}

	if !imgAlreadyPulled || job.ImagePullPolicy == "Always" {
		var reader io.ReadCloser
		// var err2 error
		err = retry.Do(
			func() error {
				logmsg.Info("Trying to pull image %s", imgRef)
				reader, err = cli.ImagePull(*ctx, imgRef, types.ImagePullOptions{})
				if err != nil {
					logmsg.Warn("ImagePull imgRef: %s, %v, will retry", imgRef, err)
					return err
				}
				// return nil
				// },
				// )
				// if err != nil {
				// 	logmsg.Error("ImagePull imgRef: %s, %v, exceeded retries", imgRef, err)
				// 	return "", err
				// }

				// This is currently needed to ensure images are downloaded before we
				// move on to creating containers...
				scanner := bufio.NewScanner(reader)
				for scanner.Scan() {
					pullStatus := make(map[string]interface{})
					jsonStr := []byte(scanner.Text())
					err = json.Unmarshal(jsonStr, &pullStatus)
					if err != nil {
						logmsg.Error("parsing docker status: %v", err)
						return err
					}
					// logmsg.Debug("image: %s", pullStatus)
					// progress := ""
					if pullStatus["progress"] != nil {
						progress := pullStatus["progress"].(string)
						logmsg.Info("%v: %s", pullStatus["status"], progress)
					} else {
						logmsg.Warn("image: jsonStr: %s", jsonStr)
						if pullStatus["errorDetail"] != nil {
							return fmt.Errorf("%v", pullStatus["errorDetail"])
						}
						logmsg.Info("%v", pullStatus["status"])
					}
				}
				if err = scanner.Err(); err != nil {
					return err
				}
				// io.Copy(os.Stdout, reader)
				return nil
			},
		)
		if err != nil {
			logmsg.Error("ImagePull imgRef: %s, %v, exceeded retries", imgRef, err)
			return "", err
		}

	} else {
		logmsg.Info("Image %s already pulled & image_pull_policy: %s", job.ContainerImage, job.ImagePullPolicy)
	}

	if imgID == "" {
		// Get image ID
		imgList, err = cli.ImageList(*ctx, types.ImageListOptions{
			All: true,
		})
		if err != nil {
			return "", err
		}
		for _, img := range imgList {
			if len(img.RepoTags) > 0 && img.RepoTags[0] == job.ContainerImage {
				imgID = img.ID
			}
		}
	}
	return imgID, nil
}

func (job *Job) createDockerContainer(ctx *context.Context, cli *client.Client, imgID string) (container.ContainerCreateCreatedBody, error) {
	cleanup.ImageUsed(imgID, time.Now())

	cfg := container.Config{
		Image: job.ContainerImage,
		Cmd:   job.Run,
		Tty:   true,
		User:  fmt.Sprintf("%d:%d", gostintUID, gostintGID),
		Env:   job.EnvVars,
	}

	if len(job.EntryPoint) != 0 {
		cfg.Entrypoint = job.EntryPoint
	}

	if job.WorkingDir != "" {
		cfg.WorkingDir = job.WorkingDir
	}

	var resp container.ContainerCreateCreatedBody
	resp, err := cli.ContainerCreate(*ctx, &cfg, nil, nil, "")
	if err != nil {
		logmsg.Error("ContainerCreate cfg: %v", cfg)
		logmsg.Error("err: %v", err)
		return resp, err
	}
	return resp, nil
}

func (job *Job) metaFromDockerContainer(ctx *context.Context, cli *client.Client, containerID string, srcPath string) (map[interface{}]interface{}, error) {
	rdr, _, err := cli.CopyFromContainer(*ctx, containerID, srcPath)
	defer func() {
		if rdr != nil {
			rdr.Close()
		}
	}()
	if err != nil {
		// if strings.Contains(err.Error(), "Could not find the file") {
		if strings.Contains(err.Error(), "No such container:path") {
			logmsg.Debug("metaFromDockerContainer err: %s", err)
		} else {
			logmsg.Error("metaFromDockerContainer err: %s", err)
		}
		return nil, nil
	}

	// rdr here is for a TAR ball - need to extract the file then UnMarshal
	tr := tar.NewReader(rdr)
	var bufMeta bytes.Buffer
	for {
		hdr, err2 := tr.Next()
		if err2 == io.EOF {
			break // End of archive
		}
		if err2 != nil {
			return nil, fmt.Errorf("Failed %s extraction tar: %s", srcPath, err2)
		}
		if hdr.Name == srcPath {
			if _, err = io.Copy(&bufMeta, tr); err != nil {
				return nil, fmt.Errorf("Failed extracting %s from container's tar: %s", srcPath, err)
			}
		}
	}

	// parse meta yaml
	meta := make(map[interface{}]interface{})
	err = yaml.Unmarshal(bufMeta.Bytes(), &meta)
	if err != nil {
		return nil, fmt.Errorf("Failed parsing yaml in %s: %s", srcPath, err)
	}
	return meta, nil
}

func (job *Job) runRequest() {
	if job.KillRequested {
		job.UpdateJob(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": "job killed",
		})
		return
	}

	ctx, cli, err := getDockerClient()
	if err != nil {
		job.jobFailed("failed", err)
		return
	}

	token, vclient, err := approle.Authenticate(jobQueues.AppRole.ID, job.WrapSecretID)
	if err != nil {
		job.UpdateJob(bson.M{
			"status": "notauthorised",
			"ended":  time.Now(),
			"output": err.Error(),
		})
		return
	}

	// Set authenticated token
	vclient.SetToken(token)

	defer func() {
		// Revoke the ephemeral token
		_, err = vclient.Logical().Write("auth/token/revoke-self", nil)
		if err != nil {
			logmsg.Error("revoking token after job completed: %s", err)
		}
	}()

	// Decrypt the payload and merge into jobRequest
	resp, err := vclient.Logical().Write(
		fmt.Sprintf("transit/decrypt/%s", jobQueues.AppRole.Name),
		map[string]interface{}{
			"ciphertext": job.Payload,
		},
	)
	if err != nil {
		job.UpdateJob(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Failed to decrypt payload via vault: %s", err.Error()),
		})
		return
	}
	payloadJSON, err2 := base64.StdEncoding.DecodeString(resp.Data["plaintext"].(string))
	if err2 != nil {
		job.UpdateJob(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Failed to decode payload content base64: %s", err2),
		})
		return
	}
	var payloadObj Job
	err = json.Unmarshal(payloadJSON, &payloadObj)
	if err != nil {
		job.UpdateJob(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Failed unmarshaling json from payload: %s", err),
		})
		return
	}

	// sanity check
	if payloadObj.Qname != job.Qname {
		job.UpdateJob(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("payload qname and job request qname do not match: '%s' != '%s'", payloadObj.Qname, job.Qname),
		})
		return
	}
	// Cleanup job of resolved items
	job.CubbyToken = ""
	job.CubbyPath = ""
	job.WrapSecretID = ""
	job.Payload = ""

	/****************************************
	 * meta data layering:
	 *  1) docker image /gostint_image.yml
	 *  2) Content tar /gostint.yml
	 *  3) Command line SecretRefs
	 */

	// Get Content to resolve gostint.yml meta data
	job.Content = payloadObj.Content
	contentMeta, err := job.resolveContentMeta()
	if err != nil {
		job.jobFailed("failed", err)
		return
	}

	// resolve image
	job.ContainerImage = contentMeta.ContainerImage
	if payloadObj.ContainerImage != "" {
		job.ContainerImage = payloadObj.ContainerImage
	}
	job.ImagePullPolicy = payloadObj.ImagePullPolicy

	job.EntryPoint = payloadObj.EntryPoint
	job.Run = payloadObj.Run
	job.WorkingDir = payloadObj.WorkingDir
	job.EnvVars = payloadObj.EnvVars
	job.SecretFileType = payloadObj.SecretFileType
	job.ContOnWarnings = payloadObj.ContOnWarnings

	// get image
	imgID, err := job.pullDockerImage(ctx, cli)
	if err != nil {
		job.jobFailed("failed", err)
		return
	}

	// Create Container, without running
	containerBody, err := job.createDockerContainer(ctx, cli, imgID)
	if err != nil {
		job.jobFailed("failed", err)
		return
	}

	job.UpdateJob(bson.M{
		"container_id": containerBody.ID,
	})

	logmsg.Info("Created container ID: %s", containerBody.ID)

	// Automatically clean up the container
	defer func() {
		logmsg.Debug("Removing container %s", containerBody.ID)
		rmOpts := types.ContainerRemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   false,
			Force:         true,
		}
		if errD := cli.ContainerRemove(*ctx, containerBody.ID, rmOpts); errD != nil {
			logmsg.Error("removing container: %s", errD)
		}
	}()

	if job.SecretFileType == "" {
		job.SecretFileType = "yaml"
	}

	if job.ImagePullPolicy == "" {
		job.ImagePullPolicy = "IfNotPresent"
	}

	// Get /gostint_image.yml from Container, merge fields
	imageMeta, err := job.metaFromDockerContainer(ctx, cli, containerBody.ID, "gostint_image.yml")
	if err != nil {
		job.jobFailed("failed", err)
		return
	}

	// Merge fields from layered meta sources (image, content, payload)
	srs, ok := imageMeta["secret_refs"].([]interface{})
	if ok {
		for _, sr := range srs {
			job.SecretRefs = append(job.SecretRefs, sr.(string))
		}
	}
	job.SecretRefs = append(job.SecretRefs, contentMeta.SecretRefs...)
	job.SecretRefs = append(job.SecretRefs, payloadObj.SecretRefs...)

	if job.ImagePullPolicy != "IfNotPresent" && job.ImagePullPolicy != "Always" {
		job.UpdateJob(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Incorrect value for image_pull_policy: %s", job.ImagePullPolicy),
		})
		return
	}

	// Allow SecretRefs to be passed in job, e.g.
	// SecretRefs: see tests/job1.json
	// briefly cache the path response to allow multiple values to be extracted
	// without needing to re-query the Vault.
	secRefRe, err := regexp.Compile("^(\\w+)@([\\w\\-_/]+)\\.([\\w\\-_]+)$")
	if err != nil {
		job.UpdateJob(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Regex compilation error: %s", err),
		})
		return
	}
	secrets := map[string]string{}
	secrets["TOKEN"] = token
	cache := map[string]*api.Secret{}
	for _, v := range job.SecretRefs {
		parts := secRefRe.FindStringSubmatch(v)
		if len(parts) < 3 {
			job.UpdateJob(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Secretref is unparseable: %s", v),
			})
			return
		}
		secVarName := parts[1]
		secPath := parts[2]
		secKey := parts[3]

		if secVarName == "" {
			job.UpdateJob(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Target variable name in secretref cannot be empty"),
			})
			return
		}
		if secPath == "" {
			job.UpdateJob(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Secretref must have a path"),
			})
			return
		}
		if secKey == "" {
			job.UpdateJob(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Secretref must have a path.key"),
			})
			return
		}

		// var secretValues api.Secret
		secretValues := cache[secPath]
		if secretValues == nil {
			secretValues, err = vclient.Logical().Read(secPath)

			if err != nil {
				job.UpdateJob(bson.M{
					"status": "failed",
					"ended":  time.Now(),
					"output": fmt.Sprintf("Failed to retrieve secret %s from vault err: %v", secPath, err),
				})
				return
			}

			if secretValues == nil {
				job.UpdateJob(bson.M{
					"status": "failed",
					"ended":  time.Now(),
					"output": fmt.Sprintf("Failed to retrieve secret %s from vault: response is nil", secPath),
				})
				return
			}

			if !job.ContOnWarnings && len(secretValues.Warnings) > 0 {
				job.UpdateJob(bson.M{
					"status": "failed",
					"ended":  time.Now(),
					"output": fmt.Sprintf("FailOnWarnings from vault path %s lookups: %v", secPath, secretValues.Warnings),
				})
				return
			}

			cache[secPath] = secretValues
		}
		var data interface{}
		if secretValues.Data["data"] != nil { // kv v2
			data = secretValues.Data["data"]
		} else if secretValues.Data != nil { // kv v1
			data = secretValues.Data
		} else {
			job.UpdateJob(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("No data returned from vault path %s.%s", secPath, secKey),
			})
			return
		}
		secVal := (data.(map[string]interface{}))[secKey]
		if secVal == nil {
			job.UpdateJob(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Failed retrieving from vault path %s.%s", secPath, secKey),
			})
			return
		}
		secrets[secVarName] = (data.(map[string]interface{}))[secKey].(string)
	} // for SecretRefs

	// Create tar rdr for /secrets.yml|json in container
	var entries []TarEntry
	if job.SecretFileType == "yaml" {
		secretsYAML, err2 := yaml.Marshal(secrets)
		if err2 != nil {
			job.UpdateJob(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Failed to Marshal secrets to yaml for container injection: %s", err),
			})
			return
		}
		yamlHdr := []byte("---\n# gostint vault secrets injected:\n")
		secretsYAML = append(yamlHdr, secretsYAML...)

		entries = []TarEntry{
			{Name: "secrets.yml", Content: secretsYAML},
		}
	} else if job.SecretFileType == "json" {
		secretsJSON, err2 := json.Marshal(secrets)
		if err2 != nil {
			job.UpdateJob(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Failed to Marshal secrets to json for container injection: %s", err2),
			})
			return
		}
		entries = []TarEntry{
			{Name: "secrets.json", Content: secretsJSON},
		}
	} else {
		job.UpdateJob(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Invalid SecretFileType: '%s'", job.SecretFileType),
		})
		return
	}

	job.secretsRdr, err = createTar(&entries)
	if err != nil {
		job.UpdateJob(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": err.Error(),
		})
		return
	}

	if job.KillRequested {
		job.UpdateJob(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": "job killed",
		})
		return
	}

	err = job.runContainer(ctx, cli, containerBody.ID)
	if err != nil {
		job.UpdateJob(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Run container failed: %s", err),
		})
		return
	}
}

// Meta defines the format of the gostint.yml file
type Meta struct {
	ContainerImage string   `yaml:"container_image"`
	SecretRefs     []string `yaml:"secret_refs"`
}

func (job *Job) runContainer(ctx *context.Context, cli *client.Client, containerID string) error {
	// Copy content into container prior to start it
	opts := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	}
	err := cli.CopyToContainer(*ctx, containerID, "/", job.contentRdr, opts)
	if err != nil {
		return err
	}

	// Copy secrets into container prior to start it
	err = cli.CopyToContainer(*ctx, containerID, "/", job.secretsRdr, opts)
	if err != nil {
		return err
	}

	err = addUser(*ctx, cli, containerID, "gostint", gostintUID, gostintGID, "/tmp")
	if err != nil {
		return err
	}

	if err = cli.ContainerStart(*ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	statusCh, errCh := cli.ContainerWait(*ctx, containerID, "")
	var statusBody container.ContainerWaitOKBody
	select {
	case err2 := <-errCh:
		if err2 != nil {
			return err2
		}
	case statusBody = <-statusCh:
	}
	status := statusBody.StatusCode
	if status == 0 {
		logmsg.Info("status from container wait: %d", status)
	} else {
		logmsg.Error("status from container wait: %d", status)
	}

	out, err := cli.ContainerLogs(*ctx, containerID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	io.Copy(&buf, out)

	finalStatus := "success"
	if status != 0 {
		finalStatus = "failed"
	}

	job.UpdateJob(bson.M{
		"status":      finalStatus,
		"ended":       time.Now(),
		"output":      buf.String(),
		"return_code": status,
	})

	return nil
}

func addUser(ctx context.Context, cli *client.Client, containerID, name string, uid, gid int, home string) error {
	// Get /etc/passwd
	rdr, _, err := cli.CopyFromContainer(ctx, containerID, "/etc/passwd")
	if err != nil {
		return err
	}

	defer func() {
		rdr.Close()
	}()

	tr := tar.NewReader(rdr)
	var bufMeta bytes.Buffer
	for {
		hdr, err2 := tr.Next()
		if err2 == io.EOF {
			break // End of archive
		}
		if err2 != nil {
			return fmt.Errorf("Failed /etc/passwd extraction tar: %s", err2)
		}
		if hdr.Name == "passwd" {
			if _, err = io.Copy(&bufMeta, tr); err != nil {
				return fmt.Errorf("Failed extracting /etc/passwd from container's tar: %s", err)
			}
		}
	}
	passwd := bufMeta.String()

	// add gostint user
	passwd = fmt.Sprintf("%s%s:x:%d:%d:%s:%s:/bin/sh\n", passwd, name, uid, gid, name, home)

	entries := []TarEntry{
		{Name: "passwd", Content: bytes.NewBufferString(passwd).Bytes()},
	}
	tarRdr, err := createTar(&entries)
	if err != nil {
		return err
	}

	opts := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	}
	err = cli.CopyToContainer(ctx, containerID, "/etc", tarRdr, opts)
	if err != nil {
		return err
	}

	return nil
}

// TarEntry holds a tar file entity
type TarEntry struct {
	Name    string
	Content []byte
}

func createTar(entries *[]TarEntry) (rdrClose io.Reader, err error) {
	var buf bytes.Buffer
	wtr := tar.NewWriter(&buf)

	for _, entry := range *entries {
		hdr := &tar.Header{
			Name: entry.Name,
			Mode: 0444,
			Size: int64(len(entry.Content)),
		}
		if err = wtr.WriteHeader(hdr); err != nil {
			return nil, fmt.Errorf("Failed to write %s to tar header for container injection: %s", entry.Name, err)
		}
		if _, err = wtr.Write(entry.Content); err != nil {
			return nil, fmt.Errorf("Failed to write %s data to tar for container injection: %s", entry.Name, err)
		}
	}

	if err = wtr.Close(); err != nil {
		return nil, fmt.Errorf("Failed to close tar of for container injection: %s", err)
	}
	return strings.NewReader(buf.String()), nil
}

func (job *Job) kill() error {
	if job.ContainerID == "" {
		return errors.New("job.ContainerID is missing")
	}
	logmsg.Info("Stopping container %s", job.ContainerID)

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	job.UpdateJob(bson.M{
		"status": "stopping",
	})

	go func() {
		timeout := time.Duration(15) * time.Second

		err = cli.ContainerStop(ctx, job.ContainerID, &timeout)
		if err != nil {
			logmsg.Error("Stop container %s request failed: %s", job.ContainerID, err)
		}

		err = cli.ContainerKill(ctx, job.ContainerID, "KILL")
		if err != nil {
			if !strings.HasSuffix(err.Error(), "is not running") {
				logmsg.Error("Kill container %s request failed: %s", job.ContainerID, err)
			}
		}
	}()

	return nil
}

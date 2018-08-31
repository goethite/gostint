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
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	client "docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/container"
	"github.com/gbevan/gostint/approle"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/hashicorp/vault/api"
	. "github.com/visionmedia/go-debug" // nolint
	yaml "gopkg.in/yaml.v2"
)

const gostintUID = 2001
const gostintGID = 2001

var debug = Debug("jobqueues")

type PulledImage struct {
	When time.Time
}

type JobQueues struct {
	Db           *mgo.Database
	AppRoleID    string
	NodeUUID     string
	PulledImages map[string]PulledImage
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
	ContainerImage string   `json:"container_image"   bson:"container_image"`
	Content        string   `json:"content"           bson:"content"`
	EntryPoint     []string `json:"entrypoint"        bson:"entrypoint"`
	Run            []string `json:"run"               bson:"run"`
	WorkingDir     string   `json:"working_directory" bson:"working_directory"`
	SecretRefs     []string `json:"secret_refs"       bson:"secret_refs"`
	SecretFileType string   `json:"secret_file_type"  bson:"secret_file_type"`
	ContOnWarnings bool     `json:"cont_on_warnings"  bson:"cont_on_warnings"`

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
	return fmt.Sprintf("ID: %s, Qname: %s, Submitted: %s, Status: %s, Started: %s", job.ID, job.Qname, job.Submitted, job.Status, job.Started)
}

// Init Initialises the job queues loop
func Init(db *mgo.Database, appRoleID string, nodeUUID string) {
	jobQueues.Db = db
	jobQueues.AppRoleID = appRoleID
	jobQueues.PulledImages = make(map[string]PulledImage)
	jobQueues.NodeUUID = nodeUUID
	// start go routine to loop on the queues collection for new work
	// Qname defines the FIFO queue.
	// Provide a Wake channel for immediate pull ???
	go requestHandler()

	go killHandler()
}

func requestHandler() {
	db := jobQueues.Db
	c := db.C("queues")

	for {
		var queues []string
		err := c.Find(bson.M{}).Distinct("qname", &queues)
		if err != nil {
			log.Printf("Error: Find queues failed: %s\n", err)
		}

		for _, q := range queues {
			// log.Printf("queue: %s\n", q)
			job := Job{}

			chg := mgo.Change{
				Update: bson.M{"$set": bson.M{
					// "node_uuid": jobQueues.NodeUUID,
					"status": "running",
					// "started": time.Now(),
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
				log.Printf("Error: Pop from queue %s failed: %v\n", q, err)
			}
			// log.Printf("ci: %v", ci)
			// log.Println("After ci")

			// NOTE: if the returned job has status = "queued", then this has just
			// been atomically pop'd from the FIFO stack
			if job.Status != "queued" {
				break
			}
			// log.Printf("id: %v", ci.InsertedId)

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
				log.Printf("Error: Update to node uuid on queue %s failed: %v\n", q, err)
			}

			go job.runRequest()
		}

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
			log.Printf("killHandler Error: Find queues failed: %s\n", err)
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

	_, err := c.FindId(job.ID).Apply(chg, job)
	if err != nil {
		log.Printf("Error: update queue failed: %s\n", err)
		return nil, err
	}
	return job, nil
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

	token, client, err := approle.Authenticate(jobQueues.AppRoleID, job.WrapSecretID)
	if err != nil {
		job.UpdateJob(bson.M{
			"status": "notauthorised",
			"ended":  time.Now(),
			"output": err.Error(),
		})
		return
	}

	// Set authenticated token
	client.SetToken(token)

	defer func() {
		// Revoke the ephemeral token
		_, err = client.Logical().Write("auth/token/revoke-self", nil)
		if err != nil {
			log.Printf("Error: revoking token after job completed: %s", err)
		}
	}()

	// Decrypt the payload and merge into jobRequest
	resp, err := client.Logical().Write("transit/decrypt/gostint", map[string]interface{}{
		"ciphertext": job.Payload,
	})
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
	// Merge required fields from payload
	job.ContainerImage = payloadObj.ContainerImage
	job.Content = payloadObj.Content
	job.EntryPoint = payloadObj.EntryPoint
	job.Run = payloadObj.Run
	job.WorkingDir = payloadObj.WorkingDir
	job.SecretRefs = payloadObj.SecretRefs
	job.SecretFileType = payloadObj.SecretFileType
	job.ContOnWarnings = payloadObj.ContOnWarnings

	if job.SecretFileType == "" {
		job.SecretFileType = "yaml"
	}

	// Allow SecretRefs to be passed in job, e.g.
	// SecretRefs: see tests/job1.json
	// briefly cache the path response to allow multiple values to be extracted
	// without needing to re-query the Vault.
	secRefRe, err := regexp.Compile("^(\\w+)@([\\w\\-_/]+)\\.(([\\w\\-_]+))$")
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
			secretValues, err = client.Logical().Read(secPath)

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
		secretsJSON, err := json.Marshal(secrets)
		if err != nil {
			job.UpdateJob(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Failed to Marshal secrets to json for container injection: %s", err),
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

	// Handle Content
	if job.Content != "" {
		parts := strings.Split(job.Content, ",")
		if len(parts) != 2 {
			job.UpdateJob(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Failed to parse invalid content"),
			})
			return
		}
		// decode content base64
		data, err2 := base64.StdEncoding.DecodeString(parts[1])
		if err2 != nil {
			job.UpdateJob(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Failed to decode content base64: %s", err2),
			})
			return
		}
		switch parts[0] {
		case "targz":
			// Put content reader in job for later copy into container as tar reader
			job.contentRdr, err = gzip.NewReader(strings.NewReader(string(data)))
			if err != nil {
				job.UpdateJob(bson.M{
					"status": "failed",
					"ended":  time.Now(),
					"output": fmt.Sprintf("Failed to unzip content: %s", err),
				})
				return
			}
			break
		default:
			job.UpdateJob(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Failed to extract content, unsupported archive format: %s", parts[0]),
			})
			return
		}

		// Look for gostint.yml in content
		// fmt.Printf("tar data: %v", string(data))
		meta := Meta{}
		tempRdr, err := gzip.NewReader(strings.NewReader(string(data)))
		if err != nil {
			job.UpdateJob(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Failed to unzip content to rdr for gostint.yaml: %s", err),
			})
			return
		}
		tr := tar.NewReader(tempRdr)
		var bufMeta bytes.Buffer
		for {
			hdr, err2 := tr.Next()
			if err2 == io.EOF {
				break // End of archive
			}
			if err2 != nil {
				job.UpdateJob(bson.M{
					"status": "failed",
					"ended":  time.Now(),
					"output": fmt.Sprintf("Failed content tar: %s", err2),
				})
				return
			}
			if hdr.Name == "./gostint.yml" {
				if _, err = io.Copy(&bufMeta, tr); err != nil {
					job.UpdateJob(bson.M{
						"status": "failed",
						"ended":  time.Now(),
						"output": fmt.Sprintf("Failed extracting gostint.yml from tar: %s", err),
					})
					return
				}
			}

			// fmt.Printf("bufMeta: %s\n", bufMeta.Bytes())
			// parse gostint.yml
			err := yaml.Unmarshal(bufMeta.Bytes(), &meta)
			if err != nil {
				job.UpdateJob(bson.M{
					"status": "failed",
					"ended":  time.Now(),
					"output": fmt.Sprintf("Failed parsing yaml in gostint.yml: %s", err),
				})
				return
			}
		} // for

		if job.ContainerImage == "" {
			job.ContainerImage = meta.ContainerImage
		}
	}

	if job.KillRequested {
		job.UpdateJob(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": "job killed",
		})
		return
	}

	err = job.runContainer()
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
	ContainerImage string `yaml:"container_image"`
}

func (job *Job) runContainer() error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	if job.ContainerImage == "" {
		errmsg := "Error job.ContainerImage is empty"
		log.Println(errmsg)
		return errors.New(errmsg)
	}

	if strings.Index(job.ContainerImage, ":") == -1 {
		job.ContainerImage = fmt.Sprintf("%s:latest", job.ContainerImage)
	}

	imgRef := fmt.Sprintf("docker.io/%s", job.ContainerImage)

	// Get list of images on host
	imgList, err := cli.ImageList(ctx, types.ImageListOptions{
		All: true,
	})
	if err != nil {
		return err
	}
	imgAlreadyPulled := false
	for _, img := range imgList {
		if len(img.RepoTags) > 0 && img.RepoTags[0] == job.ContainerImage {
			imgAlreadyPulled = true
		}
	}

	var imgAgeDays time.Duration
	if imgAlreadyPulled {
		imgAgeDays = (time.Now().Sub(jobQueues.PulledImages[job.ContainerImage].When)) / (time.Hour * 24)
	}

	if !imgAlreadyPulled || imgAgeDays > 1 {
		// reader, err := cli.ImagePull(ctx, "docker.io/library/busybox", types.ImagePullOptions{})
		reader, err2 := cli.ImagePull(ctx, imgRef, types.ImagePullOptions{})
		if err2 != nil {
			log.Printf("ImagePull imgRef: %s", imgRef)
			return err2
		}

		// This is currently needed to ensure images are downloaded before we
		// move on to creating containers...
		io.Copy(os.Stdout, reader)

		jobQueues.PulledImages[job.ContainerImage] = PulledImage{When: time.Now()}
	} else {
		log.Printf("Image %s already pulled, age: %d", job.ContainerImage, imgAgeDays)
	}

	cfg := container.Config{
		Image: job.ContainerImage,
		// Cmd:   []string{"echo", "hello world"},
		Cmd:  job.Run,
		Tty:  true,
		User: fmt.Sprintf("%d:%d", gostintUID, gostintGID),
	}

	if len(job.EntryPoint) != 0 {
		cfg.Entrypoint = job.EntryPoint
	}

	if job.WorkingDir != "" {
		cfg.WorkingDir = job.WorkingDir
	}

	resp, err := cli.ContainerCreate(ctx, &cfg, nil, nil, "")
	if err != nil {
		log.Printf("ContainerCreate cfg: %v", cfg)
		return err
	}

	// save contentRdr and secretsRdr as UpdateJob() drops them from job
	contentRdr := job.contentRdr
	secretsRdr := job.secretsRdr

	job.UpdateJob(bson.M{
		"container_id": resp.ID,
	})

	log.Printf("Created container ID: %s", resp.ID)

	defer func() {
		log.Printf("Removing container %s", resp.ID)
		rmOpts := types.ContainerRemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   false,
			Force:         true,
		}
		if errD := cli.ContainerRemove(ctx, resp.ID, rmOpts); errD != nil {
			log.Printf("Error: removing container: %s", errD)
		}
	}()

	// Copy content into container prior to start it
	opts := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
		// CopyUIDGID:                true,
	}
	err = cli.CopyToContainer(ctx, resp.ID, "/", contentRdr, opts)
	if err != nil {
		return err
	}

	// Copy secrets into container prior to start it
	err = cli.CopyToContainer(ctx, resp.ID, "/", secretsRdr, opts)
	if err != nil {
		return err
	}

	err = addUser(cli, ctx, resp.ID, "gostint", gostintUID, gostintGID, "/tmp")
	if err != nil {
		return err
	}

	if err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, "")
	var statusBody container.ContainerWaitOKBody
	select {
	case err2 := <-errCh:
		if err2 != nil {
			return err2
		}
	case statusBody = <-statusCh:
	}
	status := statusBody.StatusCode
	log.Printf("status from container wait: %d", status)

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	io.Copy(&buf, out)
	// fmt.Println(buf.String())

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

	// rmOpts := types.ContainerRemoveOptions{
	// 	RemoveVolumes: true,
	// 	RemoveLinks:   false,
	// 	Force:         true,
	// }
	// if err := cli.ContainerRemove(ctx, resp.ID, rmOpts); err != nil {
	// 	log.Printf("Error: removing container: %s", err)
	// }

	return nil
}

func addUser(cli *client.Client, ctx context.Context, containerID, name string, uid, gid int, home string) error {
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
	// log.Printf("bufMeta: %v", bufMeta.String())
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
		// CopyUIDGID:                true,
	}
	err = cli.CopyToContainer(ctx, containerID, "/etc", tarRdr, opts)
	if err != nil {
		return err
	}

	return nil
}

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
	log.Printf("Stopping container %s", job.ContainerID)

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
			log.Printf("Stop container %s request failed: %s", job.ContainerID, err)
		}

		err = cli.ContainerKill(ctx, job.ContainerID, "KILL")
		if err != nil {
			if !strings.HasSuffix(err.Error(), "is not running") {
				log.Printf("Kill container %s request failed: %s", job.ContainerID, err)
			}
		}
	}()

	return nil
}

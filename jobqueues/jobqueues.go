/*
Copyright 2018 Graham Lee Bevan <graham.bevan@ntlworld.com>

This file is part of goswim.

goswim is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

goswim is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Foobar.  If not, see <https://www.gnu.org/licenses/>.
*/

package jobqueues

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	client "docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/container"
	"github.com/gbevan/goswim/approle"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/hashicorp/vault/api"
	. "github.com/visionmedia/go-debug" // nolint
	yaml "gopkg.in/yaml.v2"
)

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
	ID             bson.ObjectId `json:"_id"             bson:"_id,omitempty"`
	NodeUUID       string        `json:"node_uuid"       bson:"node_uuid"`
	Qname          string        `json:"qname"           bson:"qname"`
	ContainerImage string        `json:"container_image" bson:"container_image"`
	Content        string        `json:"content"         bson:"content"`
	EntryPoint     []string      `json:"entrypoint"      bson:"entrypoint"`
	Run            []string      `json:"run"             bson:"run"`
	Status         string        `json:"status"          bson:"status"`
	ReturnCode     int           `json:"return_code"     bson:"return_code"`
	Submitted      time.Time     `json:"submitted"       bson:"submitted"`
	Started        time.Time     `json:"started"         bson:"started,omitempty"`
	Ended          time.Time     `json:"ended"           bson:"ended,omitempty"`
	Output         string        `json:"output"          bson:"output"`
	SecretID       string        `json:"secret_id"       bson:"secret_id"`
	SecretRefs     []string      `json:"secret_refs"     bson:"secret_refs"`
	ContOnWarnings bool          `json:"cont_on_warnings" bson:"cont_on_warnings"`
	contentRdr     io.Reader
	secretsRdr     io.Reader
	// ReturnCode     int           `json:"return_code"     bson:"return_code,omitempty"`
	// NOTE: ContainerImage: this may be passed in the content itself as meta data
}

func (job *Job) String() string {
	return fmt.Sprintf("ID: %s, Qname: %s, Submitted: %s, Status: %s, Started: %s", job.ID, job.Qname, job.Submitted, job.Status, job.Started)
}

// Init Initialises the job queues loop
func Init(db *mgo.Database, appRoleID string, nodeUuid string) {
	jobQueues.Db = db
	jobQueues.AppRoleID = appRoleID
	jobQueues.PulledImages = make(map[string]PulledImage)
	jobQueues.NodeUUID = nodeUuid
	// start go routine to loop on the queues collection for new work
	// Qname defines the FIFO queue.
	// Provide a Wake channel for immediate pull ???
	go requestHandler()
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

func (job *Job) updateQueue(u bson.M) (*Job, error) {
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
	token, client, err := approle.Authenticate(jobQueues.AppRoleID, job.SecretID)
	if err != nil {
		job.updateQueue(bson.M{
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

	// Allow SecretRefs to be passed in job, e.g.
	// SecretRefs: see tests/job1.json
	// briefly cache the path response to allow multiple values to be extracted
	// without needing to re-query the Vault.
	secRefRe, err := regexp.Compile("^(\\w+)@([\\w\\-_/]+)\\.(([\\w\\-_]+))$")
	if err != nil {
		job.updateQueue(bson.M{
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
		secVarName := parts[1]
		secPath := parts[2]
		secKey := parts[3]

		// var secretValues api.Secret
		secretValues := cache[secPath]
		if secretValues == nil {
			secretValues, err = client.Logical().Read(secPath)
			if err != nil {
				job.updateQueue(bson.M{
					"status": "failed",
					"ended":  time.Now(),
					"output": fmt.Sprintf("Failed to retrieve secret %s from vault: %s", secPath, err),
				})
				return
			}

			if !job.ContOnWarnings && len(secretValues.Warnings) > 0 {
				job.updateQueue(bson.M{
					"status": "failed",
					"ended":  time.Now(),
					"output": fmt.Sprintf("FailOnWarnings from vault path %s lookups: %v", secPath, secretValues.Warnings),
				})
				return
			}

			cache[secPath] = secretValues
		}
		secrets[secVarName] = (secretValues.Data["data"].(map[string]interface{}))[secKey].(string)
	}

	// Create tar rdr for /secrets.yml in container
	secretsYaml, err := yaml.Marshal(secrets)
	if err != nil {
		job.updateQueue(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Failed to Marshal secrets to yaml for container injection: %s", err),
		})
		return
	}
	yamlHdr := []byte("---\n# goswim vault secrets injected:\n")
	secretsYaml = append(yamlHdr, secretsYaml...)
	var buf bytes.Buffer
	wtr := tar.NewWriter(&buf)
	hdr := &tar.Header{
		Name: "secrets.yml",
		Mode: 0444,
		Size: int64(len(secretsYaml)),
	}
	if err = wtr.WriteHeader(hdr); err != nil {
		job.updateQueue(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Failed to write secrets.yaml to tar header for container injection: %s", err),
		})
		return
	}
	if _, err = wtr.Write(secretsYaml); err != nil {
		job.updateQueue(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Failed to write secrets.yaml data to tar for container injection: %s", err),
		})
		return
	}
	if err = wtr.Close(); err != nil {
		job.updateQueue(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Failed to close tar of secrets.yaml for container injection: %s", err),
		})
		return
	}
	job.secretsRdr = strings.NewReader(buf.String())

	// Handle Content
	if job.Content != "" {
		parts := strings.Split(job.Content, ",")
		if len(parts) != 2 {
			job.updateQueue(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Failed to parse invalid content"),
			})
			return
		}
		// decode content base64
		data, err2 := base64.StdEncoding.DecodeString(parts[1])
		if err2 != nil {
			job.updateQueue(bson.M{
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
				job.updateQueue(bson.M{
					"status": "failed",
					"ended":  time.Now(),
					"output": fmt.Sprintf("Failed to unzip content: %s", err),
				})
				return
			}
			break
		default:
			job.updateQueue(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Failed to extract content, unsupported archive format: %s", parts[0]),
			})
			return
		}

		// Look for goswim.yml in content
		// fmt.Printf("tar data: %v", string(data))
		meta := Meta{}
		tempRdr, err := gzip.NewReader(strings.NewReader(string(data)))
		if err != nil {
			job.updateQueue(bson.M{
				"status": "failed",
				"ended":  time.Now(),
				"output": fmt.Sprintf("Failed to unzip content to rdr for goswim.yaml: %s", err),
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
				job.updateQueue(bson.M{
					"status": "failed",
					"ended":  time.Now(),
					"output": fmt.Sprintf("Failed content tar: %s", err2),
				})
				return
			}
			if hdr.Name == "./goswim.yml" {
				if _, err = io.Copy(&bufMeta, tr); err != nil {
					job.updateQueue(bson.M{
						"status": "failed",
						"ended":  time.Now(),
						"output": fmt.Sprintf("Failed extracting goswim.yml from tar: %s", err),
					})
					return
				}
			}

			// fmt.Printf("bufMeta: %s\n", bufMeta.Bytes())
			// parse goswim.yml
			err := yaml.Unmarshal(bufMeta.Bytes(), &meta)
			if err != nil {
				job.updateQueue(bson.M{
					"status": "failed",
					"ended":  time.Now(),
					"output": fmt.Sprintf("Failed parsing yaml in goswim.yml: %s", err),
				})
				return
			}
		} // for

		if job.ContainerImage == "" {
			job.ContainerImage = meta.ContainerImage
		}
	}

	err = job.runContainer()
	if err != nil {
		job.updateQueue(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Run container failed: %s", err),
		})
		return
	}
}

// Meta defines the format of the goswim.yml file
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
		if img.RepoTags[0] == job.ContainerImage {
			imgAlreadyPulled = true
		}
	}

	var imgAgeDays time.Duration
	if imgAlreadyPulled {
		imgAgeDays = (time.Now().Sub(jobQueues.PulledImages[job.ContainerImage].When)) / (time.Hour * 24)
	}

	if !imgAlreadyPulled || imgAgeDays > 1 {
		// reader, err := cli.ImagePull(ctx, "docker.io/library/busybox", types.ImagePullOptions{})
		_, err2 := cli.ImagePull(ctx, imgRef, types.ImagePullOptions{})
		if err2 != nil {
			log.Printf("ImagePull imgRef: %s", imgRef)
			return err2
		}
		// io.Copy(os.Stdout, reader)

		jobQueues.PulledImages[job.ContainerImage] = PulledImage{When: time.Now()}
	}
	// } else {
	// 	log.Printf("Image %s already pulled, age: %d", job.ContainerImage, imgAgeDays)

	cfg := container.Config{
		Image: job.ContainerImage,
		// Cmd:   []string{"echo", "hello world"},
		Cmd: job.Run,
		Tty: true,
		// User: "1000:1000",
	}

	if len(job.EntryPoint) != 0 {
		cfg.Entrypoint = job.EntryPoint
	}

	resp, err := cli.ContainerCreate(ctx, &cfg, nil, nil, "")
	if err != nil {
		log.Printf("ContainerCreate cfg: %v", cfg)
		return err
	}

	// Copy content into container prior to start it
	opts := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
		// CopyUIDGID:                true,
	}
	err = cli.CopyToContainer(ctx, resp.ID, "/", job.contentRdr, opts)
	if err != nil {
		return err
	}

	// Copy secrets into container prior to start it
	err = cli.CopyToContainer(ctx, resp.ID, "/", job.secretsRdr, opts)
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
	// log.Printf("status from container wait: %d", status)

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

	job.updateQueue(bson.M{
		"status":      finalStatus,
		"ended":       time.Now(),
		"output":      buf.String(),
		"return_code": status,
	})

	rmOpts := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         true,
	}
	if err := cli.ContainerRemove(ctx, resp.ID, rmOpts); err != nil {
		log.Printf("Error: removing container: %s", err)
	}

	return nil
}

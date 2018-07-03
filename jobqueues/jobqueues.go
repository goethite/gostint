package jobqueues

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/hashicorp/vault/api"
	. "github.com/visionmedia/go-debug" // nolint
)

var debug = Debug("jobqueues")

type JobQueues struct {
	Db        *mgo.Database
	AppRoleID string
}

var jobQueues JobQueues

// Job structure to represent a job submission request
type Job struct {
	ID             bson.ObjectId `json:"_id"             bson:"_id,omitempty"`
	Qname          string        `json:"qname"           bson:"qname"`
	ContainerImage string        `json:"container_image" bson:"container_image"`
	Content        string        `json:"content"         bson:"content"`
	Run            []string      `json:"run"             bson:"run"`
	Status         string        `json:"status"          bson:"status"`
	Submitted      time.Time     `json:"submitted"       bson:"submitted"`
	Started        time.Time     `json:"started"         bson:"started,omitempty"`
	Ended          time.Time     `json:"ended"           bson:"ended,omitempty"`
	Output         []string      `json:"log_lines"       bson:"log_lines"`
	SecretID       string        `json:"secret_id"       bson:"secret_id"`
	SecretRefs     []string      `json:"secret_refs"     bson:"secret_refs"`
	ContOnWarnings bool          `json:"cont_on_warnings" bson:"cont_on_warnings"`
	// ReturnCode     int           `json:"return_code"     bson:"return_code,omitempty"`
	// NOTE: ContainerImage: this may be passed in the content itself as meta data
}

func (job *Job) String() string {
	return fmt.Sprintf("ID: %s, Qname: %s, Submitted: %s, Status: %s, Started: %s", job.ID, job.Qname, job.Submitted, job.Status, job.Started)
}

// Init Initialises the job queues loop
func Init(db *mgo.Database, appRoleID string) {
	jobQueues.Db = db
	jobQueues.AppRoleID = appRoleID
	// start go routine to loop on the queues collection for new work
	// Qname defines the FIFO queue.
	// Provide a Wake channel for immediate pull ???
	go requestHandler()
}

func requestHandler() {
	log.Println("Starting Request Handler")
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
			j := Job{}

			chg := mgo.Change{
				Update:    bson.M{"$set": bson.M{"status": "running", "started": time.Now()}},
				ReturnNew: false,
			}

			statusCond := []bson.M{}
			statusCond = append(statusCond, bson.M{"status": "queued"})
			statusCond = append(statusCond, bson.M{"status": "running"})

			_, err := c.Find(bson.M{"qname": q, "$or": statusCond}).Sort("submitted").Limit(1).Apply(chg, &j)
			if err != nil {
				if err.Error() == "not found" {
					continue
				}
				log.Printf("Error: Pop from queue %s failed: %v\n", q, err)
			}
			// log.Printf("ci: %v", ci)

			// NOTE: if the returned job has status = "queued", then this has just
			// been atomically pop'd from the FIFO stack
			if j.Status != "queued" {
				break
			}
			// log.Printf("j: %s", j.String())

			go runRequest(&j)
		}

		time.Sleep(1000 * time.Millisecond)
	}
}

func (j *Job) updateQueue(u bson.M) (*Job, error) {
	db := jobQueues.Db
	c := db.C("queues")

	chg := mgo.Change{
		Update:    u,
		ReturnNew: true,
	}

	_, err := c.FindId(j.ID).Apply(chg, j)
	if err != nil {
		log.Printf("Error: update queue failed: %s\n", err)
		return nil, err
	}
	return j, nil
}

func runRequest(j *Job) {
	log.Printf("Run %s", (*j).String())

	/////////////////////////////////////
	// AppRole Authenticate
	// Get Token for passed secret_id
	if j.SecretID == "" {
		j.updateQueue(bson.M{
			"status": "notauthorised",
			"ended":  time.Now(),
			"output": "Vault SecretID was not provided in request",
		})
		return
	}
	appRoleID := jobQueues.AppRoleID

	client, err := api.NewClient(&api.Config{
		Address: os.Getenv("VAULT_ADDR"),
	})
	if err != nil {
		j.updateQueue(bson.M{
			"status": "notauthorised",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Failed create vault client api: %s", err),
		})
		return
	}

	// Authenticate this request using AppRole RoleID and SecretID
	data := map[string]interface{}{
		"role_id":   appRoleID,
		"secret_id": j.SecretID,
	}
	resp, err := client.Logical().Write("auth/approle/login", data)
	if err != nil {
		j.updateQueue(bson.M{
			"status": "notauthorised",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Request failed AppRole authentication with vault: %s", err),
		})
		return
	}
	if resp.Auth == nil {
		j.updateQueue(bson.M{
			"status": "notauthorised",
			"ended":  time.Now(),
			"output": "Request's Vault AppRole authentication returned no Auth token",
		})
		return
	}
	token := (*resp.Auth).ClientToken
	log.Printf("token: %s", token)

	// Authenticate with Vault using newly acquired token
	client.SetToken(token)

	secretValues, err := client.Logical().Read("secret/data/my-secret")
	if err != nil {
		j.updateQueue(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Failed to retrieve secret from vault: %s", err),
		})
		return
	}
	for k, v := range secretValues.Data {
		log.Printf("secretValues Data: %s: %v", k, v)
	}
	log.Printf("secretValues Data.data: %v", secretValues.Data["data"])
	for k, v := range secretValues.Data["data"].(map[string]interface{}) {
		log.Printf("secretValues Data.data: %s: %v", k, v)
	}
	for _, w := range secretValues.Warnings {
		log.Printf("secretValues Warning: %s", w)
	}

	if !j.ContOnWarnings && len(secretValues.Warnings) > 0 {
		j.updateQueue(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("FailOnWarnings from vault path lookups: %v", secretValues.Warnings),
		})
		return
	}

	// TODO: Allow SecretRefs to be passed in job, e.g.
	// SecretRefs: [
	//   "variable_1:[secret/data/my-secret].my-value",
	//   "variable_2:[secret/data/my-secret].my-value-2"
	// ]
	// briefly cache the path response to allow multiple values to be extracted
	// without needing to re-query the Vault.

	// TODO: Options for how secretrefs will be passed to the container, e.g.:
	// As "envars", "volume", "args", ...

	////////////////////////////////////
	// Run job in requested container
	dockerCmd := "docker"
	dockerArgs := []string{"run", "--rm", j.ContainerImage}
	dockerArgs = append(dockerArgs, j.Run...)
	cmd := exec.Command(dockerCmd, dockerArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error: execute job container failed: %s\n", err)
		j.updateQueue(bson.M{"$set": bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("%s\n%s", output, err.Error()),
		}})
		return
	}

	j.updateQueue(bson.M{"$set": bson.M{
		"status": "success",
		"ended":  time.Now(),
		"output": fmt.Sprintf("%s", output),
	}})

	// log.Printf("After %s", (*j).String())
}

package jobqueues

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
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
	Output         string        `json:"log_lines"       bson:"log_lines"`
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

	defer func() {
		// Revoke the ephemeral token
		resp, err = client.Logical().Write("auth/token/revoke-self", nil)
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
		j.updateQueue(bson.M{
			"status": "failed",
			"ended":  time.Now(),
			"output": fmt.Sprintf("Regex compilation error: %s", err),
		})
		return
	}
	log.Printf("SecretRefs: %v", j.SecretRefs)
	secrets := map[string]string{}
	secrets["TOKEN"] = token
	cache := map[string]*api.Secret{}
	for _, v := range j.SecretRefs {
		log.Printf("SecretRef: %s", v)

		parts := secRefRe.FindStringSubmatch(v)
		log.Printf("parts: %v", parts)
		secVarName := parts[1]
		secPath := parts[2]
		secKey := parts[3]

		// var secretValues api.Secret
		secretValues := cache[secPath]
		if secretValues == nil {
			secretValues, err = client.Logical().Read(secPath)
			if err != nil {
				j.updateQueue(bson.M{
					"status": "failed",
					"ended":  time.Now(),
					"output": fmt.Sprintf("Failed to retrieve secret %s from vault: %s", secPath, err),
				})
				return
			}

			if !j.ContOnWarnings && len(secretValues.Warnings) > 0 {
				j.updateQueue(bson.M{
					"status": "failed",
					"ended":  time.Now(),
					"output": fmt.Sprintf("FailOnWarnings from vault path %s lookups: %v", secPath, secretValues.Warnings),
				})
				return
			}

			cache[secPath] = secretValues
		}
		log.Printf("data: %v", secretValues)
		secrets[secVarName] = (secretValues.Data["data"].(map[string]interface{}))[secKey].(string)
	}
	log.Printf("secrets: %v", secrets)

	// TODO: Options for how secretrefs will be passed to the container, e.g.:
	// As "envars", "volume", "args", ...

	envs := []string{}
	eArgs := []string{}
	for k, v := range secrets {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))

		// build -e args with just keys, not values, as those are passed in cmd.Env
		eArgs = append(eArgs, "-e")
		eArgs = append(eArgs, k)
	}

	////////////////////////////////////
	// Run job in requested container
	dockerCmd := "docker"
	dockerArgs := []string{"run", "--rm"}
	dockerArgs = append(dockerArgs, eArgs...)
	dockerArgs = append(dockerArgs, j.ContainerImage)
	dockerArgs = append(dockerArgs, j.Run...)
	log.Printf("args: %v", dockerArgs)
	cmd := exec.Command(dockerCmd, dockerArgs...)

	cmd.Env = envs // secrets passed as env vars
	log.Printf("cmd Env: %v", cmd.Env)

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
	fmt.Println(string(output))
	j.updateQueue(bson.M{"$set": bson.M{
		"status": "success",
		"ended":  time.Now(),
		"output": fmt.Sprintf("%s", output),
	}})
}

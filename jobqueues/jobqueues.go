package jobqueues

import (
	"fmt"
	"log"
	"time"

	fifo "github.com/foize/go.fifo"
	. "github.com/visionmedia/go-debug" // nolint
)

var debug = Debug("jobqueues")

// Job structure to represent a job submission request
type Job struct {
	ID      string `json:"id"`
	Qname   string `json:"qname"`
	JobType string `json:"jobtype"`
	Content string `json:"content"`
	Run     string `json:"run"`
	Status  string `json:"status"`
}

// track jobs by uuid
var jobTracker = make(map[string]*Job)

func AddJobTracker(job *Job) {
	jobTracker[job.ID] = job
}

// JobQueue type to embed fifo.Queue
type JobQueue struct {
	Name string
	Q    *fifo.Queue
	Wake chan bool
}

var jobQueues = make(map[string]JobQueue)

func (jq *JobQueue) String() string {
	return fmt.Sprintf("Job Queue: %s length: %d", jq.Name, jq.Q.Len())
}

// Init Initialises the job queues
func Init() {
	jobQueues["play"] = JobQueue{
		Name: "play",
		Q:    fifo.NewQueue(),
		Wake: make(chan bool),
	}

	// Start JobQueue workers
	for name, jq := range jobQueues {
		go eatJobs(name, jq)
	}
}

func eatJobs(name string, jq JobQueue) {
	log.Printf("job queue: %s %v\n", name, jq)

	for {
		select {
		case <-jq.Wake:
			break
		case <-time.After(1000 * time.Millisecond):
			break
		}
		nxt := jq.Q.Next()
		// log.Printf("nxt: %v\n", nxt)
		if nxt != nil {
			log.Printf("eatJobs for queue %s received work: %v\n", name, nxt)
		}
	}
}

// GetQueue returns the named queue
func GetQueue(name string) JobQueue {
	return jobQueues[name]
}

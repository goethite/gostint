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

package job

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gbevan/gostint/apierrors"
	"github.com/gbevan/gostint/authenticate"
	"github.com/gbevan/gostint/jobqueues"
	"github.com/gbevan/gostint/logmsg"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/hashicorp/vault/api"
)

const notfound = "not found"

// JobRouter holds config state, e.g. the handle for the database
type JobRouter struct { // nolint
	Db *mgo.Database
}

var (
	jobRouter JobRouter

	// jobDuration = promauto.NewHistogramVec(
	// 	prometheus.HistogramOpts{
	// 		Name: "gostint_job_request_duration_seconds",
	// 		Help: "gostint histogram of job api request times in seconds.",
	// 	},
	// 	[]string{
	// 		"path",
	// 	},
	// )
)

// JobRequest localises jobqueues.Job in this module
type JobRequest jobqueues.Job // nolint

// Bind Binder of decoded request payload
func (j *JobRequest) Bind(req *http.Request) error {
	j.Qname = strings.ToLower(j.Qname)
	j.Status = "queued"
	j.Submitted = time.Now()

	return nil
}

// Routes Route handlers for jobs
func Routes(db *mgo.Database) *chi.Mux {
	jobRouter = JobRouter{
		Db: db,
	}
	router := chi.NewRouter()

	router.Use(
		authenticate.Authenticate,
	)

	router.Post("/", postJob)
	router.Post("/kill/{jobID}", killJob)
	router.Get("/{jobID}", getJob)
	router.Get("/", listJobs)
	router.Delete("/{jobID}", deleteJob)

	return router
}

type getResponse struct {
	ID             string    `json:"_id"`
	Status         string    `json:"status"`
	NodeUUID       string    `json:"node_uuid"`
	Qname          string    `json:"qname"`
	ContainerImage string    `json:"container_image"`
	Submitted      time.Time `json:"submitted"`
	Started        time.Time `json:"started"`
	Ended          time.Time `json:"ended"`
	Output         string    `json:"output"`
	Stderr         string    `json:"stderr"`
	ReturnCode     int       `json:"return_code"`
	Tty            bool      `json:"tty"`
}

// // AuthCtxKey context key for authentication state & policy map
// type AuthCtxKey string

type listResponse struct {
	Data  []getResponse `json:"data"`
	Skip  int           `json:"skip"`
	Limit int           `json:"limit"`
	Total int           `json:"total"`
}

func listJobs(w http.ResponseWriter, req *http.Request) {
	// timer := prometheus.NewTimer(jobDuration.WithLabelValues("listJobs"))
	// defer timer.ObserveDuration()
	// Parse URL params
	err := req.ParseForm()
	if err != nil {
		render.Render(w, req, apierrors.ErrInternalError(err))
		return
	}

	skip := 0
	skipParam := req.FormValue("skip") //.(string)
	if v, err2 := strconv.Atoi(skipParam); err2 == nil {
		skip = v
	}

	limit := 10
	coll := jobRouter.Db.C("queues")
	count, err := coll.Find(bson.M{}).Count()
	if err != nil {
		render.Render(w, req, apierrors.ErrInternalError(err))
		return
	}

	var jobs []JobRequest
	err = coll.Find(bson.M{}).Sort("-submitted").Skip(skip).Limit(limit).All(&jobs)
	if err != nil {
		// if err.Error() == notfound {
		// 	render.Render(w, req, apierrors.ErrNotFound(err))
		// 	return
		// }
		render.Render(w, req, apierrors.ErrInternalError(err))
		return
	}
	resp := []getResponse{}
	for _, job := range jobs {
		resp = append(resp, getResponse{
			ID:             job.ID.Hex(),
			Status:         job.Status,
			NodeUUID:       job.NodeUUID,
			Qname:          job.Qname,
			ContainerImage: job.ContainerImage,
			Submitted:      job.Submitted,
			Started:        job.Started,
			Ended:          job.Ended,
			Output:         job.Output,
			Stderr:         job.Stderr,
			ReturnCode:     job.ReturnCode,
			Tty:            job.Tty,
		})
	}
	paginateResp := listResponse{
		Data:  resp,
		Skip:  skip,
		Limit: limit,
		Total: count,
	}
	render.JSON(w, req, paginateResp)
}

func getJob(w http.ResponseWriter, req *http.Request) {
	// timer := prometheus.NewTimer(jobDuration.WithLabelValues("getJob"))
	// defer timer.ObserveDuration()

	jobID := strings.TrimSpace(chi.URLParam(req, "jobID"))
	if jobID == "" {
		render.Render(w, req, apierrors.ErrInvalidJobRequest(errors.New("job ID missing from GET path")))
		return
	}
	if !bson.IsObjectIdHex(jobID) {
		render.Render(w, req, apierrors.ErrInvalidJobRequest(errors.New("Invalid job ID (not ObjectIdHex)")))
		return
	}
	coll := jobRouter.Db.C("queues")
	var job JobRequest
	err := coll.FindId(bson.ObjectIdHex(jobID)).One(&job)
	if err != nil {
		if err.Error() == notfound {
			render.Render(w, req, apierrors.ErrNotFound(err))
			return
		}
		render.Render(w, req, apierrors.ErrInternalError(err))
		return
	}
	logmsg.Warn("Tty:", job.Tty)
	render.JSON(w, req, getResponse{
		ID:             job.ID.Hex(),
		Status:         job.Status,
		NodeUUID:       job.NodeUUID,
		Qname:          job.Qname,
		ContainerImage: job.ContainerImage,
		Submitted:      job.Submitted,
		Started:        job.Started,
		Ended:          job.Ended,
		Output:         job.Output,
		Stderr:         job.Stderr,
		ReturnCode:     job.ReturnCode,
		Tty:            job.Tty,
	})
}

type deleteResponse struct {
	ID string `json:"_id"`
}

func deleteJob(w http.ResponseWriter, req *http.Request) {
	// timer := prometheus.NewTimer(jobDuration.WithLabelValues("deleteJob"))
	// defer timer.ObserveDuration()

	jobID := strings.TrimSpace(chi.URLParam(req, "jobID"))
	if jobID == "" {
		render.Render(w, req, apierrors.ErrInvalidJobRequest(errors.New("job ID missing from GET path")))
		return
	}
	if !bson.IsObjectIdHex(jobID) {
		render.Render(w, req, apierrors.ErrInvalidJobRequest(errors.New("Invalid job ID (not ObjectIdHex)")))
		return
	}
	coll := jobRouter.Db.C("queues")

	// Get status and ensure job is not running/stopping
	// TODO: Look at making the find-and-remove atomic
	var job jobqueues.Job
	err := coll.FindId(bson.ObjectIdHex(jobID)).One(&job)
	if err != nil {
		if err.Error() == notfound {
			render.Render(w, req, apierrors.ErrNotFound(err))
			return
		}
		render.Render(w, req, apierrors.ErrInternalError(err))
		return
	}

	if job.Status == "running" || job.Status == "stopping" {
		render.Render(w, req, apierrors.ErrInvalidJobRequest(errors.New("Cannot delete a running/stopping job")))
		return
	}

	err = coll.RemoveId(bson.ObjectIdHex(jobID))
	if err != nil {
		if err.Error() == notfound {
			render.Render(w, req, apierrors.ErrNotFound(err))
			return
		}
		render.Render(w, req, apierrors.ErrInternalError(err))
		return
	}
	render.JSON(w, req, deleteResponse{
		ID: jobID,
	})
}

type postResponse struct {
	ID     string `json:"_id"`
	Status string `json:"status"`
	Qname  string `json:"qname"`
}

// postJob post a job to the fifo queue
// curl http://127.0.0.1:3232/v1/api/job -X POST -d '{"qname":"play", "jobtype": "ansible", "content": "base64 here", "run": "hello.yml"}'
// -> {"qname":"play","jobtype":"ansible","content":"base64 here","run":"hello.yml"}
func postJob(w http.ResponseWriter, req *http.Request) {
	// timer := prometheus.NewTimer(jobDuration.WithLabelValues("postJob"))
	// defer timer.ObserveDuration()

	data := &JobRequest{}
	if err := render.Bind(req, data); err != nil {
		render.Render(w, req, apierrors.ErrInvalidJobRequest(err))
		return
	}
	job := data

	coll := jobRouter.Db.C("queues")
	newID := bson.NewObjectId()
	jobRequest := job
	jobRequest.ID = newID

	if jobRequest.WrapSecretID == "" {
		render.Render(w, req, apierrors.ErrInvalidJobRequest(errors.New("AppRole SecretID's Wrapping Token must be present in the job request")))
		return
	}

	// Allow bypassing of cubbyhole, assuming unbroken TLS used for the request
	if job.CubbyToken != "" && job.CubbyPath != "" {
		// get encrypted payload from cubbyhole
		client, err := api.NewClient(&api.Config{
			Address: os.Getenv("VAULT_ADDR"),
		})
		if err != nil {
			render.Render(w, req, apierrors.ErrInternalError(fmt.Errorf("Failed create vault client api: %s", err)))
			return
		}
		client.SetToken(job.CubbyToken)
		resp, err := client.Logical().Read(job.CubbyPath)
		if err != nil {
			render.Render(w, req, apierrors.ErrInternalError(fmt.Errorf("POSSIBLE SECURITY/INTERCEPTION ALERT!!! Failed to read cubbyhole from vault, error: %s", err)))
			return
		}
		job.Payload = resp.Data["payload"].(string)
	}

	err := coll.Insert(jobRequest)
	if err != nil {
		panic(err)
	}

	render.JSON(w, req, postResponse{
		ID:     jobRequest.ID.Hex(),
		Status: jobRequest.Status,
		Qname:  jobRequest.Qname,
	})
}

type killResponse struct {
	ID            string `json:"_id"`
	ContainerID   string `json:"container_id"`
	Status        string `json:"status"`
	KillRequested bool   `json:"kill_requested"`
}

func killJob(w http.ResponseWriter, req *http.Request) {
	// timer := prometheus.NewTimer(jobDuration.WithLabelValues("killJob"))
	// defer timer.ObserveDuration()

	jobID := strings.TrimSpace(chi.URLParam(req, "jobID"))
	logmsg.Warn("killJob ID: %s", jobID)
	if jobID == "" {
		render.Render(w, req, apierrors.ErrInvalidJobRequest(errors.New("job ID missing from GET path")))
		return
	}
	if !bson.IsObjectIdHex(jobID) {
		render.Render(w, req, apierrors.ErrInvalidJobRequest(errors.New("Invalid job ID (not ObjectIdHex)")))
		return
	}
	coll := jobRouter.Db.C("queues")
	var job jobqueues.Job
	err := coll.FindId(bson.ObjectIdHex(jobID)).One(&job)
	if err != nil {
		if err.Error() == notfound {
			render.Render(w, req, apierrors.ErrNotFound(err))
			return
		}
		render.Render(w, req, apierrors.ErrInternalError(err))
		return
	}

	// flag job to be killed - we cant do this directly here because this
	// instance of gostint may not be the same one that is running the job
	job.UpdateJob(bson.M{
		"kill_requested": true,
	})

	render.JSON(w, req, killResponse{
		ID:            job.ID.Hex(),
		ContainerID:   job.ContainerID,
		Status:        job.Status,
		KillRequested: true,
	})
}

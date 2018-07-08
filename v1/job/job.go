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

package job

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gbevan/goswim/jobqueues"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

const notfound = "not found"

type JobRouter struct {
	Db *mgo.Database
}

var jobRouter JobRouter

type JobRequest jobqueues.Job

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
	router.Post("/", postJob)
	router.Post("/kill/{jobID}", killJob)
	router.Get("/{jobID}", getJob)
	router.Delete("/{jobID}", deleteJob)
	return router
}

type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	AppCode    int64  `json:"code,omitempty"`  // application-specific error code
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid job request.",
		ErrorText:      err.Error(),
	}
}

func ErrNotFound(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 404,
		StatusText:     "Not Found.",
		ErrorText:      err.Error(),
	}
}

func ErrInternalError(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 500,
		StatusText:     "Internal Error.",
		ErrorText:      err.Error(),
	}
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
	ReturnCode     int       `json:"return_code"`
}

func getJob(w http.ResponseWriter, req *http.Request) {
	jobID := strings.TrimSpace(chi.URLParam(req, "jobID"))
	if jobID == "" {
		render.Render(w, req, ErrInvalidRequest(errors.New("job ID missing from GET path")))
		return
	}
	if !bson.IsObjectIdHex(jobID) {
		render.Render(w, req, ErrInvalidRequest(errors.New("Invalid job ID (not ObjectIdHex)")))
		return
	}
	coll := jobRouter.Db.C("queues")
	var job JobRequest
	err := coll.FindId(bson.ObjectIdHex(jobID)).One(&job)
	if err != nil {
		if err.Error() == notfound {
			render.Render(w, req, ErrNotFound(err))
			return
		}
		render.Render(w, req, ErrInternalError(err))
		return
	}
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
		ReturnCode:     job.ReturnCode,
	})
}

type deleteResponse struct {
	ID string `json:"_id"`
}

func deleteJob(w http.ResponseWriter, req *http.Request) {
	jobID := strings.TrimSpace(chi.URLParam(req, "jobID"))
	if jobID == "" {
		render.Render(w, req, ErrInvalidRequest(errors.New("job ID missing from GET path")))
		return
	}
	if !bson.IsObjectIdHex(jobID) {
		render.Render(w, req, ErrInvalidRequest(errors.New("Invalid job ID (not ObjectIdHex)")))
		return
	}
	coll := jobRouter.Db.C("queues")

	// TODO: Get status and ensure job is not running/stopping

	err := coll.RemoveId(bson.ObjectIdHex(jobID))
	if err != nil {
		if err.Error() == notfound {
			render.Render(w, req, ErrNotFound(err))
			return
		}
		render.Render(w, req, ErrInternalError(err))
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
	data := &JobRequest{}
	if err := render.Bind(req, data); err != nil {
		render.Render(w, req, ErrInvalidRequest(err))
		return
	}
	job := data

	coll := jobRouter.Db.C("queues")
	newID := bson.NewObjectId()
	jobRequest := job
	jobRequest.ID = newID

	// if there isnt another SecretID in the job, then inject the one from the
	// initial request.  This could allow support for two levels of authentication
	// 1) the POSTer of the request, and 2) the originator of the job itself.
	// e.g. the original job request may have been created by some orchestration
	// tool and then passed to an intermediary, like a Lambda function, which then
	// POSTs the reques to goswim.
	if jobRequest.SecretID == "" {
		jobRequest.SecretID = req.Header["X-Secret-Token"][0]
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
	ID          string `json:"_id"`
	ContainerID string `json:"container_id"`
	Status      string `json:"status"`
}

func killJob(w http.ResponseWriter, req *http.Request) {
	jobID := strings.TrimSpace(chi.URLParam(req, "jobID"))
	log.Printf("killJob ID: %s", jobID)
	if jobID == "" {
		render.Render(w, req, ErrInvalidRequest(errors.New("job ID missing from GET path")))
		return
	}
	if !bson.IsObjectIdHex(jobID) {
		render.Render(w, req, ErrInvalidRequest(errors.New("Invalid job ID (not ObjectIdHex)")))
		return
	}
	coll := jobRouter.Db.C("queues")
	var job jobqueues.Job
	err := coll.FindId(bson.ObjectIdHex(jobID)).One(&job)
	if err != nil {
		if err.Error() == notfound {
			render.Render(w, req, ErrNotFound(err))
			return
		}
		render.Render(w, req, ErrInternalError(err))
		return
	}
	err = job.Kill()
	if err != nil {
		render.Render(w, req, ErrInternalError(err))
		return
	}

	render.JSON(w, req, killResponse{
		ID:          job.ID.Hex(),
		ContainerID: job.ContainerID,
		Status:      job.Status,
	})
}

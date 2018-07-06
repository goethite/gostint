package job

import (
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
	router.Post("/", PostJob)
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
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

// PostJob post a job to the fifo queue
// curl http://127.0.0.1:3232/v1/api/job -X POST -d '{"qname":"play", "jobtype": "ansible", "content": "base64 here", "run": "hello.yml"}'
// -> {"qname":"play","jobtype":"ansible","content":"base64 here","run":"hello.yml"}

func PostJob(w http.ResponseWriter, req *http.Request) {
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
	err := coll.Insert(jobRequest)
	if err != nil {
		panic(err)
	}
	// log.Printf("ci: %v\n", ci)
	log.Printf("PostJob() Inserted id: %s\n", newID)

	// queue := jobqueues.GetQueue("play")
	// queue.Q.Add(job)
	// queue.Wake <- true
	render.JSON(w, req, jobRequest)
}

package job

import (
	"net/http"
	"strings"

	"github.com/gbevan/goswim/jobqueues"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/satori/go.uuid"
)

type JobRequest struct {
	*jobqueues.Job
}

func (j *JobRequest) Bind(req *http.Request) error {
	// a.Job.Qname
	j.Job.Qname = strings.ToLower(j.Job.Qname)
	j.Job.ID = uuid.NewV4().String()
	j.Job.Status = "queued"

	jobqueues.AddJobTracker(j.Job)
	return nil
}

func Routes() *chi.Mux {
	router := chi.NewRouter()

	// router.Get("/", GetAllDocs)
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
// curl http://127.0.0.1:3232/v1/api/job \
//   -X POST \
//   -d '{"qname":"play", "jobtype": "ansible", "content": "base64 here", "run": "hello.yml"}'
// -> {"qname":"play","jobtype":"ansible","content":"base64 here","run":"hello.yml"}

func PostJob(w http.ResponseWriter, req *http.Request) {
	data := &JobRequest{}
	if err := render.Bind(req, data); err != nil {
		render.Render(w, req, ErrInvalidRequest(err))
		return
	}
	job := data.Job

	queue := jobqueues.GetQueue("play")
	queue.Q.Add(job)
	queue.Wake <- true
	render.JSON(w, req, job)
}

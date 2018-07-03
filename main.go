package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gbevan/goswim/jobqueues"
	"github.com/gbevan/goswim/v1/doc"
	"github.com/gbevan/goswim/v1/job"
	"github.com/globalsign/mgo"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/hashicorp/vault/api"
)

// MongoDB session and db
var dbSession *mgo.Session
var goswimDb *mgo.Database

var appRoleID string

// GetDbSession returns the MongoDB session
func GetDbSession() *mgo.Session {
	return dbSession
}

// GetDb returns the goswim Db
func GetDb() *mgo.Database {
	return goswimDb
}

func GetAppRoleID() string {
	return appRoleID
}

// getDbCreds() Get Ephemeral username & password from Vault using the
// One-Time (num_uses=2) token passed from provisioner (in dev see
// Gododir/main.go tasks "default" -> "gettoken").
func getDbCreds() (string, string, error) {
	// new Vault API Client
	client, err := api.NewClient(&api.Config{
		Address: os.Getenv("VAULT_ADDR"),
	})
	if err != nil {
		return "", "", err
	}

	// Authenticate with Vault using passed one-time token
	client.SetToken(os.Getenv("GOSWIM_DBAUTH_TOKEN"))
	os.Setenv("GOSWIM_DBAUTH_TOKEN", "")

	// Get MongoDB ephemeral credentials
	secretValues, err := client.Logical().Read("database/creds/goswim-dbauth-role")
	if err != nil {
		return "", "", err
	}

	username := ""
	password := ""
	for k, v := range secretValues.Data {
		switch k {
		case "username":
			username = v.(string)
			break
		case "password":
			password = v.(string)
			break
		}
	}

	return username, password, nil
}

func Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
		middleware.DefaultCompress,
		middleware.RedirectSlashes,
		middleware.Recoverer,
	)

	router.Route("/v1", func(r chi.Router) {
		r.Mount("/api/doc", doc.Routes())
		r.Mount("/api/job", job.Routes(GetDb()))
	})

	return router
}

func main() {
	username, password, err := getDbCreds()
	if err != nil {
		panic(err)
	}
	dbSession, err = mgo.Dial(os.Getenv("GOSWIM_DBURL"))
	if err != nil {
		panic(err)
	}
	goswimDb = dbSession.DB("goswim")
	err = goswimDb.Login(username, password)
	if err != nil {
		panic(err)
	}

	appRoleID = os.Getenv("GOSWIM_ROLEID")

	// Create RESTful routes
	router := Routes()

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		log.Printf("%s %s\n", method, route)
		return nil
	}

	if err := chi.Walk(router, walkFunc); err != nil {
		log.Panicf("Logging err: %s\n", err.Error())
	}

	// Start job queues
	jobqueues.Init(goswimDb, appRoleID)

	log.Fatal(http.ListenAndServe(":3232", router))
}

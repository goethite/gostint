package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gbevan/goswim/jobqueues"
	"github.com/gbevan/goswim/v1/doc"
	"github.com/gbevan/goswim/v1/job"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/hashicorp/vault/api"
	mgo "gopkg.in/mgo.v2"
)

// MongoDB session and db
var session *mgo.Session
var goswimDb *mgo.Database

type Test struct {
	Field1 string
}

// GetDbSession returns the MongoDB session
func GetDbSession() *mgo.Session {
	return session
}

// GetDb returns the goswim Db
func GetDb() *mgo.Database {
	return goswimDb
}

func getDbCreds() (string, string, error) {
	log.Printf("VAULT_ADDR=%s\n", os.Getenv("VAULT_ADDR"))
	log.Printf("GOSWIM_DBAUTH_TOKEN=%s\n", os.Getenv("GOSWIM_DBAUTH_TOKEN"))

	client, err := api.NewClient(&api.Config{
		Address: os.Getenv("VAULT_ADDR"),
	})
	if err != nil {
		return "", "", err
	}
	log.Println("after new client")

	client.SetToken(os.Getenv("GOSWIM_DBAUTH_TOKEN"))
	log.Println("after set token")

	secretValues, err := client.Logical().Read("database/creds/goswim-dbauth-role")
	if err != nil {
		return "", "", err
	}
	log.Printf("secretValues: %v\n", secretValues)

	username := ""
	password := ""
	for k, v := range secretValues.Data {
		log.Printf("vault %s: %v\n", k, v)
		switch k {
		case "username":
			username = v.(string)
			break
		case "password":
			password = v.(string)
			break
		}
	}
	log.Printf("Data: %v\n", secretValues.Data)

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
		r.Mount("/api/job", job.Routes())
	})

	return router
}

func main() {
	username, password, err := getDbCreds()
	if err != nil {
		panic(err)
	}
	session, err = mgo.Dial(os.Getenv("GOSWIM_DBURL"))
	if err != nil {
		panic(err)
	}
	log.Printf("sesson: %v\n", *session)
	goswimDb = session.DB("goswim")
	err = goswimDb.Login(username, password)
	if err != nil {
		panic(err)
	}
	log.Printf("goswimDb: %v\n", *goswimDb)

	coll := goswimDb.C("test")
	err = coll.Insert(&Test{"helloworld"})
	if err != nil {
		panic(err)
	}

	// Create RESTful routes
	router := Routes()

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		log.Printf("%s %s\n", method, route)
		return nil
	}

	log.Println("before walk")
	if err := chi.Walk(router, walkFunc); err != nil {
		log.Panicf("Logging err: %s\n", err.Error())
	}

	// Start job queues
	jobqueues.Init()

	log.Fatal(http.ListenAndServe(":3232", router))
}

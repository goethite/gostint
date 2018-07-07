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

package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/gbevan/goswim/approle"
	"github.com/gbevan/goswim/jobqueues"
	"github.com/gbevan/goswim/pingclean"
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

func authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Header["X-Secret-Token"]; !ok {
			log.Println("Missing X-Secret-Token")
			render.Render(w, r, ErrInvalidRequest(errors.New("Missing X-Secret-Token")))
			return
		}
		secretID := r.Header["X-Secret-Token"][0]
		unusedToken, client, err := approle.Authenticate(appRoleID, secretID)
		if err != nil {
			log.Printf("Authentication Failure with AppRole: %v", err)
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
		client.SetToken(unusedToken)
		// Revoke the unused token
		_, err = client.Logical().Write("auth/token/revoke-self", nil)
		if err != nil {
			log.Printf("Error: revoking token after job completed: %s", err)
		}

		ctx := context.WithValue(r.Context(), "authenticated", true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
		middleware.DefaultCompress,
		middleware.RedirectSlashes,
		middleware.Recoverer,
		authenticate,
	)

	router.Route("/v1", func(r chi.Router) {
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

	// init ping and clean
	nodeUuid := pingclean.Init(goswimDb)

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
	jobqueues.Init(goswimDb, appRoleID, nodeUuid)

	log.Fatal(http.ListenAndServe(":3232", router))
}

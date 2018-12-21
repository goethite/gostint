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

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"

	"github.com/gbevan/gostint/health"
	"github.com/gbevan/gostint/jobqueues"
	"github.com/gbevan/gostint/logmsg"
	"github.com/gbevan/gostint/pingclean"
	"github.com/gbevan/gostint/state"
	"github.com/gbevan/gostint/ui"
	"github.com/gbevan/gostint/v1/health"
	"github.com/gbevan/gostint/v1/job"
	"github.com/gbevan/gostint/v1/vault"
	"github.com/globalsign/mgo"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/hashicorp/vault/api"
)

//go:generate esc -o banner.go banner.txt
//go:generate esc -prefix "ui" -include "(^ui/index.html|^ui/css|^ui/js|^ui/dist|css/bootstrap.css)" -pkg ui -o ui/ui.go ui

// MongoDB session and db
var dbSession *mgo.Session
var gostintDb *mgo.Database

var appRoleID string

// GetDbSession returns the MongoDB session
func GetDbSession() *mgo.Session {
	return dbSession
}

// GetDb returns the gostint Db
func GetDb() *mgo.Database {
	return gostintDb
}

// GetAppRoleID returns the instance's App Role ID
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
	client.SetToken(os.Getenv("GOSTINT_DBAUTH_TOKEN"))
	os.Setenv("GOSTINT_DBAUTH_TOKEN", "")

	// Get MongoDB ephemeral credentials
	secretValues, err := client.Logical().Read("database/creds/gostint-dbauth-role")
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

// Routes defines RESTful api middleware and routes.
func Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
		middleware.DefaultCompress,
		middleware.RedirectSlashes,
		middleware.Recoverer,
		// authenticate,
	)

	router.Route("/v1", func(r chi.Router) {
		r.Mount("/api/job", job.Routes(GetDb()))
		r.Mount("/api/health", healthApi.Routes(GetDb()))
		r.Mount("/api/vault", vault.Routes())
	})

	router.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "../", 301)
	})

	if os.Getenv("GOSTINT_UI") == "1" {
		logmsg.Info("Enabling UI")
		// Note: http.FileServer will automatically resolve Content-Type headers
		router.Mount("/", http.FileServer(ui.FS(false)))
	}

	return router
}

func main() {
	if os.Getenv("GOSTINT_DEBUG") != "" {
		logmsg.EnableDebug()
	}

	banner, err := FSString(false, "/banner.txt")
	if err != nil {
		logmsg.Error("banner failed: %v", err)
	}
	fmt.Println(banner)

	serverPort := 3232
	if os.Getenv("GOSTINT_PORT") != "" {
		sp, err2 := strconv.Atoi(os.Getenv("GOSTINT_PORT"))
		if err2 != nil {
			panic(err2)
		}
		serverPort = sp
	}
	logmsg.Info("Compiled with: %v", runtime.Version())
	logmsg.Info("Starting gostint...")

	username, password, err := getDbCreds()
	if err != nil {
		panic(err)
	}
	logmsg.Debug("Dialing Mongodb")
	dbSession, err = mgo.Dial(os.Getenv("GOSTINT_DBURL"))
	if err != nil {
		panic(err)
	}
	logmsg.Debug("Logging in to gostint db")
	gostintDb = dbSession.DB("gostint")
	err = gostintDb.Login(username, password)
	if err != nil {
		panic(err)
	}

	// init ping and clean
	nodeUUID := pingclean.Init(gostintDb)

	appRole := jobqueues.AppRole{
		ID:   os.Getenv("GOSTINT_ROLEID"),
		Name: os.Getenv("GOSTINT_ROLENAME"),
	}

	// Create RESTful routes
	router := Routes()

	// initialise state
	state.Init(nodeUUID)

	// initialise health
	health.Init(gostintDb)

	// Start job queues
	jobqueues.Init(gostintDb, &appRole, nodeUUID)

	logmsg.Info("gostint listening on https port %d", serverPort)
	log.Fatal(http.ListenAndServeTLS(
		fmt.Sprintf(":%d", serverPort),
		os.Getenv("GOSTINT_SSL_CERT"),
		os.Getenv("GOSTINT_SSL_KEY"),
		router,
	))
}

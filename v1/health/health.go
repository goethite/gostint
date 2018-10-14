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

package healthApi

import (
	"errors"
	"net/http"

	"github.com/gbevan/gostint/apierrors"
	"github.com/gbevan/gostint/health"
	"github.com/globalsign/mgo"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

// HealthRouter holds config state, e.g. the handle for the database
type HealthRouter struct { // nolint
	Db *mgo.Database
}

var healthRouter HealthRouter

// Routes Route handler for health
func Routes(db *mgo.Database) *chi.Mux {
	healthRouter = HealthRouter{
		Db: db,
	}
	router := chi.NewRouter()
	router.Get("/", getHealth)
	return router
}

func getHealth(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	h, err := health.GetHealth()
	if err != nil {
		render.Render(w, req, apierrors.ErrInternalError(err))
		return
	}

	// request for individual value ?k=field
	if key, ok := req.Form["k"]; ok {
		if len(key) > 0 {
			if val, ok := (*h)[key[0]]; ok {
				render.PlainText(w, req, val)
				return
			}
		}
		render.Render(w, req, apierrors.ErrInvalidRequest(errors.New("Invalid health metric key")))
		return
	}

	// return all health metrics
	render.JSON(w, req, h)
}

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

package vault

import (
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	. "github.com/visionmedia/go-debug" // nolint
)

var debug = Debug("vault")

var vaultAddr string
var vaultExtAddr string

// Routes Route handler for vault
func Routes() *chi.Mux {
	vaultAddr = os.Getenv("VAULT_ADDR")
	vaultExtAddr = os.Getenv("VAULT_EXTERNAL_ADDR")

	if vaultExtAddr == "" {
		vaultExtAddr = vaultAddr
	}

	router := chi.NewRouter()
	router.Get("/info", getVault)
	return router
}

// Retrieve Gotstint's Vault info
func getVault(w http.ResponseWriter, req *http.Request) {
	m := make(map[string]string)
	m["vault_addr"] = vaultAddr
	m["vault_external_addr"] = vaultExtAddr
	render.JSON(w, req, m)
}

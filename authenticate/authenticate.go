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

package authenticate

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gbevan/gostint/apierrors"
	"github.com/gbevan/gostint/logmsg"
	"github.com/go-chi/render"
	"github.com/hashicorp/vault/api"
	. "github.com/visionmedia/go-debug" // nolint
)

var debug = Debug("authenticate")

// AuthStruct holds authenticated state and policy map from vault for the token
// placed in context
type AuthStruct struct {
	Authenticated bool
	PolicyMap     map[string]bool
}

// AuthCtxKey context key for authentication state & policy map
type AuthCtxKey string

// Authenticate caller's token with vault
func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow heath data without authenticating
		if r.Method == "GET" && r.URL.Path == "/v1/api/health" {
			ctx := context.WithValue(r.Context(), AuthCtxKey("auth"), nil)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if _, ok := r.Header["X-Auth-Token"]; !ok {
			logmsg.Error("Missing X-Auth-Token")
			render.Render(w, r, apierrors.ErrInvalidRequest(errors.New("Missing X-Auth-Token")))
			return
		}
		token := r.Header["X-Auth-Token"][0]
		// log.Printf("X-Auth-Token: %v", token)

		client, err := api.NewClient(&api.Config{
			Address: os.Getenv("VAULT_ADDR"),
		})
		if err != nil {
			errmsg := fmt.Sprintf("Failed create vault client api: %s", err)
			logmsg.Error(errmsg)
			render.Render(w, r, apierrors.ErrInvalidRequest(fmt.Errorf(errmsg)))
			return
		}

		client.SetToken(token)

		// Verify the token is good
		tokDetails, err := client.Logical().Read("auth/token/lookup-self")
		if err != nil {
			logmsg.Error("Authentication Failure with Token: %v", err)
			if strings.Contains(err.Error(), "Code: 403") {
				render.Render(w, r, apierrors.ErrPermissionDenied(err))
			} else {
				render.Render(w, r, apierrors.ErrInvalidRequest(err))
			}
			return
		}
		// tokJson, _ := json.Marshal(tokDetails)
		// log.Printf("tokDetails: %s", tokJson)

		authStruct := AuthStruct{
			Authenticated: true,
			PolicyMap:     map[string]bool{},
		}

		// log.Printf("Data policies: %v", tokDetails.Data["policies"])
		for _, p := range tokDetails.Data["policies"].([]interface{}) {
			authStruct.PolicyMap[p.(string)] = true
		}

		ctx := context.WithValue(r.Context(), AuthCtxKey("auth"), authStruct)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

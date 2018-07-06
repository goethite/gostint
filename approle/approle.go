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

package approle

import (
	"fmt"
	"os"

	"github.com/hashicorp/vault/api"
)

// Authenticate using our AppRoleID and given SecretID with Vault
func Authenticate(appRoleID string, secretID string) (string, *api.Client, error) {
	/////////////////////////////////////
	// AppRole Authenticate
	// Get Token for passed secret_id
	if secretID == "" {
		return "", &api.Client{}, fmt.Errorf("Vault SecretID was not provided in request")
	}

	client, err := api.NewClient(&api.Config{
		Address: os.Getenv("VAULT_ADDR"),
	})
	if err != nil {
		return "", &api.Client{}, fmt.Errorf("Failed create vault client api: %s", err)
	}

	// Authenticate this request using AppRole RoleID and SecretID
	data := map[string]interface{}{
		"role_id":   appRoleID,
		"secret_id": secretID,
	}
	resp, err := client.Logical().Write("auth/approle/login", data)
	if err != nil {
		return "", &api.Client{}, fmt.Errorf("Request failed AppRole authentication with vault: %s", err)
	}
	if resp.Auth == nil {
		return "", &api.Client{}, fmt.Errorf("Request's Vault AppRole authentication returned no Auth token")
	}
	token := (*resp.Auth).ClientToken
	return token, client, nil
}

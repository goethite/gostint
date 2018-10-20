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

package approle

import (
	"fmt"
	"os"

	"github.com/hashicorp/vault/api"
)

// Authenticate using our AppRoleID and given SecretID with Vault
func Authenticate(appRoleID string, wrapSecretID string) (string, *api.Client, error) {
	/////////////////////////////////////
	// AppRole Authenticate
	// Get Token for passed secret_id
	if wrapSecretID == "" {
		return "", &api.Client{}, fmt.Errorf("Vault SecretID Wrapping Token was not provided in request")
	}

	client, err := api.NewClient(&api.Config{
		Address: os.Getenv("VAULT_ADDR"),
	})
	if err != nil {
		return "", &api.Client{}, fmt.Errorf("Failed create vault client api: %s", err)
	}

	// Unwrap the wrapping token to get the SecretID
	secret, err := client.Logical().Unwrap(wrapSecretID)
	if err != nil {
		return "", &api.Client{}, fmt.Errorf("Request failed to unwrap the token to retrieve the SecretID - POSSIBLE SECURITY/INTERCEPTION ALERT!!!: THIS REQUEST MAY HAVE BEEN TAMPERED WITH, error: %s", err)
	}
	secretID := secret.Data["secret_id"]

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

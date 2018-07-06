package approle

import (
	"fmt"
	"os"

	"github.com/hashicorp/vault/api"
)

// // AuthError to return status and message
// type AuthError struct {
// 	Status string
// 	Msg    string
// }

// func (e AuthError) Error() string {
// 	return fmt.Sprintf("%s: %s", e.Status, e.Msg)
// }

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

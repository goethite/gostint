package main

// IMPORTANT: Use this fork of godo - go get -u github.com/gbevan/godo/cmd/godo

import (
	"strings"

	do "github.com/gbevan/godo"
)

func tasks(p *do.Project) {
	// do.Env = `GOPATH=.vendor::$GOPATH`
	token := ""

	p.Task("mocksecrets", nil, func(c *do.Context) {
		c.Bash(`
      vault login root
      vault kv put secret/my-secret my-value=s3cr3t
      vault kv put secret/my-form field1=value1 field2=value2 field3=value3
    `)
	})

	// this step gets a one-time (2 uses) token to allow goswim to get an
	// ephemeral user/password pair to authenticate with MongoDB
	p.Task("gettoken", do.S{"mocksecrets"}, func(c *do.Context) {
		token = c.BashOutput(`
    curl -s \
      --request POST \
      --header 'X-Vault-Token: root' \
      --data '{"policies": ["goswim-mongodb-auth"], "ttl": "10m", "num_uses": 2}' \
      ${VAULT_ADDR}/v1/auth/token/create | jq .auth.client_token -r
    `)
		token = strings.Trim(token, " \t\n")
	})

	p.Task("default", do.S{"gettoken"}, func(c *do.Context) {
		c.Start(`GOSWIM_SSL_CERT=etc/cert.pem GOSWIM_SSL_KEY=etc/key.pem GOSWIM_DBAUTH_TOKEN={{.token}} GOSWIM_DBURL=127.0.0.1:27017 main.go`, do.M{"token": token})
	}).Src("**/*.go")

	// To be run alongside default to drive BATS tests against the instance
	p.Task("test", nil, func(c *do.Context) {
		c.Bash("cd tests/bats && run-parts --regex=[0-9].* .")
	})
}

func main() {
	do.Godo(tasks)
}

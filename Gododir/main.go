package main

// IMPORTANT: Use this fork of godo - go get -u github.com/davars/godo/cmd/godo

import (
	"fmt"
	"strings"

	do "github.com/gbevan/godo"
)

func tasks(p *do.Project) {
	// do.Env = `GOPATH=.vendor::$GOPATH`
	// do.Env = `VAULT_ADDR=http://127.0.0.1:8200`
	token := ""

	p.Task("gettoken", nil, func(c *do.Context) {
		token = c.BashOutput(`
    curl -s \
      --request POST \
      --header 'X-Vault-Token: root' \
      --data '{"policies": ["goswim-mongodb-auth"], "ttl": "10m", "num_uses": 2}' \
      ${VAULT_ADDR}/v1/auth/token/create | jq .auth.client_token -r
    `)
		token = strings.Trim(token, " \t\n")
		fmt.Printf("token: %s\n", token)
		// do.Env += fmt.Sprintf("GOSWIM_DBAUTH_TOKEN=%s", token)
		// fmt.Printf("do.Env=%s\n", do.Env)
		// os.Setenv("GOSWIM_DBAUTH_TOKEN", token)
	})

	p.Task("default", do.S{"gettoken"}, func(c *do.Context) {
		// c.Start(`scripts/start_goswim.sh`)
		// do.InheritParentEnv = false
		c.Start(`GOSWIM_DBAUTH_TOKEN={{.token}} GOSWIM_DBURL=127.0.0.1:27017 main.go`, do.M{"token": token})
		// }).Src("main.go")
	}).Src("**/*.go")
}

func main() {
	do.Godo(tasks)
}

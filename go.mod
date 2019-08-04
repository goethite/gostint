module github.com/gbevan/gostint

go 1.12

require (
	docker.io/go-docker v1.0.0
	github.com/MichaelTJones/walk v0.0.0-20161122175330-4748e29d5718 // indirect
	github.com/Microsoft/go-winio v0.4.11 // indirect
	github.com/avast/retry-go v0.0.0-20180502193734-611bd93c6d74
	github.com/docker/distribution v0.0.0-20170726174610-edc3ab29cdff // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.3.0 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/fatih/color v1.7.0
	github.com/gbevan/godo v2.1.3+incompatible
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/go-chi/render v1.0.1
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/hashicorp/vault/api v1.0.4
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/mattn/go-isatty v0.0.4 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mgutz/minimist v0.0.0-20151219120022-39eb8cf573ca // indirect
	github.com/mgutz/str v1.2.0 // indirect
	github.com/mgutz/to v1.0.0 // indirect
	github.com/nozzle/throttler v0.0.0-20180816223912-93e5576933fe // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/prometheus/client_golang v0.9.3-0.20190127221311-3c4408c8b829
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90 // indirect
	github.com/prometheus/procfs v0.0.0-20190328153300-af7bedc223fb // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/visionmedia/go-debug v0.0.0-20180109164601-bfacf9d8a444
	golang.org/x/crypto v0.0.0-20190325154230-a5d413f7728c // indirect
	golang.org/x/sys v0.0.0-20190523142557-0e01d883c5c5 // indirect
	golang.org/x/text v0.3.2 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/docker/docker => github.com/docker/engine v0.0.0-20180816081446-320063a2ad06

//replace docker.io/go-docker => github.com/docker/engine v0.0.0-20180816081446-320063a2ad06

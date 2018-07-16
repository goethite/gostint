##############################################
# Build:
#   docker build --squash -t goswim .
#
# Run:
#   docker run --name goswim -p 3333:3232 \
#     --privileged=true \
#     -v /srv/goswim/lib:/var/lib/goswim \
#     -e VAULT_ADDR="$VAULT_ADDR" \
#     -e GOSWIM_DBAUTH_TOKEN="$token" \
#     -e GOSWIM_ROLEID="$roleid" \
#     -e GOSWIM_DBURL="dbhost:27017"
#     goswim
#
# cert.pem and key.pm are needed in /srv/goswim/lib, these can be redirected to
# another location using GOSWIM_SSL_CERT and GOSWIM_SSL_KEY respectively.


##############################################
# Build stage
FROM golang:latest as builder
WORKDIR /go/src/github.com/gbevan/goswim

COPY main.go Gopkg* ./
COPY v1 ./v1/
COPY approle ./approle/
COPY pingclean ./pingclean/
COPY jobqueues ./jobqueues/

RUN \
  go get github.com/golang/dep/cmd/dep && \
  dep ensure -v --vendor-only && \
  CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o goswim .


##############################################
# Exe stage
FROM alpine

COPY --from=builder /go/src/github.com/gbevan/goswim/goswim /usr/bin

WORKDIR /app
COPY start-image.sh .

# apk add --no-cache docker jq curl openssl sudo && \
RUN \
  apk add --no-cache docker sudo && \
  adduser -S -D -H -G docker -h /app goswim && \
  mkdir -p /var/lib/goswim && \
  chown goswim /var/lib/goswim && \
  echo "goswim	ALL=(ALL:ALL) NOPASSWD: ALL" >> /etc/sudoers

# sudo priv is removed immediately after dockerd starts, see start-image.sh

USER goswim
ENTRYPOINT ["/app/start-image.sh"]

##############################################
# Build:
#   docker build -t gostint .
#
# Run:
#   docker run --name gostint -p 3333:3232 \
#     --privileged=true \
#     -v /srv/gostint/lib:/var/lib/gostint \
#     -e VAULT_ADDR="$VAULT_ADDR" \
#     -e GOSTINT_DBAUTH_TOKEN="$token" \
#     -e GOSTINT_ROLEID="$roleid" \
#     -e GOSTINT_ROLENAME="gostint-role" \
#     -e GOSTINT_DBURL="dbhost:27017"
#     gostint
#
# cert.pem and key.pm are needed in /srv/gostint/lib, these can be redirected to
# another location using GOSTINT_SSL_CERT and GOSTINT_SSL_KEY respectively.


##############################################
# Build stage
FROM golang:latest as builder
WORKDIR /go/src/github.com/gbevan/gostint

COPY main.go Gopkg* ./
COPY v1 ./v1/
COPY approle ./approle/
COPY pingclean ./pingclean/
COPY jobqueues ./jobqueues/
COPY apierrors ./apierrors/
COPY health ./health/
COPY cleanup ./cleanup/
COPY logmsg ./logmsg/

RUN \
  go get github.com/golang/dep/cmd/dep && \
  dep ensure -v --vendor-only && \
  CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o gostint .


##############################################
# Exe stage
FROM alpine
# FROM alpine:3.3

COPY --from=builder /go/src/github.com/gbevan/gostint/gostint /usr/bin

WORKDIR /app
COPY start-image.sh .

# apk add --no-cache docker jq curl openssl sudo && \
RUN \
  apk add --no-cache docker sudo curl && \
  adduser -S -D -H -G docker -h /app gostint && \
  mkdir -p /var/lib/gostint && \
  chown gostint /var/lib/gostint && \
  echo "gostint	ALL=(ALL:ALL) NOPASSWD: ALL" >> /etc/sudoers

# sudo priv is removed immediately after dockerd starts, see start-image.sh

USER gostint
ENTRYPOINT ["/app/start-image.sh"]

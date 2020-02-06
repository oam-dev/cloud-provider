ARG GO_VERSION=1.13

# Build the manager binary
FROM golang:${GO_VERSION} AS builder

RUN apt-get update
RUN apt-get install -y ca-certificates make git curl mercurial

ARG PACKAGE=github.com/oam-dev/cloud-provider/alibabacloud/ros
RUN mkdir -p /go/src/${PACKAGE}
WORKDIR /go/src/${PACKAGE}

ADD . /go/src/${PACKAGE}/
# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o ros ./cmd/ros/


# Copy the controller-manager into a thin image
FROM alpine:3.9.4
RUN apk add --update ca-certificates \
 && apk add --update -t deps curl jq iproute2 \
 && apk del --purge deps \
 && rm /var/cache/apk/*

WORKDIR /
COPY --from=builder /go/src/github.com/oam-dev/cloud-provider/alibabacloud/ros/ros /usr/bin/

ENTRYPOINT ["/usr/bin/ros"]

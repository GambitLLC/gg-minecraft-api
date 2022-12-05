# syntax=docker/dockerfile:1
FROM golang:1.19.0
ENV GO111MODULE=on

WORKDIR /go/gambitdev/

# Copy go.mod and go.sum first to download dependencies.
# Docker caching is [in]validated by input file changes, so if dependencies do not
# change previous image layers can be used.

COPY . .
COPY ./profile-scanner/go.mod ./profile-scanner/go.sum ./profile-scanner/

WORKDIR /go/gambitdev/profile-scanner/
RUN go mod download

RUN CGO_ENABLED=0 go build -o build/gg-profile-scanner

WORKDIR build/
ENTRYPOINT ["/go/gambitdev/profile-scanner/build/gg-profile-scanner"]
EXPOSE 8080
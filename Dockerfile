# syntax=docker/dockerfile:1
FROM golang:1.19.0
ENV GO111MODULE=on

WORKDIR /go/gambitdev/gg-minecraft-api

# Copy go.mod and go.sum first to download dependencies.
# Docker caching is [in]validated by input file changes, so if dependencies do not
# change previous image layers can be used.
COPY go.sum go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o build/gg-minecraft-api

WORKDIR build/
ENTRYPOINT ["/go/gambitdev/gg-minecraft-api/build/gg-minecraft-api"]
EXPOSE 8080
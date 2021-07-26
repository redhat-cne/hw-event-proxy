.PHONY: build

# Export GO111MODULE=on to enable project to be built from within GOPATH/src
export GO111MODULE=on
export CGO_ENABLED=1
export GOFLAGS=-mod=vendor
export COMMON_GO_ARGS=-race
export GOOS=linux

ifeq (,$(shell go env GOBIN))
  GOBIN=$(shell go env GOPATH)/bin
else
  GOBIN=$(shell go env GOBIN)
endif

deps-update:
	go mod tidy && \
	go mod vendor

build:
	go fmt ./...
	make lint
	go build -o ./build/hw-event-proxy
	go test ./...  -coverprofile=cover.out

build-only:
	go build -o ./build/hw-event-proxy

lint:
	golint -set_exit_status `go list ./... | grep -v vendor`
	golangci-lint run

run:
	go run main.go

test:
	go test ./...  -coverprofile=cover.out

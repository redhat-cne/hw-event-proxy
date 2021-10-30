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

build: build-only test
	go fmt ./...
	make lint

build-only:
	go build -o ./build/hw-event-proxy cmd/main.go

lint:
	golint -set_exit_status `go list ./... | grep -v vendor`
	golangci-lint run

run:
	go run cmd/main.go

test:
	go test ./...  -coverprofile=cover.out

# For GitHub Actions CI
gha: build-only

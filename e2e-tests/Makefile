.PHONY: build-docker

# Export GO111MODULE=on to enable project to be built from within GOPATH/src
export GO111MODULE=on
export CGO_ENABLED=0

ifeq (,$(shell go env GOBIN))
  GOBIN=$(shell go env GOPATH)/bin
else
  GOBIN=$(shell go env GOBIN)
endif

export COMMON_GO_ARGS=-race

build:
	go fmt ./...
	make lint
	go build -o ./build/redfish-event-test cmd/main.go

run:
	go run cmd/main.go

lint:
	golint `go list ./... | grep -v vendor`
	golangci-lint run

test:
	./scripts/test.sh

cleanup:
	kubectl delete -f ./manifests/redfish-event-test.yaml

.PHONY: build

# Current  version
VERSION ?=latest

# Default image tag
PROXY_IMG ?= quay.io/jzding/hw-event-proxy:$(VERSION)
SIDECAR_IMG ?= quay.io/jzding/cloud-event-proxy:$(VERSION)
CONSUMER_IMG ?= quay.io/jzding/cloud-native-event-consumer:$(VERSION)

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

build:test
	go fmt ./...
	make lint
	go build -o ./build/hw-event-proxy cmd/main.go

build-only:
	go build -o ./build/hw-event-proxy cmd/main.go

lint:
	golint -set_exit_status `go list ./... | grep -v vendor`
	golangci-lint run

run:
	go run cmd/main.go

test:
	go test ./...  -coverprofile=cover.out

# Deploy all in the configured Kubernetes cluster in ~/.kube/config
deploy-example:kustomize
	cd ../examples/manifests && $(KUSTOMIZE) edit set image hw-event-proxy=${PROXY_IMG} && cloud-event-proxy=${SIDECAR_IMG} && $(KUSTOMIZE) edit set image && $(KUSTOMIZE) edit set image  cloud-native-event-consumer=${CONSUMER_IMG}
	$(KUSTOMIZE) build ../examples/manifests | kubectl apply -f -

# Deploy all in the configured Kubernetes cluster in ~/.kube/config
undeploy-example:kustomize
	cd ../examples/manifests  && $(KUSTOMIZE) edit set image hw-event-proxy=${PROXY_IMG} &&  cloud-event-proxy=${SIDECAR_IMG} && $(KUSTOMIZE) edit set image  && $(KUSTOMIZE) edit set image  cloud-native-event-consumer=${CONSUMER_IMG}
	$(KUSTOMIZE) build ../examples/manifests | kubectl delete -f -

# For GitHub Actions CI
gha:
	go test ./...  -coverprofile=cover.out
.PHONY: test

# Current  version
VERSION ?=latest

# Default image tag
PROXY_IMG ?= quay.io/redhat_emp1/hw-event-proxy:$(VERSION)
SIDECAR_IMG ?= quay.io/redhat_emp1/cloud-event-proxy:$(VERSION)
CONSUMER_IMG ?= quay.io/redhat_emp1/cloud-native-event-consumer:$(VERSION)

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

kustomize:
ifeq (, $(shell which kustomize))
		@{ \
		set -e ;\
		KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
		cd $$KUSTOMIZE_GEN_TMP_DIR ;\
		# remove -mod=vendor flag to allow install\
		export GOFLAGS=;\
		go mod init tmp ;\
		go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
		rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
		}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# label a worker node as local
label-node:
	kubectl label --overwrite node $(shell kubectl get nodes -l node-role.kubernetes.io/worker="" | grep Ready | cut -f1 -d" " | head -1) app=local

# Deploy all in the configured Kubernetes cluster in ~/.kube/config
deploy-example:kustomize
	cd ./examples/manifests && $(KUSTOMIZE) edit set image hw-event-proxy=${PROXY_IMG} \
		&& $(KUSTOMIZE) edit set image cloud-event-proxy=${SIDECAR_IMG} \
		&& $(KUSTOMIZE) edit set image  cloud-native-event-consumer=${CONSUMER_IMG} \
		&& $(KUSTOMIZE) edit set replicas consumer=1
	$(KUSTOMIZE) build ./examples/manifests | kubectl apply -f -

# Deploy all in the configured Kubernetes cluster in ~/.kube/config
undeploy-example:kustomize
	cd ./examples/manifests && $(KUSTOMIZE) edit set image hw-event-proxy=${PROXY_IMG} \
		&& $(KUSTOMIZE) edit set image cloud-event-proxy=${SIDECAR_IMG} \
		&& $(KUSTOMIZE) edit set image  cloud-native-event-consumer=${CONSUMER_IMG}
	$(KUSTOMIZE) build ./examples/manifests | kubectl delete -f -

# Deploy with 20 consumers for performance testing
deploy-perf:kustomize
	cd ./examples/manifests && $(KUSTOMIZE) edit set image hw-event-proxy=${PROXY_IMG} \
		&& $(KUSTOMIZE) edit set image cloud-event-proxy=${SIDECAR_IMG} \
		&& $(KUSTOMIZE) edit set image  cloud-native-event-consumer=${CONSUMER_IMG} \
		&& $(KUSTOMIZE) edit set replicas consumer=20
	$(KUSTOMIZE) build ./examples/manifests | kubectl apply -f -

# Deploy all in the configured Kubernetes cluster in ~/.kube/config
deploy-amq:kustomize
	$(KUSTOMIZE) build ./examples/manifests/amq-installer | kubectl apply -f -

# Deploy all in the configured Kubernetes cluster in ~/.kube/config
undeploy-amq:kustomize
	$(KUSTOMIZE) build ./examples/manifests/amq-installer | kubectl delete -f -

test-only:
	e2e-tests/scripts/test.sh

test-perf-only:
	e2e-tests/scripts/test.sh -p

test: | label-node deploy-amq deploy-example test-only undeploy-example undeploy-amq

test-perf: | label-node deploy-amq deploy-perf test-perf-only undeploy-example undeploy-amq

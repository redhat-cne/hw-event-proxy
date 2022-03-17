.PHONY: test

# Current  version
VERSION ?=latest

# Image name
PROXY_IMG_NAME ?= quay.io/openshift/origin-baremetal-hardware-event-proxy
SIDECAR_IMG_NAME ?= quay.io/openshift/origin-cloud-event-proxy
CONSUMER_IMG_NAME ?= quay.io/redhat-cne/cloud-event-consumer

PROXY_IMG ?= ${PROXY_IMG_NAME}:${VERSION}
SIDECAR_IMG ?= ${SIDECAR_IMG_NAME}:${VERSION}
CONSUMER_IMG ?= ${CONSUMER_IMG_NAME}:${VERSION}

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
		go get sigs.k8s.io/kustomize/kustomize/v4@v4.4.0 ;\
		rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
		}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

check-env:
	@test $${REDFISH_USERNAME?Please set environment variable REDFISH_USERNAME}
	@test $${REDFISH_PASSWORD?Please set environment variable REDFISH_PASSWORD}
	@test $${REDFISH_HOSTADDR?Please set environment variable REDFISH_HOSTADDR}

# Configure redfish credentials and BMC ip from environment variables
redfish-config:
	@sed -i -e "s/username=.*/username=${REDFISH_USERNAME}/" ./manifests/base/kustomization.yaml
	@sed -i -e "s/password=.*/password=${REDFISH_PASSWORD}/" ./manifests/base/kustomization.yaml
	@sed -i -e "s/hostaddr=.*/hostaddr=${REDFISH_HOSTADDR}/" ./manifests/base/kustomization.yaml

# label the first Ready worker node as local
label-node:
	@kubectl label --overwrite node $(shell kubectl get nodes -l node-role.kubernetes.io/worker="" | grep Ready | cut -f1 -d" " | head -1) app=local

update-image:kustomize
	cd ./manifests/base && $(KUSTOMIZE) edit set image ${PROXY_IMG_NAME}=${PROXY_IMG} \
		&& $(KUSTOMIZE) edit set image ${SIDECAR_IMG_NAME}=${SIDECAR_IMG} \
		&& $(KUSTOMIZE) edit set image ${CONSUMER_IMG_NAME}=${CONSUMER_IMG}

# Deploy all in the configured Kubernetes cluster in ~/.kube/config
deploy-amq:kustomize
	$(KUSTOMIZE) build ./manifests/amq-installer | kubectl apply -f -

undeploy-amq:kustomize
	@$(KUSTOMIZE) build ./manifests/amq-installer | kubectl delete -f -

deploy:update-image redfish-config label-node deploy-amq
	$(KUSTOMIZE) build ./manifests/base | kubectl apply -f -

undeploy:update-image undeploy-amq
	@$(KUSTOMIZE) build ./manifests/base | kubectl delete -f -

deploy-perf:update-image redfish-config label-node deploy-amq
	$(KUSTOMIZE) build ./manifests/perf | kubectl apply -f -

undeploy-perf:update-image undeploy-amq
	@$(KUSTOMIZE) build ./manifests/perf | kubectl delete -f -

test-only:
	e2e-tests/scripts/test.sh

test-perf-only:
	e2e-tests/scripts/test.sh -p

test: | check-env deploy test-only undeploy

test-perf: | check-env deploy-perf test-perf-only undeploy-perf

# Used by openshift/release. Do not check-env here since redfish hardware is not available
test-ci: | deploy test-only undeploy

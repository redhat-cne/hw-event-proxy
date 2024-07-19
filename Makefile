.PHONY: test

# Current  version
VERSION ?=latest

PROXY_IMG ?= quay.io/redhat-cne/hw-event-proxy:$(VERSION)
SIDECAR_IMG ?= quay.io/redhat-cne/cloud-event-proxy:release-4.15
CONSUMER_IMG ?= quay.io/redhat-cne/cloud-event-consumer:release-4.15

# For performance test with HTTP transport
NUM_CONSUMER ?=20

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

OS := $(shell uname -s)

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
ifeq ($(OS), Darwin)
	@sed -i "" -e "s/username=.*/username=${REDFISH_USERNAME}/" ./manifests/proxy/kustomization.yaml
	@sed -i "" -e "s/password=.*/password=${REDFISH_PASSWORD}/" ./manifests/proxy/kustomization.yaml
	@sed -i "" -e "s/hostaddr=.*/hostaddr=${REDFISH_HOSTADDR}/" ./manifests/proxy/kustomization.yaml
else
	@sed -i -e "s/username=.*/username=${REDFISH_USERNAME}/" ./manifests/proxy/kustomization.yaml
	@sed -i -e "s/password=.*/password=${REDFISH_PASSWORD}/" ./manifests/proxy/kustomization.yaml
	@sed -i -e "s/hostaddr=.*/hostaddr=${REDFISH_HOSTADDR}/" ./manifests/proxy/kustomization.yaml
endif

# label the first Ready worker node as local
label-node:
	@kubectl label --overwrite node $(shell kubectl get nodes -l node-role.kubernetes.io/worker="" | grep Ready | cut -f1 -d" " | head -1) app=local

update-image:kustomize
	cd ./manifests/proxy && $(KUSTOMIZE) edit set image hw-event-proxy=${PROXY_IMG} \
		&& $(KUSTOMIZE) edit set image cloud-event-sidecar=${SIDECAR_IMG}
	cd ./manifests/consumer && $(KUSTOMIZE) edit set image cloud-event-consumer=${CONSUMER_IMG} \
	    && $(KUSTOMIZE) edit set image cloud-event-sidecar=${SIDECAR_IMG}

# Deploy manifests in the configured Kubernetes cluster in ~/.kube/config
deploy:update-image redfish-config label-node
	$(KUSTOMIZE) build ./manifests/ns | kubectl apply -f -
	$(KUSTOMIZE) build ./manifests/proxy | kubectl apply -f -
	$(KUSTOMIZE) build ./manifests/consumer | kubectl apply -f -

undeploy:update-image
	$(KUSTOMIZE) build ./manifests/consumer | kubectl delete -f - | true
	$(KUSTOMIZE) build ./manifests/proxy | kubectl delete -f - | true
	$(KUSTOMIZE) build ./manifests/ns | kubectl delete -f -

install-pv:
	$(KUSTOMIZE) build ./manifests/pv | kubectl apply -f -

uninstall-pv:
	kubectl patch pv cloud-event-proxy-store -p '{"metadata": {"finalizers": null}}' | true
	kubectl delete pv cloud-event-proxy-store --grace-period=0 --force | true
	$(KUSTOMIZE) build ./manifests/pv | kubectl delete -f - | true

# Use PersistentVolumeClaim for pubsubstore
# The LocalVolume configuration needs to be customized for the specific hardware being used
deploy-pvc:update-image redfish-config label-node
	$(KUSTOMIZE) build ./manifests/ns | kubectl apply -f -
	$(KUSTOMIZE) build ./manifests/layers/proxy-pvc | kubectl apply -f -
	$(KUSTOMIZE) build ./manifests/consumer | kubectl apply -f -

undeploy-pvc:update-image
	$(KUSTOMIZE) build ./manifests/consumer | kubectl delete -f - | true
	$(KUSTOMIZE) build ./manifests/layers/proxy-pvc | kubectl delete -f - | true
	$(KUSTOMIZE) build ./manifests/ns | kubectl delete -f -

deploy-amq:update-image redfish-config label-node
	$(KUSTOMIZE) build ./manifests/amq-installer | kubectl apply -f -
	kubectl -n amq-router wait --for condition=available --timeout=60s deployment/amq-router
	$(KUSTOMIZE) build ./manifests/ns | kubectl apply -f -
	$(KUSTOMIZE) build ./manifests/layers/proxy-amq | kubectl apply -f -
	$(KUSTOMIZE) build ./manifests/layers/consumer-amq | kubectl apply -f -

undeploy-amq:update-image
	$(KUSTOMIZE) build ./manifests/layers/consumer-amq | kubectl delete -f - | true
	$(KUSTOMIZE) build ./manifests/layers/proxy-amq | kubectl delete -f - | true
	$(KUSTOMIZE) build ./manifests/ns | kubectl delete -f -
	$(KUSTOMIZE) build ./manifests/amq-installer | kubectl delete -f -

deploy-perf:update-image redfish-config label-node
	$(KUSTOMIZE) build ./manifests/ns | kubectl apply -f -
	$(KUSTOMIZE) build ./manifests/proxy | kubectl apply -f -
	$(KUSTOMIZE) build ./manifests/consumer | kubectl apply -f -
	@e2e-tests/scripts/deploy-multi-consumer.sh deploy ${NUM_CONSUMER}

undeploy-perf:update-image
	$(KUSTOMIZE) build ./manifests/consumer | kubectl delete -f - | true
	@e2e-tests/scripts/deploy-multi-consumer.sh undeploy ${NUM_CONSUMER}
	$(KUSTOMIZE) build ./manifests/proxy | kubectl delete -f - | true
	$(KUSTOMIZE) build ./manifests/ns | kubectl delete -f -

deploy-perf-amq:update-image redfish-config label-node
	$(KUSTOMIZE) build ./manifests/amq-installer | kubectl apply -f -
	kubectl -n amq-router wait --for condition=available --timeout=60s deployment/amq-router
	$(KUSTOMIZE) build ./manifests/ns | kubectl apply -f -
	$(KUSTOMIZE) build ./manifests/layers/proxy-amq | kubectl apply -f -
	$(KUSTOMIZE) build ./manifests/layers/multi-consumer-amq | kubectl apply -f -

undeploy-perf-amq:update-image
	$(KUSTOMIZE) build ./manifests/layers/multi-consumer-amq | kubectl delete -f - | true
	$(KUSTOMIZE) build ./manifests/layers/proxy-amq | kubectl delete -f - | true
	$(KUSTOMIZE) build ./manifests/ns | kubectl delete -f -
	$(KUSTOMIZE) build ./manifests/amq-installer | kubectl delete -f -

# example consumer deployments
deploy-consumer:update-image
	$(KUSTOMIZE) build ./manifests/consumer | kubectl apply -f -

undeploy-consumer:update-image
	$(KUSTOMIZE) build ./manifests/consumer | kubectl delete -f -

deploy-consumer-amq:update-image
	$(KUSTOMIZE) build ./manifests/layers/consumer-amq | kubectl apply -f -

undeploy-consumer-amq:update-image
	$(KUSTOMIZE) build ./manifests/layers/consumer-amq | kubectl delete -f -

test-only:
	e2e-tests/scripts/test.sh

test-perf-only:
	e2e-tests/scripts/test.sh -p -c ${NUM_CONSUMER}

test-perf-only-amq:
	e2e-tests/scripts/test.sh -p

test: | check-env deploy test-only undeploy

test-perf: | check-env deploy-perf test-perf-only undeploy-perf

test-amq: | check-env deploy-amq test-only undeploy-amq

test-perf-amq: | check-env deploy-perf-amq test-perf-only-amq undeploy-perf-amq

# Used by openshift/release. Do not check-env here since redfish hardware is not available
test-ci: | deploy test-only undeploy

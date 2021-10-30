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
	@sed -i -e "s/username=.*/username=${REDFISH_USERNAME}/" ./manifests/basic/kustomization.yaml
	@sed -i -e "s/password=.*/password=${REDFISH_PASSWORD}/" ./manifests/basic/kustomization.yaml
	@sed -i -e "s/hostaddr=.*/hostaddr=${REDFISH_HOSTADDR}/" ./manifests/basic/kustomization.yaml

# label the first Ready worker node as local
label-node:
	@kubectl label --overwrite node $(shell kubectl get nodes -l node-role.kubernetes.io/worker="" | grep Ready | cut -f1 -d" " | head -1) app=local

# Deploy all in the configured Kubernetes cluster in ~/.kube/config
deploy-amq:kustomize
	$(KUSTOMIZE) build ./manifests/amq-installer | kubectl apply -f -

undeploy-amq:kustomize
	@$(KUSTOMIZE) build ./manifests/amq-installer | kubectl delete -f -

deploy-basic:kustomize redfish-config label-node deploy-amq
	cd ./manifests/basic && $(KUSTOMIZE) edit set image hw-event-proxy=${PROXY_IMG} \
		&& $(KUSTOMIZE) edit set image cloud-event-proxy=${SIDECAR_IMG} \
		&& $(KUSTOMIZE) edit set image  cloud-native-event-consumer=${CONSUMER_IMG}
	$(KUSTOMIZE) build ./manifests/basic | kubectl apply -f -

undeploy-basic:kustomize undeploy-amq
	cd ./manifests/basic && $(KUSTOMIZE) edit set image hw-event-proxy=${PROXY_IMG} \
		&& $(KUSTOMIZE) edit set image cloud-event-proxy=${SIDECAR_IMG} \
		&& $(KUSTOMIZE) edit set image  cloud-native-event-consumer=${CONSUMER_IMG}
	@$(KUSTOMIZE) build ./manifests/basic | kubectl delete -f -

deploy-perf:kustomize redfish-config label-node deploy-amq
	cd ./manifests/basic && $(KUSTOMIZE) edit set image hw-event-proxy=${PROXY_IMG} \
		&& $(KUSTOMIZE) edit set image cloud-event-proxy=${SIDECAR_IMG} \
		&& $(KUSTOMIZE) edit set image  cloud-native-event-consumer=${CONSUMER_IMG}
	$(KUSTOMIZE) build ./manifests/perf | kubectl apply -f -

undeploy-perf:kustomize undeploy-amq
	cd ./manifests/basic && $(KUSTOMIZE) edit set image hw-event-proxy=${PROXY_IMG} \
		&& $(KUSTOMIZE) edit set image cloud-event-proxy=${SIDECAR_IMG} \
		&& $(KUSTOMIZE) edit set image  cloud-native-event-consumer=${CONSUMER_IMG}
	@$(KUSTOMIZE) build ./manifests/perf | kubectl delete -f -

test-only:
	e2e-tests/scripts/test.sh

test-perf-only:
	e2e-tests/scripts/test.sh -p

test: | check-env deploy-basic test-only undeploy-basic

test-perf: | check-env deploy-perf test-perf-only undeploy-perf

# Used by openshift/release. Do not check-env here since redfish hardware is not available
test-ci: | deploy-basic test-only undeploy-basic

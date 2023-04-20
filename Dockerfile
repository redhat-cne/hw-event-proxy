# Changes to this file are not auto-propagated to the downstream build automation. We need to make the changes manually in the midstream repo located here:
# https://gitlab.cee.redhat.com/cpaas-midstream/telco-5g-ran/baremetal-hardware-event-proxy/-/blob/rhaos-4.13-rhel-8/distgit/containers/baremetal-hardware-event-proxy/Dockerfile.in
FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.19-openshift-4.14 AS go-builder
ENV GO111MODULE=off
ENV CGO_ENABLED=1
ENV COMMON_GO_ARGS=-race
ENV GOOS=linux
ENV GOPATH=/go

COPY ./scripts /scripts
WORKDIR /go/src/github.com/redhat-cne/hw-event-proxy
COPY ./hw-event-proxy ./hw-event-proxy
RUN /scripts/build-go.sh

# Install dependencies for message-parser
FROM registry.access.redhat.com/ubi8/python-39
COPY /scripts/entrypoint.sh /
WORKDIR /message-parser
COPY ./message-parser .
RUN pip3 install -r requirements.txt

COPY --from=go-builder /go/src/github.com/redhat-cne/hw-event-proxy/hw-event-proxy/build/hw-event-proxy /

LABEL io.k8s.display-name="Bare Metal Event Relay" \
      io.k8s.description="This is a component of OpenShift Container Platform for handling hardware events." \
      io.openshift.tags="openshift" \
      maintainer="Jack Ding <jacding@redhat.com>"

ENTRYPOINT ["/entrypoint.sh"]

# Build hw-event-proxy binaries
FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.16-openshift-4.9 AS go-builder
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
FROM registry.access.redhat.com/ubi8/python-38
COPY /scripts/entrypoint.sh /
WORKDIR /message-parser
COPY ./message-parser .
RUN pip3 install -r requirements.txt

COPY --from=go-builder /go/src/github.com/redhat-cne/hw-event-proxy/hw-event-proxy/build/hw-event-proxy /

LABEL io.k8s.display-name="Hw Event Proxy" \
      io.k8s.description="This is a component of OpenShift Container Platform for handling hardware events." \
      io.openshift.tags="openshift" \
      maintainer="Jack Ding <jacding@redhat.com>"

ENTRYPOINT ["/entrypoint.sh"]
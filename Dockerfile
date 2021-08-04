# Build hw-event-proxy binaries
# FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.16-openshift-4.9 AS builder
FROM docker.io/openshift/origin-release:golang-1.15 AS go-builder
ENV GO111MODULE=off
ENV CGO_ENABLED=1
ENV COMMON_GO_ARGS=-race
ENV GOOS=linux
ENV GOPATH=/go

COPY ./scripts /scripts
WORKDIR /go/src/github.com/redhat-cne/hw-event-proxy
COPY ./hw-event-proxy ./hw-event-proxy
RUN /scripts/build-go.sh

# Build message-parser and install virtual environment
FROM docker.io/centos:centos8 as python-builder
COPY ./scripts /scripts
WORKDIR /message-parser
COPY ./message-parser .

RUN dnf install -y python3 python3-devel gcc-c++
RUN python3 -m venv venv
ENV VIRTUAL_ENV=/message-parser/venv
ENV PATH="$VIRTUAL_ENV/bin:$PATH"
RUN pip3 install -r requirements.txt

FROM docker.io/centos:centos8
COPY --from=go-builder /go/src/github.com/redhat-cne/hw-event-proxy/hw-event-proxy/build/hw-event-proxy /
COPY --from=python-builder /message-parser /message-parser
COPY /scripts/entrypoint.sh /

# python3 system libraries are required by Python virtual environment
RUN dnf install -y python3 && dnf clean all

LABEL io.k8s.display-name="Hw Event Proxy" \
      io.k8s.description="This is a component of OpenShift Container Platform for handling hardware events." \
      io.openshift.tags="openshift" \
      maintainer="Jack Ding <jacding@redhat.com>"

ENTRYPOINT ["/entrypoint.sh"]
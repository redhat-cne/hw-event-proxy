FROM quay.io/redhat-cne/openshift-origin-release:rhel-8-golang-1.20-openshift-4.15

WORKDIR /
COPY . .
RUN go build -o redfish-event-test ./cmd/main.go

ENTRYPOINT ["./scripts/entrypoint.sh"]
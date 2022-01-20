FROM quay.io/redhat-cne/openshift-origin-release:golang-1.17

WORKDIR /
COPY . .
RUN go build -o redfish-event-test ./cmd/main.go

ENTRYPOINT ["./scripts/entrypoint.sh"]
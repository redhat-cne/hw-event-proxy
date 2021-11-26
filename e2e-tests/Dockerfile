FROM quay.io/jacding/openshift-origin-release:golang-1.16

WORKDIR /
COPY . .
RUN go build -o redfish-event-test ./cmd/main.go

ENTRYPOINT ["./scripts/entrypoint.sh"]
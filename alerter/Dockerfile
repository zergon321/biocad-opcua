FROM dependencies AS builder
# Copy the application source code.
COPY ./web-server /go/src/biocad-opcua/alerter
# Build the application.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/alerter /go/src/biocad-opcua/alerter/
ENTRYPOINT [ "/go/bin/alerter" ]

FROM alpine:latest
COPY --from=builder /go/bin/alerter /bin/alerter
ENTRYPOINT [ "/bin/alerter" ]
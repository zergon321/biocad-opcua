FROM dependencies AS builder
# Copy the application source code.
COPY ./web-server /go/src/biocad-opcua/opcua-monitor
# Build the application.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/opcua-monitor /go/src/biocad-opcua/opcua-monitor/
ENTRYPOINT [ "/go/bin/opcua-monitor" ]

FROM alpine:latest
COPY --from=builder /go/bin/opcua-monitor /bin/opcua-monitor
COPY ./opcua-monitor/debug/parameters.txt /bin/parameters.txt
ENTRYPOINT [ "/bin/opcua-monitor" ]
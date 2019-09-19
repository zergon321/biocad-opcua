FROM golang:1.12 AS builder
COPY . /go/src/opcua-client
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/opcua-client /go/src/opcua-client/
ENTRYPOINT [ "/go/bin/opcua-client" ]

FROM alpine:latest
COPY --from=builder /go/bin/opcua-client /bin/opcua-client
ENTRYPOINT [ "/bin/opcua-client" ]
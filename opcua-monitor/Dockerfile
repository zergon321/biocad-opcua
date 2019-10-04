FROM golang:1.12 AS builder
COPY . /go/src/biocad-opcua
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/opcua-monitor /go/src/biocad-opcua/opcua-monitor/
ENTRYPOINT [ "/go/bin/opcua-monitor" ]

FROM alpine:latest
COPY --from=builder /go/bin/opcua-monitor /bin/opcua-monitor
COPY ./opcua-monitor/debug/parameters.txt /bin/parameters.txt
ENTRYPOINT [ "/bin/opcua-monitor" ]
FROM golang:1.12 AS builder
COPY . /go/src/biocad-opcua
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/alerter /go/src/biocad-opcua/alerter/
ENTRYPOINT [ "/go/bin/alerter" ]

FROM alpine:latest
COPY --from=builder /go/bin/alerter /bin/alerter
ENTRYPOINT [ "/bin/alerter" ]
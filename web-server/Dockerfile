FROM golang:1.12 AS builder
COPY . /go/src/biocad-opcua
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/web /go/src/biocad-opcua/web-server/
ENTRYPOINT [ "/go/bin/web" ]

FROM alpine:latest
COPY --from=builder /go/bin/web /bin/web
ENTRYPOINT [ "/bin/web" ]
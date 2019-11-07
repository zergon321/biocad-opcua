FROM dependencies AS builder
# Copy the application source code.
COPY ./web-server /go/src/biocad-opcua/web-server
# Build the application.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/web /go/src/biocad-opcua/web-server/
ENTRYPOINT [ "/go/bin/web" ]

FROM alpine:latest
COPY --from=builder /go/bin/web /bin/web
COPY --from=builder /go/src/biocad-opcua/web-client /bin/client
ENTRYPOINT [ "/bin/web" ]
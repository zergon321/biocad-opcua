FROM golang:1.12 AS dep
# Add the module files and download dependencies.
ENV GO111MODULE=on
COPY ./go.mod /go/src/biocad-opcua/go.mod
COPY ./go.sum /go/src/biocad-opcua/go.sum
WORKDIR /go/src/biocad-opcua
RUN go mod download
# Add the shared modules.
COPY ./data /go/src/biocad-opcua/data
COPY ./shared /go/src/biocad-opcua/shared
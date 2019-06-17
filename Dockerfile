FROM golang:1.12-stretch AS build

ENV GO111MODULE=on

RUN mkdir /kubewatch
WORKDIR /kubewatch
COPY go.mod .
COPY go.sum .

# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download
# COPY the source code as the last step
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /usr/local/bin/kubewatch



FROM debian:stretch-slim
COPY --from=build /usr/local/bin/kubewatch /usr/local/bin/kubewatch

ENTRYPOINT ["kubewatch"]

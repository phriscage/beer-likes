###############################################################################
##  Name:   Dockerfile
##  Date:   2018-08-20
##  Developer:  Chris Page
##  Email:  phriscage@gmail.com
##  Purpose:   This Dockerfile contains the Beers Likes example
################################################################################
## build stage
FROM golang:1.10.2-alpine3.7 AS build-env

# Set the file maintainer (your name - the file's author)
MAINTAINER Chris Page <phriscage@gmail.com>

# app working directory
WORKDIR /app

# Install Git, Go dependencies, and build the app
RUN apk --no-cache add --virtual git && \
        rm -rf /var/cache/apk/*

# Add the proto files
COPY beerlikes /go/src/github.com/phriscage/beer-likes/beerlikes

# Pull the Go dependencies
RUN go get -d -v \
	golang.org/x/net/context \
	google.golang.org/grpc \
	google.golang.org/grpc/credentials \
	google.golang.org/grpc/testdata \
        github.com/sirupsen/logrus \
	github.com/golang/protobuf/proto

# Add the sample data
COPY testdata ./testdata

# Add the application
COPY server/*.go .

# Build the package
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

EXPOSE 10000

ENTRYPOINT ["/app/main"]


## final stage
FROM alpine:3.7
WORKDIR /app
COPY --from=build-env /app /app
ENTRYPOINT ["/app/main"]

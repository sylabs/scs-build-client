# Copyright (c) 2022, Sylabs, Inc. All rights reserved.

ARG GOLANG_TAG=1.17-alpine

FROM golang:$GOLANG_TAG as build-stage
RUN apk add --no-cache git ca-certificates && update-ca-certificates
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go install -mod=vendor ./cmd/buildclient/

FROM alpine:latest
COPY --from=build-stage /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-stage /go/bin/buildclient /usr/local/bin/

USER 1000

ENTRYPOINT ["/usr/local/bin/buildclient"]

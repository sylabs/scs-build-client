// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

ARG GOLANG_TAG=1.17-alpine

FROM golang:$GOLANG_TAG as build-stage
RUN apk add --no-cache git ca-certificates && update-ca-certificates
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go install -mod=vendor -ldflags "-X main.version=`git describe --long --dirty --always`" ./cmd/scs-build/

FROM alpine:latest
COPY --from=build-stage /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-stage /go/bin/scs-build /usr/local/bin/

USER 1000

ENTRYPOINT ["/usr/local/bin/scs-build"]

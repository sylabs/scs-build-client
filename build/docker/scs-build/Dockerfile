# Copyright (c) 2022-2024, Sylabs Inc. All rights reserved.
# This software is licensed under a 3-clause BSD license. Please consult the
# LICENSE.md file distributed with the sources of this project regarding your
# rights to use or distribute this software.

FROM alpine:3 AS build-stage
RUN apk --no-cache add ca-certificates

FROM alpine:latest
COPY --from=build-stage /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY scs-build /

USER 1000

ENTRYPOINT ["/scs-build"]

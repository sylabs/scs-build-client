# SCS Build Client

[![PkgGoDev](https://pkg.go.dev/badge/github.com/sylabs/scs-build-client)](https://pkg.go.dev/github.com/sylabs/scs-build-client/client)
[![Build Status](https://circleci.com/gh/sylabs/scs-build-client.svg?style=shield)](https://circleci.com/gh/sylabs/workflows/scs-build-client)
[![Code Coverage](https://codecov.io/gh/sylabs/scs-build-client/branch/master/graph/badge.svg)](https://codecov.io/gh/sylabs/scs-build-client)
[![Go Report Card](https://goreportcard.com/badge/github.com/sylabs/scs-build-client)](https://goreportcard.com/report/github.com/sylabs/scs-build-client)

This project provides a Go client for the Singularity Container Services (SCS) Build Service.

## Go Version Compatibility

This module aims to maintain support for the two most recent stable versions of Go. This corresponds to the Go [Release Maintenance Policy](https://github.com/golang/go/wiki/Go-Release-Cycle#release-maintenance) and [Security Policy](https://golang.org/security), ensuring critical bug fixes and security patches are available for all supported language versions.

## buildclient

### Overview

`buildclient` is a tool for invoking [Sylabs Cloud](https://cloud.sylabs.io)
and Singularity Enterprise Remote Build without the need for installing and
configuring Singularity. It is intended to be integrated into a CI/CD workflow.

### Usage

#### Build and download artifact

```sh
buildclient --output alpine_latest.sif --auth-token ${SYLABS_AUTH_TOKEN} docker://alpine
```

#### Build and push to cloud library

```sh
buildclient --image-spec library://user/default/alpine:latest --auth-token ${SYLABS_AUTH_TOKEN} alpine.def
```

#### Build ephemeral artifact

```sh
export SYLABS_AUTH_TOKEN=xxx
buildclient alpine.def
```

`SYLABS_AUTH_TOKEN` is obtained through "Access Tokens" in Sylabs Cloud web UI.

### CI/CD Integration

#### GitHub Actions

Be sure to create a secret named `SYLABS_AUTH_TOKEN` containing token obtained from "Access Tokens" in [Sylabs Cloud](https://cloud.sylabs.io).

See [examples/build-def-via-gh.yaml](examples/build-def-via-gh.yaml) for an example configuration.

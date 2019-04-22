# SCS Build Client

[![GoDoc](https://godoc.org/github.com/sylabs/scs-build-client?status.svg)](https://godoc.org/github.com/sylabs/scs-build-client)
[![Build Status](https://circleci.com/gh/sylabs/scs-build-client.svg?style=shield)](https://circleci.com/gh/sylabs/workflows/scs-build-client)
[![Code Coverage](https://codecov.io/gh/sylabs/scs-build-client/branch/master/graph/badge.svg)](https://codecov.io/gh/sylabs/scs-build-client)
[![Go Report Card](https://goreportcard.com/badge/github.com/sylabs/scs-build-client)](https://goreportcard.com/report/github.com/sylabs/scs-build-client)

This project provides a Go client for the Singularity Container Services (SCS) Build Service.

## Quick Start

To build and test:

```sh
go test ./...
```

## Continuous Integration

This package uses [CircleCI](https://circleci.com) for Continuous Integration (CI). It runs automatically on commits and pull requests involving a protected branch. All CI checks must pass before a merge to a proected branch can be performed.

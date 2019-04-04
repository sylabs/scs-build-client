# SCS Remote-Build Client

[![GoDoc](https://godoc.org/github.com/sylabs/scs-remote-build-client?status.svg)](https://godoc.org/github.com/sylabs/scs-remote-build-client)
[![Build Status](https://circleci.com/gh/sylabs/scs-remote-build-client.svg?style=shield)](https://circleci.com/gh/sylabs/workflows/scs-remote-build-client)
[![Go Report Card](https://goreportcard.com/badge/github.com/sylabs/scs-remote-build-client)](https://goreportcard.com/report/github.com/sylabs/scs-remote-build-client)

This project provides a Go client for the Singularity Container Services (SCS) Remote-Build Service.

## Quick Start

Install the [CircleCI Local CLI](https://circleci.com/docs/2.0/local-cli/). See the [Continuous Integration](#continuous-integration) section below for more detail.

To build and test:

```sh
circleci build
```

## Continuous Integration

This package uses [CircleCI](https://circleci.com) for Continuous Integration (CI). It runs automatically on commits and pull requests involving a protected branch. All CI checks must pass before a merge to a proected branch can be performed.

The CI checks are typically run in the cloud without user intervention. If desired, the CI checks can also be run locally using the [CircleCI Local CLI](https://circleci.com/docs/2.0/local-cli/).

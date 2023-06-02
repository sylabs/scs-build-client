# scs-build

[![PkgGoDev](https://pkg.go.dev/badge/github.com/sylabs/scs-build-client)](https://pkg.go.dev/github.com/sylabs/scs-build-client/client)
[![Build Status](https://circleci.com/gh/sylabs/scs-build-client.svg?style=shield)](https://circleci.com/gh/sylabs/workflows/scs-build-client)
[![Code Coverage](https://codecov.io/gh/sylabs/scs-build-client/branch/main/graph/badge.svg)](https://codecov.io/gh/sylabs/scs-build-client)
[![Go Report Card](https://goreportcard.com/badge/github.com/sylabs/scs-build-client)](https://goreportcard.com/report/github.com/sylabs/scs-build-client)

`scs-build` allows users to build [Singularity Image Format (SIF)](https://github.com/sylabs/sif) images via [Singularity Container Services](https://cloud.sylabs.io) or [Singularity Enterprise](https://sylabs.io/singularity-enterprise) without the need to install and configure Singularity.

The repository also contains the [`github.com/sylabs/scs-build-client/client`](https://pkg.go.dev/github.com/sylabs/scs-build-client/client) Go module, which can be used to integrate support into other applications.

## Usage

`scs-build` is available in DockerHub ([sylabsio/scs-build](https://hub.docker.com/r/sylabsio/scs-build)) and standalone binaries are also published with [each release](https://github.com/sylabs/scs-build-client/releases).

Obtain an access token from [Singularity Container Services](https://cloud.sylabs.io) or Singularity Enterprise installation, and export it in the environment:

```sh
export SYLABS_AUTH_TOKEN=...
```

To build an image and retrieve it:

```sh
# Singularity Container Services (cloud.sylabs.io)
scs-build build recipe.def image.sif

# Singularity Enterprise (replace "cloud.enterprise.local" with your installation host name)
scs-build build --url https://cloud.enterprise.local recipe.def image.sif
```

To build an image, tag it and publish it directly to the Library:

```sh
# Singularity Container Services (cloud.sylabs.io)
scs-build build recipe.def library:<entity>/default/image:latest

# Singularity Enterprise (replace "cloud.enterprise.local" with your installation host name)
scs-build build recipe.def library://cloud.enterprise.local/<entity>/default/image:latest
```

Build an image, sign it using PGP key matching fingerprint, and publish directly to the library:

```sh
# Singularity Container Services (cloud.sylabs.io)
scs-build build recipe.def library://<entity>/default/image:latest --fingerprint ABABABABABA
```

Build an image, sign it using key 1 from the keyring, and publish directly to the library:

```sh
# Singularity Container Services (cloud.sylabs.io)
scs-build build recipe.def library://<entity>/default/image:latest --keyidx 1
```

Be sure to substitute `<entity>` appropriately (generally your username.)

## CI/CD Integration

### GitHub Actions

Create a secret named `SYLABS_AUTH_TOKEN` containing access token obtained from "Access Tokens" in [Singularity Container Services](https://cloud.sylabs.io).

See [examples/github-actions-ci.yaml](examples/github-actions-ci.yaml) for an example configuration.

### GitLab

Example [gitlab-ci.yml](examples/gitlab-ci.yml) is configured to build using file `alpine.def` contained within the project directory.

This example configuration will store the build artifact (in this case, `artifact.sif`) within GitLab. Using a library reference (ie. `library:myuser/myproject/image`) will result in the build artifact automatically being pushed to [Singularity Container Services](https://cloud.sylabs.io) or a local Singularity Enterprise installation.

## Go Version Compatibility

This module aims to maintain support for the two most recent stable versions of Go. This corresponds to the Go [Release Maintenance Policy](https://github.com/golang/go/wiki/Go-Release-Cycle#release-maintenance) and [Security Policy](https://golang.org/security), ensuring critical bug fixes and security patches are available for all supported language versions.

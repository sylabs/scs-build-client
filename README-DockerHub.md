# scs-build

`scs-build` allows users to build [Singularity Image Format (SIF)](https://github.com/sylabs/sif) images via the [Sylabs Cloud](https://cloud.sylabs.io) or [Singularity Enterprise](https://sylabs.io/singularity-enterprise) without the need to install and configure Singularity.

## Usage

Obtain an authentication token from [Sylabs Cloud](https://cloud.sylabs.io) or Singularity Enterprise installation, and export it in the environment:

```sh
export SYLABS_AUTH_TOKEN=...
```

To build an image, tag it and publish it directly to the Library:

```sh
# Sylabs Cloud (cloud.sylabs.io)
docker run -e SYLABS_AUTH_TOKEN="${SYLABS_AUTH_TOKEN}" \
    sylabsio/scs-build build recipe.def library:<entity>/default/image:latest

# Singulairty Enterprise (replace cloud.enterprise.local with your installation)
docker run -e SYLABS_AUTH_TOKEN="${SYLABS_AUTH_TOKEN}" \
    sylabsio/scs-build build recipe.def library://cloud.enterprise.local/<entity>/default/image:latest
```

To build an image and retrieve it:

```sh
# Sylabs Cloud (cloud.sylabs.io)
docker run -e SYLABS_AUTH_TOKEN="${SYLABS_AUTH_TOKEN}" \
    -u $(id -u) -v `pwd`:/work \
    sylabsio/scs-build build /work/recipe.def /work/image.sif

# Singulairty Enterprise (replace cloud.enterprise.local with your installation)
docker run -e SYLABS_AUTH_TOKEN="${SYLABS_AUTH_TOKEN}" \
    -u $(id -u) -v `pwd`:/work \
    sylabsio/scs-build build --url https://cloud.enterprise.local /work/recipe.def /work/image.sif
```

## CI/CD Integration

### GitHub Actions

Be sure to create a secret named `SYLABS_AUTH_TOKEN` containing token obtained from "Access Tokens" in [Sylabs Cloud](https://cloud.sylabs.io).

See [examples/github-actions-ci.yaml](https://github.com/sylabs/scs-build-client/blob/master/examples/github-actions-ci.yaml) for an example configuration.

### GitLab

Example [gitlab-ci.yml](https://github.com/sylabs/scs-build-client/blob/master/examples/gitlab-ci.yml) is configured to build using file `alpine.def` contained within the project directory.

This example configuration will store the build artifact (in this case, `artifact.sif`) within GitLab. Using a library reference (ie. `library:myuser/myproject/image`) will result in the build artifact automatically being pushed to [Sylabs Cloud](https://cloud.sylabs.io) or a local Singularity Enterprise installation.

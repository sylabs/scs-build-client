# scs-build

`scs-build` allows users to build [Singularity Image Format (SIF)](https://github.com/sylabs/sif) images via [Singularity Container Services](https://cloud.sylabs.io) or [Singularity Enterprise](https://sylabs.io/singularity-enterprise) without the need to install and configure Singularity.

## Usage

Obtain an access token from [Singularity Container Services](https://cloud.sylabs.io) or Singularity Enterprise installation, and export it in the environment:

```sh
export SYLABS_AUTH_TOKEN=...
```

To build an image, tag it and publish it directly to the Library:

```sh
# Singularity Container Services (cloud.sylabs.io)
docker run -e SYLABS_AUTH_TOKEN \
    -u $(id -u) -v `pwd`:/work \
    sylabsio/scs-build build /work/recipe.def library:<entity>/default/image:latest

# Singularity Enterprise (replace "cloud.enterprise.local" with your installation host name)
docker run -e SYLABS_AUTH_TOKEN \
    -u $(id -u) -v `pwd`:/work \
    sylabsio/scs-build build /work/recipe.def library://cloud.enterprise.local/<entity>/default/image:latest
```

To build an image and retrieve it:

```sh
# Singularity Container Services (cloud.sylabs.io)
docker run -e SYLABS_AUTH_TOKEN \
    -u $(id -u) -v `pwd`:/work \
    sylabsio/scs-build build /work/recipe.def /work/image.sif

# Singularity Enterprise (replace "cloud.enterprise.local" with your installation host name)
docker run -e SYLABS_AUTH_TOKEN \
    -u $(id -u) -v `pwd`:/work \
    sylabsio/scs-build build --url https://cloud.enterprise.local /work/recipe.def /work/image.sif
```

Build an image, sign it using PGP key matching fingerprint, and publish directly to the library:

```sh
# Singularity Container Services (cloud.sylabs.io)
docker run -e SYLABS_AUTH_TOKEN \
    -u $(id -u) -v `pwd`:/work \
    -v ~/.gnupg:/gnupg:ro \
    sylabsio/scs-build build /work/recipe.def library://cloud.enterprise.local/<entity>/default/image:latest \
    --keyring /gnupg/secring.gpg --fingerprint ABABABABABA

# Singularity Enterprise (replace "cloud.enterprise.local" with your installation host name)
docker run -e SYLABS_AUTH_TOKEN \
    -u $(id -u) -v `pwd`:/work \
    -v ~/.gnupg:/gnupg:ro \
    sylabsio/scs-build build /work/recipe.def library://cloud.enterprise.local/<entity>/default/image:latest \
    --url https://cloud.enterprise.local \
    --keyring /gnupg/secring.gpg --fingerprint ABABABABABA
```

Build an image, sign it using key 1 from the keyring, and retrieve it:

```sh
# Singularity Container Services (cloud.sylabs.io)
docker run -e SYLABS_AUTH_TOKEN \
    -u $(id -u) -v `pwd`:/work \
    -v ~/.gnupg:/gnupg:ro \
    sylabsio/scs-build build /work/recipe.def /work/image.sif \
    --keyring /gnupg/secring.gpg --keyidx 1 \

# Singularity Enterprise (replace "cloud.enterprise.local" with your installation host name)
docker run -e SYLABS_AUTH_TOKEN \
    -u $(id -u) -v `pwd`:/work \
    -v ~/.gnupg:/gnupg:ro \
    sylabsio/scs-build build --url https://cloud.enterprise.local /work/recipe.def /work/image.sif \
    --keyring /gnupg/secring.gpg --keyidx 1
```

Be sure to substitute `<entity>` appropriately (generally your username.)

## CI/CD Integration

### GitHub Actions

Create a secret named `SYLABS_AUTH_TOKEN` containing access token obtained from "Access Tokens" in [Singularity Container Services](https://cloud.sylabs.io).

See [examples/github-actions-ci.yaml](https://github.com/sylabs/scs-build-client/blob/main/examples/github-actions-ci.yaml) for an example configuration.

### GitLab

Example [gitlab-ci.yml](https://github.com/sylabs/scs-build-client/blob/main/examples/gitlab-ci.yml) is configured to build using file `alpine.def` contained within the project directory.

This example configuration will store the build artifact (in this case, `artifact.sif`) within GitLab. Using a library reference (ie. `library:myuser/myproject/image`) will result in the build artifact automatically being pushed to [Singularity Container Services](https://cloud.sylabs.io) or a local Singularity Enterprise installation.

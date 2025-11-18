# fsio

```bash
go get github.com/pluto-org-co/fsio
```

## License

All project runs under the AGPLv3 LICENSE.

```bash
addlicense -c 'ZedCloud Org.' -f LICENSE-HEADER *
```

## Examples

- [cmd/drive2s3](cmd/drive2s3/README.md): This is a battle tested tool used for backing up an entire Google Workspace Drive into an S3 bucket.

## Testing

Setting up podman for `testcontainers`:

```bash
set -gx DOCKER_HOST unix://$XDG_RUNTIME_DIR/podman/podman.sock
set -gx TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE /var/run/docker.sock
set -gx TESTCONTAINERS_RYUK_DISABLED true
```

# fsio

## Testing

Setting up podman for `testcontainers`:

```bash
set -gx DOCKER_HOST unix://$XDG_RUNTIME_DIR/podman/podman.sock
set -gx TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE /var/run/docker.sock
set -gx TESTCONTAINERS_RYUK_DISABLED true
```

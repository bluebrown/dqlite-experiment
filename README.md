# dqlite experiment

```bash
# build deps and start prject
make

# then stop the stack (ctrl-c)
docker compose -p test -f assets/compose.yaml down

# and start it again to see the issue
docker compose -p test -f assets/compose.yaml up
```

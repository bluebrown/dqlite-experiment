# dqlite experiment

```bash
# build deps and start prject
make

# then stop the stack
docker compose -p dqlite-experiments -f assets/compose.yaml down

# and start it again to see the issue
docker compose -p dqlite-experiments -f assets/compose.yaml up
```

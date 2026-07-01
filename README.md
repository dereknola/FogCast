# FogCast

## Building

Run the build script from the repository root:

```bash
./scripts/build.sh
```

This will:

- install frontend dependencies for `web/dm` and `web/player`
- build both frontend apps
- build the Go server binary to `bin/fogcast`

## Running And Connecting

Start the server from the repository root:

```bash
./bin/fogcast
```

By default, FogCast listens on port `8080`.

- On the same machine:
	- `http://localhost:8080`
- DM controls:
	- `http://localhost:8080/dm`
- Player display:
	- `http://localhost:8080/player`

To connect from another device on your local network, replace `localhost` with the host machine's LAN IP, for example:

- `http://192.168.1.25:8080/dm`
- `http://192.168.1.25:8080/player`

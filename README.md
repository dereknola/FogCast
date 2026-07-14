# FogCast

A lightweight, self-hosted, fog-of-war system for tabletop RPGs. Designed for use with a digital map and a projector or large display, FogCast allows a game master to reveal and shroud areas of the map in real time, while players see only what has been revealed.

DM view can supports drawing and erasing fog with a brush or rectangle tool, as well as zooming and panning the map. Pen input is supported for drawing with a stylus or finger on touch devices.

This is not a fully-featured virtual tabletop, but designed to complement in-person play.

My personal setup:
- A server running a dockerized FogCast instance, exposed via Cloudflare tunnel.
- A android TV projector, running the playe view in a browser. Projects map onto tabletop.
- A tablet running the DM view in a browser, connected to the same LAN.

This project was inspired by https://github.com/dungeon-revealer/dungeon-revealer, but that project is no longer maintained and I wanted a simpler, more lightweight solution that could run on a low-power device. 

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

## Testing And CI Quality Gates

FogCast uses a browser E2E memory gate for CI quality checks.

### Browser E2E memory gate (Player view)

This test runs the full player-memory scenario:

- starts the server
- uploads the large map (Goblin Lair)
- opens the Player view in Chromium and samples browser memory
- uploads the small map (Town Neighborhood)
- samples browser memory again and enforces per-map thresholds

Install dependencies once:

```bash
npm ci
npx playwright install chromium
```

Run the E2E memory test:

```bash
npm run test:e2e:memory
```

Threshold env vars:

- `FOGCAST_E2E_GOBLIN_MAX_MB=140`
- `FOGCAST_E2E_TOWN_MAX_MB=120`
- `FOGCAST_E2E_SUMMARY_PATH=test-results/e2e-player-memory-summary.json`

## Self-Hosting With Docker

FogCast ships with a production multi-stage [Dockerfile](Dockerfile) and a GitHub Actions workflow that publishes images to GHCR.

### Run published image from GHCR

Replace `<owner>` with your GitHub username or organization:

```bash
docker run --name fogcast \
	-p 8080:8080 \
	-v fogcast-data:/data \
	ghcr.io/<owner>/fogcast:latest
```

Open:

- `http://localhost:8080/dm`
- `http://localhost:8080/player`

### Build and run locally

```bash
docker build -t fogcast:local .
docker run --rm -p 8080:8080 -v fogcast-data:/data fogcast:local
```

### Image tags published by CI

The workflow publishes:

- `latest` (default branch)
- semantic version tags, e.g. `v1.2.3` and `1.2`
- commit SHA tags, e.g. `sha-abcdef1`

### Container configuration

These defaults are container-friendly out of the box:

- `FOGCAST_ADDR=:8080`
- `FOGCAST_DATA_DIR=/data`
- `FOGCAST_STATIC_DIR=/app/static`

You can override them by passing environment variables to `docker run`.


## AI Usage

This project was designed and implemented with the assistance of AI. Gemini-3.5 was used for initial design and architecture, and ChatGPT-5.3-Codex was used for code generation and refactoring. All code was reviewed and tested by a human developer (me) before being committed to the repository.

## Test Map Attribution (CC BY 4.0)

Some test map assets included in this repository are from Elven Tower Adventures:

- Creator: [Elven Tower Adventures](https://www.patreon.com/cw/elventower)
- License: Creative Commons Attribution 4.0 International (CC BY 4.0)
	- https://creativecommons.org/licenses/by/4.0/

These assets are used only for testing and development purposes in FogCast.




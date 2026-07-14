# FogCast

FogCast is an ultra-lightweight, self-hosted fog-of-war system built specifically to bring your digital battlemaps to life during in-person gaming sessions. 

If you use a projector, a TV laid flat on the table, or a large digital display for your tabletop RPG campaigns, FogCast gives you total control over what your players can see—in real time.

### Key Features

* **Real-Time Fog of War:** Effortlessly reveal or hide map areas on the fly. Your players only see what you want them to see.
* **Intuitive DM Controls:** Dynamically draw or erase fog using flexible **brush** or **rectangle** tools.
* **Smooth Navigation:** Zoom and pan across massive maps without losing your place.
* **Tablet & Touch Friendly:** Full support for pen styluses and finger-gestures, making it perfect for Game Masters who run their sessions from a tablet.
* **Built for Low-Power Devices:** The player view is highly optimized, meaning it runs flawlessly even on low-memory hardware like basic browsers or Android TV sticks plugged into a projector.

### What FogCast Isn't

FogCast is **not** a fully-featured, complex Virtual Tabletop (VTT) with built-in character sheets, dice rollers, or chat windows. 

### My personal setup
- A server running a dockerized FogCast instance, exposed via Cloudflare tunnel.
- A android TV projector, running the playe view in a browser. Projects map onto tabletop.
- A tablet running the DM view in a browser, connected to the same LAN.

This project was inspired by https://github.com/dungeon-revealer/dungeon-revealer, but that project is no longer maintained and I wanted a simpler, more lightweight solution that could run on a low-power device. 

## Self-Hosting With Container Image

The easiest way to self-host FogCast is to run the published container image.

### Run published image from GHCR

```bash
docker run --name fogcast \
	-p 8080:8080 \
	-v fogcast-data:/data \
	ghcr.io/dereknola/fogcast:latest
```

Open:

- `http://localhost:8080/dm`
- `http://localhost:8080/player`

### Container configuration

These defaults are container-friendly out of the box:

- `FOGCAST_ADDR=:8080`
- `FOGCAST_DATA_DIR=/data`
- `FOGCAST_STATIC_DIR=/app/static`

You can override them by passing environment variables to `docker run`.

## Building From Source

Run the build script from the repository root:

```bash
./scripts/build.sh
```

This will:

- install frontend dependencies for `web/dm` and `web/player`
- build both frontend apps
- build the Go server binary to `bin/fogcast`

### Running And Connecting

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

## Testing And CI

FogCast uses a browser E2E memory gate test for quality checks.

- starts the server
- uploads the large map (Goblin Lair)
- opens the Player view in Chromium and samples browser memory
- uploads the small map (Town Neighborhood)
- samples browser memory again and enforces per-map thresholds

### Running E2E Memory Test Locally

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

## AI Usage

This project was designed and implemented with the assistance of AI. Gemini-3.5 was used for initial design and architecture, and ChatGPT-5.3-Codex was used for code generation and refactoring. All code was reviewed and tested by a human developer (me) before being committed to the repository.

## Test Map Attribution (CC BY 4.0)

Some test map assets included in this repository are from Elven Tower Adventures:

- Creator: [Elven Tower Adventures](https://www.patreon.com/cw/elventower)
- License: Creative Commons Attribution 4.0 International (CC BY 4.0)
	- https://creativecommons.org/licenses/by/4.0/

These assets are used only for testing and development purposes in FogCast.




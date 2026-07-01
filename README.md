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


## AI Usage

This project was designed and implemented with the assistance of AI. Gemini-3.5 was used for initial design and architecture, and ChatGPT-5.3-Codex was used for code generation and refactoring. All code was reviewed and tested by a human developer (me) before being committed to the repository.
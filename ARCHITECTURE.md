# FogCast Architecture (Current Implementation)

## Purpose

FogCast is a self-hosted fog-of-war server for in-person tabletop play.

It has two browser surfaces:

- DM surface for map upload and fog editing.
- Player surface for display-only rendering on constrained devices.

Current scope intentionally excludes accounts, chat, dice, campaign management, and cloud sync.

## System Overview

| Component | Stack | Responsibility |
| --- | --- | --- |
| Backend | Go + net/http + gorilla/websocket | Authoritative session state, map optimization, persistence, API and WS transport |
| DM client | Svelte 5 + Canvas 2D | Map upload, fog editing tools, stage/push controls, player viewport controls |
| Player client | Vanilla TypeScript + WebGL | Passive render of map + fog mask with low CPU and memory overhead |

High-level data flow:

```text
DM browser
  -> POST /api/map (multipart upload)
  -> GET /api/state (authoritative state)
  -> WS /ws/dm (full-mask or patch updates + control messages)

Go backend
  -> validates and converts map uploads to WebP
  -> stores active map + mask in memory
  -> persists state to data/state.json
  -> broadcasts fog updates to WS /ws/player clients

Player browser
  -> GET /api/player/state (published state)
  -> WS /ws/player (initial mask + ongoing updates)
  -> GET /assets/maps/{id}.webp (published or active map asset)
```

## Backend Architecture

### Process and configuration

Entrypoint: cmd/fogcast/main.go

At startup:

1. Load config from env with defaults.
2. Load active map from data/state.json.
3. Initialize session manager with configured mask size.
4. Load persisted mask if dimensions match expected mask length.
5. Start HTTP server.

Config source: internal/config/config.go

| Env var | Default | Description |
| --- | --- | --- |
| FOGCAST_ADDR | :8080 | HTTP listen address |
| FOGCAST_DATA_DIR | data | Persistent state directory |
| FOGCAST_STATIC_DIR | static | Built frontend root |
| FOGCAST_MAX_UPLOAD_MB | 50 | Upload size limit |
| FOGCAST_MASK_SIZE | 512 | Requested mask width/height |

Mask size is normalized in session manager to [128, 2048], with 512 fallback for invalid low values.

### In-memory authoritative state

State model: internal/session/manager.go

- activeMap: current DM-selected map metadata.
- playerView: scale and offsets applied by player renderer.
- mask dimensions.
- serverVersion string.
- raw mask bytes in an internal byte slice.

Manager methods support:

- Full mask replacement.
- Dirty-rectangle patch application.
- reveal_all and shroud_all operations.
- Concurrent-safe reads/writes via mutex.

### Persistence model

Persistence: internal/session/persist.go

- Single file: data/state.json.
- Stored payload:
  - activeMap metadata
  - mask byte slice
- Writes are atomic via temp file + rename.
- Optimized map files are stored separately under data/maps/{id}.webp.

### HTTP API and route map

Routes are registered in internal/web/handlers.go.

| Route | Method | Behavior |
| --- | --- | --- |
| / | GET | Landing page with links to DM and Player |
| /dm, /dm/* | GET | DM app index and static files |
| /player, /player/* | GET | Player app index and static files |
| /api/state | GET | Full authoritative session state for DM |
| /api/player/state | GET | Player-safe state using published map pointer |
| /api/player/view | POST | Update player scale/offset (validated) |
| /api/map | POST | Upload and activate map, optional auto-shroud and auto-sync |
| /api/push | POST | Publish staged state and broadcast mask |
| /assets/maps/{id}.webp | GET | Serve map asset only if id is active or published |
| /ws/dm | GET | DM websocket |
| /ws/player | GET | Player websocket |

Upload behavior at /api/map:

- Accepts multipart field map.
- Supported input media types: PNG, JPEG, WebP, GIF.
- Validates dimensions (1..8192 each axis).
- Enforces max upload bytes.
- Converts to WebP and stores under data/maps.
- Sets active map and persists state.
- Optional form flags:
  - autoShroudAll (default true)
  - autoSync (default true)

When autoSync is false, updates can be staged and later published via /api/push.

### DM and player websocket protocol

Websocket handlers: internal/web/ws.go

Behavior common to both sockets:

- On connect, server sends current full mask as binary payload.

DM socket (/ws/dm):

- Binary message with exact full mask length: replace full authoritative mask.
- Binary patch message format:
  - byte 0: type (1 = patch)
  - bytes 1..4: x (uint32 LE)
  - bytes 5..8: y (uint32 LE)
  - bytes 9..12: width (uint32 LE)
  - bytes 13..16: height (uint32 LE)
  - bytes 17..end: width * height mask bytes
- Text messages are JSON control packets:
  - {"type":"reveal_all"}
  - {"type":"shroud_all"}

Player socket (/ws/player):

- Receives binary broadcasts from backend.
- Broadcast may be full mask bytes or a patch payload.
- Connection list is tracked in a player hub; broken sockets are removed on write/read failure.

## Frontend Architecture

### DM client

Main file: web/dm/src/App.svelte

Core behaviors:

- Upload map and toggle auto-shroud / auto-sync behavior.
- Brush and rectangle reveal/shroud tools.
- reveal_all and shroud_all controls.
- Optional staged workflow with manual push.
- Pointer capture for drag interactions.
- Zoom/pan on DM stage.
- Player viewport controls (scale and offsets), sent to /api/player/view.

Synchronization behavior:

- Maintains local mask typed array.
- Sends patch messages for dirty rectangles when possible.
- Falls back to full-mask send when needed.
- Applies incoming full-mask or patch updates from /ws/dm.

### Player client

Main file: web/player/src/main.ts

Core behaviors:

- Polls /api/player/state periodically (2s interval) and on socket events.
- Connects to /ws/player for near-real-time mask updates.
- Uses WebGL single full-screen quad.
- Composites map texture with fog mask texture in fragment shader.
- Supports mask patch uploads via texSubImage2D for dirty updates.
- Handles player viewport transform (scale + offsets) from backend state.

The player app has no framework runtime and is optimized around minimal allocations in update/render paths.

## Asset Processing

Map optimization: internal/assets/optimizer.go

- Detects type from bytes.
- Decodes image to validate and normalize.
- Re-encodes to WebP using nativewebp.
- Generates random 16-hex map IDs.

Only current active/published map IDs are served through /assets/maps.
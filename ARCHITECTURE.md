# D&D Battlemap Fog of War: Ultra-Lightweight Architecture


## Goal

Recreating https://github.com/dungeon-revealer/dungeon-revealer but in a more modern framework and a slimmer client footprint.

- Don't need chat or dice rolls

- For display to on client, efficient refresh and image loading. Needs to work on low end (arm mobile CPUs) and 1GB of RAM devices (like android TV sticks).

- For the DM edit mode, only need to control and display a single image at a time. Focused on the following tools:

-- reveal/shroud all

-- brush reveal/shroud with brush size

-- square box reveal/shroud

- Focus on tablet/touch screen and pen stylus for the edit mode, also mouse support

- no user/password, for the DM mode


## Architecture

FogCast is split into three runtime components: a DM editor, a lightweight backend, and a player display. The player display is the most constrained surface, so the architecture keeps nearly all interaction logic off that device.

### Component responsibilities

| Component | Primary responsibility | Technology direction |
| --- | --- | --- |
| DM client | Upload the active map, edit the fog mask, and send state changes. | Svelte, Pointer Events, Canvas 2D, typed arrays |
| Backend | Serve assets, coordinate active session state, persist local data, and broadcast updates. | Go HTTP server, filesystem storage, WebSockets |
| Player client | Load the active map and render the current fog mask with minimal CPU and memory usage. | Vanilla TypeScript, WebGL/WebGL2 canvas |

### Data flow

```text
DM tablet
  -> edits fog mask with brush, rectangle, or reveal/shroud-all tools
  -> sends mask updates over binary WebSocket frames

Go backend
  -> stores active map metadata and fog mask state
  -> optimizes uploaded maps for player display
  -> broadcasts map and mask updates to connected players

Player display
  -> receives optimized map and fog mask bytes
  -> uploads both to GPU textures
  -> composites them in a WebGL shader
```

### Player client architecture

The player client should be a pure display surface. It should load one optimized map image, receive a grayscale fog mask, upload both to WebGL textures, and redraw only when state changes.

Key constraints:

- No framework runtime on the player device.
- Avoid unnecessary object allocation during socket updates and rendering.
- Keep the render loop simple enough for low-end ARM devices and 1GB Android TV sticks.
- Prefer WebGL over Canvas 2D because the player needs GPU-backed texture compositing, not CPU-bound pixel operations.

The player renderer should use two textures:

| Texture | Contents | Notes |
| --- | --- | --- |
| Map texture | Optimized battlemap image, preferably WebP | Uploaded when the active map changes. |
| Fog texture | Low-resolution grayscale alpha mask | Uploaded when mask state changes. |

The fragment shader samples both textures with the same normalized UV coordinates. The fog texture controls final visibility, with `0` representing hidden pixels and `255` representing revealed pixels.

### Network architecture

Fog updates should travel as binary WebSocket messages instead of JSON drawing commands. The DM client performs the editing operation against a mask grid, then sends the resulting mask bytes or dirty rectangle to the server. The server validates the update, stores the authoritative state, and broadcasts it to player clients.

This keeps the player client from running brush geometry, path replay, or coordinate transform logic. Its socket handler can parse a compact header, receive a byte buffer, and upload that buffer directly to the fog texture.

### DM client architecture

The DM client can spend more resources on interaction quality because it runs on a stronger tablet, laptop, or desktop browser. It should provide the map editing workspace, tool controls, upload flow, and pointer handling.

Core input requirements:

- Support mouse, touch, and pen through Pointer Events.
- Use pointer capture during active strokes.
- Preserve accurate coordinate mapping while zooming and panning.
- Support brush reveal/shroud, rectangle reveal/shroud, reveal all, and shroud all.
- Keep mask data in typed arrays so updates can be serialized without conversion-heavy intermediate objects.

### Backend architecture

The backend should remain a lightweight local coordinator. It should avoid complex database requirements for the initial release and use local filesystem storage for uploaded maps, optimized images, and mask snapshots.

Backend responsibilities:

- Serve the DM and player frontends.
- Accept one active map upload at a time for the first release.
- Validate uploaded image type, dimensions, and size.
- Generate an optimized WebP asset for player displays.
- Maintain authoritative active map and mask state.
- Broadcast state changes to connected clients over WebSockets.
- Persist enough state to recover after a server restart.

### Architecture summary

| Area | Decision | Reason |
| --- | --- | --- |
| Player UI | Vanilla TypeScript | Minimizes memory and bundle overhead on constrained devices. |
| Player rendering | WebGL/WebGL2 | Pushes map and mask compositing to the GPU. |
| DM UI | Svelte | Provides ergonomic reactive UI for controls and editing state. |
| Input | Pointer Events | Handles mouse, touch, and stylus with one API. |
| Network | Binary WebSockets | Sends mask bytes without JSON parsing overhead. |
| Backend | Go | Produces a small self-hostable binary with efficient HTTP/WebSocket handling. |
| Storage | Local filesystem | Keeps self-hosting simple and avoids a required database. |

---

## Implementation Plan

### Product scope

FogCast is a local-network battlemap fog-of-war server with two browser surfaces:

1. **DM control surface:** Loads one active map, edits the fog mask, and controls reveal/shroud tools.
2. **Player display surface:** Passively renders the active map plus current fog mask with the smallest possible runtime.

The first production milestone should avoid campaign management, accounts, chat, dice, multi-map scenes, permissions, cloud sync, and remote hosting. The target use case is a DM device plus one or more local display devices on the same LAN.

### Technical decisions

| Area | Decision | Notes |
| --- | --- | --- |
| Backend | Go HTTP server | Serves static files, map assets, session state, and WebSocket endpoints from one small binary. |
| DM app | SvelteKit or Svelte + Vite | Use Svelte for tool panels, reactive controls, upload forms, and canvas interaction state. |
| Player app | Vanilla TypeScript | Keep runtime tiny; no framework bundle on the low-memory display device. |
| Rendering | WebGL2 with WebGL fallback if practical | Player composites map texture and fog texture in a fragment shader. DM can use Canvas 2D for editing ergonomics, then emits mask bytes. |
| Protocol | Binary WebSocket frames | Use small typed message headers plus raw payload bytes for masks and map metadata. |
| Mask format | `Uint8Array` grayscale alpha grid | `0` = hidden, `255` = revealed. Start with a fixed mask resolution, then make it configurable. |
| Storage | Local filesystem | Store uploaded source maps, optimized WebP maps, and fog mask snapshots under a data directory. |

### Target project structure

```text
FogCast/
  cmd/
    fogcast/
      main.go                 # server entrypoint and CLI flags
  internal/
    assets/
      optimizer.go            # map validation and WebP generation
    config/
      config.go               # host, port, data directory, limits
    protocol/
      messages.go             # binary message types and encoders
    session/
      manager.go              # active map, fog state, connected clients
      mask.go                 # mask grid operations and persistence
    web/
      handlers.go             # HTTP routes and static file serving
      websocket.go            # DM/player WebSocket handling
  web/
    dm/
      package.json
      src/
        routes/
          +page.svelte        # DM control page
        lib/
          components/         # toolbars, brush controls, upload controls
          fog/                # mask editing helpers
          websocket/          # DM socket client
    player/
      package.json
      src/
        main.ts               # vanilla player bootstrap
        renderer/
          webgl.ts            # shader setup and draw loop
          shaders.ts          # vertex/fragment shader source
        protocol/
          socket.ts           # binary frame parsing
  static/
    dm/                       # built DM app output
    player/                   # built player app output
  data/
    maps/                     # ignored runtime uploads
    masks/                    # ignored runtime fog snapshots
  scripts/
    build.sh                  # builds frontend bundles and Go binary
  .github/
    workflows/
      release-image.yml       # builds and publishes Docker image to GHCR
  Dockerfile                  # multi-stage production image
  docker-compose.yml          # simple self-hosting example
  go.mod
  README.md
  ARCHITECTURE.md
  ROADMAP.md
```

The `data/` directory should be gitignored once implementation starts. The `static/` build outputs can either be generated during builds or embedded into the Go binary with `embed`; embedding is preferred for single-binary deployment.

### Backend implementation details

The Go server should own authoritative session state. It should keep the active map metadata and fog mask in memory, persist snapshots to disk after edits, and broadcast state changes to connected players.

Core HTTP routes:

| Route | Purpose |
| --- | --- |
| `GET /` | Redirect to `/dm` or serve a simple landing page with DM/player links. |
| `GET /dm` | DM control surface. |
| `GET /player` | player display surface. |
| `POST /api/map` | upload and activate a battlemap. |
| `GET /api/state` | current active map metadata and mask dimensions. |
| `GET /assets/maps/{id}.webp` | optimized battlemap asset. |
| `GET /ws/dm` | authoritative DM control socket. |
| `GET /ws/player` | player subscription socket. |

Initial server constraints:

- Accept one active map at a time.
- Enforce upload size and image dimension limits.
- Convert uploaded maps to WebP before exposing them to players.
- Store fog masks as raw bytes or compressed bytes with a small metadata sidecar.
- Reject malformed WebSocket frames instead of silently ignoring them.
- Keep the player protocol stable and minimal.

### Binary protocol sketch

Every WebSocket binary frame should start with a compact fixed header:

```text
byte 0      message type
byte 1      protocol version
bytes 2-5   payload length, uint32 little-endian
bytes 6..   payload
```

Initial message types:

| Type | Direction | Payload |
| --- | --- | --- |
| `1` map metadata | server -> clients | JSON metadata is acceptable here because it is infrequent and small. |
| `2` full mask | server -> players, DM -> server | raw `Uint8Array` mask bytes. |
| `3` mask patch | DM -> server -> players | rectangle coordinates plus raw bytes for that region. |
| `4` clear/reveal all | DM -> server -> players | one byte target value: `0` or `255`. |
| `5` ping/state request | clients -> server | empty payload. |

The first milestone can send full masks after each completed stroke for simplicity. The second milestone should add rectangular patches so brush edits do not rebroadcast the entire mask.

### Fog mask model

Start with a mask resolution independent from the source map resolution, for example `512 x 512` or `1024 x 1024`. The mask maps normalized image coordinates to mask pixels. This keeps player uploads cheap and makes editing speed predictable.

Required mask operations:

- Reveal all and shroud all.
- Circular brush reveal/shroud with configurable radius.
- Rectangular reveal/shroud.
- Snapshot load/save.
- Dirty-rectangle tracking for future patch broadcasts.

### DM client implementation details

The DM client should prioritize accurate pointer handling and a responsive editing loop:

- Use Pointer Events for mouse, touch, and stylus.
- Capture active pointers with `setPointerCapture`.
- Convert screen coordinates to normalized map coordinates after zoom/pan transforms.
- Apply brush operations to an in-memory mask grid.
- Send edits to the backend on stroke completion for milestone 1.
- Add throttled patch streaming during stroke movement for milestone 2.
- Provide controls for brush size, reveal/shroud mode, rectangle mode, reveal all, shroud all, and map upload.

The DM view can use Canvas 2D for editing because it runs on stronger hardware and benefits from simpler drawing APIs. It should still use typed arrays for mask operations so the same data shape can be sent directly over the socket.

### Player client implementation details

The player app should do almost no application work:

- Load active map metadata and WebP image.
- Open `/ws/player`.
- Initialize one WebGL canvas sized to the display.
- Upload the map as one texture.
- Upload the fog mask as a single-channel texture.
- Redraw only when the map, viewport, or mask changes.
- Avoid per-frame allocations in the render loop.
- Handle reconnect by requesting the latest full state.

The fragment shader should sample the map texture and fog texture using the same normalized UV coordinates. The mask alpha determines whether a pixel is visible, hidden, or partially blended at fog edges.

### Build and development workflow

Recommended commands once implementation starts:

```text
go run ./cmd/fogcast
go test ./...
npm --prefix web/dm run build
npm --prefix web/player run build
```

The final build should:

1. Build the DM frontend.
2. Build the player frontend.
3. Copy or embed both outputs into the Go server.
4. Produce a single `fogcast` binary.
5. Build a production Docker image that runs the embedded binary.

### Self-hosting and container release workflow

The app should be easy to run on a home server, Raspberry Pi, NAS, or small cloud VM. The default deployment target should be a Docker image published to GitHub Container Registry (GHCR), with persistent map and mask data mounted as a volume.

Container goals:

- Provide a multi-stage `Dockerfile` that builds frontend assets and the Go binary, then copies only the runtime binary into a slim final image.
- Run as a non-root user in the final image.
- Expose the HTTP/WebSocket port, defaulting to `8080`.
- Store mutable data under `/data` so self-hosters can mount a volume.
- Support configuration through environment variables such as `FOGCAST_ADDR`, `FOGCAST_DATA_DIR`, `FOGCAST_MAX_UPLOAD_MB`, and `FOGCAST_PUBLIC_URL`.
- Include a `docker-compose.yml` example for one-command local hosting.
- Publish tagged images to `ghcr.io/dereknola/fogcast`.
- Publish multi-architecture images for at least `linux/amd64` and `linux/arm64`.
- Document pull/run/compose commands in `README.md`.

Recommended self-hosting commands once the image exists:

```text
docker run --rm -p 8080:8080 -v fogcast-data:/data ghcr.io/dereknola/fogcast:latest
docker compose up -d
```

### Testing strategy

- Unit test mask operations in Go or TypeScript, depending on where the authoritative editing logic lands.
- Unit test binary protocol encoding and decoding.
- Add server tests for upload validation, state responses, and WebSocket frame handling.
- Add lightweight browser smoke tests only after the initial UI stabilizes.
- Add container smoke tests for the published runtime image.
- Manually profile player memory and frame behavior on the target Android TV stick before adding nonessential UI.

### Open questions

- Should the DM editing mask operations be authoritative in the browser, the Go server, or shared between both?
- What maximum source map size should be accepted before resizing or rejecting?
- Should the optimized player map preserve full source resolution or cap dimensions for VRAM safety?
- Is a single active map enough for the first release, or should saved maps be selectable from disk?
- Should the final Go binary embed frontend assets or serve generated files from disk during development only?
- Should container releases be published only from tags, or also from every `main` branch commit?
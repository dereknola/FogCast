# FogCast Roadmap

## Implementation milestones

#### Milestone 1 (COMPLETED): runnable skeleton

- Initialize Go module and HTTP server.
- Add `/dm`, `/player`, and `/api/state` routes.
- Add placeholder DM and player frontends.
- Add a basic build script.

#### Milestone 2 (COMPLETED): active map upload

- Implement `POST /api/map`.
- Add a file upload form to the DM frontend.
- Validate uploaded images.
- Generate optimized WebP output.
- Serve the active map to both clients.
- Persist active map metadata.

#### Milestone 3 (COMPLETED): full-mask fog sync

- Add authoritative in-memory mask state.
- Add DM WebSocket and player WebSocket endpoints.
- Implement reveal all and shroud all GUI.
- Broadcast full mask updates to connected players.
- Render map plus mask in the player WebGL client.

#### Milestone 4 (COMPLETED): DM editing tools

- Implement brush reveal/shroud GUI.
- Implement brush size controls GUI.
- Implement rectangle reveal/shroud GUI.
- Add zoom and pan for DM map editing GUI.
- Persist mask snapshots.

#### Milestone 5 (COMPLETED): performance pass

- Add ENV VAR control for mask size.
- Add dirty-rectangle mask patches.
- Reduce player allocations during socket updates and rendering.
- Add reconnect/state recovery.
- Add clear limits and error messages for unsupported maps/devices.

#### Milestone 6 (COMPLETED): Docker image and GHCR release

- Add a production multi-stage `Dockerfile`.
- Add `.dockerignore` to keep image builds small and deterministic.
- Add server configuration defaults that work cleanly in containers.
- Add a GitHub Actions workflow that builds and publishes images to GHCR.
- Publish `latest`, semantic version tags.
- Build and publish `linux/amd64` and `linux/arm64` images.
- Document self-hosting with Docker.

#### Milestone 7 (COMPLETED): CI validation and quality gates

- Add a GitHub Actions workflow that runs on `push` and `workflow_dispatch`.
- Add a container smoke test job that starts the built image and checks `/api/state`.
- Add a performance test job that validates player-view low-memory behavior and baseline performance for low-end devices.
- Enforce pass/fail quality gates for smoke and performance checks before merge/release.


#### Milestone 8: Future Work

These are ideas for future work, but are not currently planned or scheduled. They may be implemented in the future if there is interest and time.
- Store multiple maps and allow switching between them.
- Add "ping" feature for DM to highlight areas of the map for players.

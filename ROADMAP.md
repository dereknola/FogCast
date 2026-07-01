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

#### Milestone 5: performance pass

- Add ENV VAR control for mask size.
- Add dirty-rectangle mask patches.
- Reduce player allocations during socket updates and rendering.
- Add reconnect/state recovery.
- Add test that validates player-view has low memory requirements and good performance for low-end devices.
- Add clear limits and error messages for unsupported maps/devices.

#### Milestone 6: Docker image and GHCR release

- Add a production multi-stage `Dockerfile`.
- Add `.dockerignore` to keep image builds small and deterministic.
- Add `docker-compose.yml` for simple self-hosting with a persistent data volume.
- Add server configuration defaults that work cleanly in containers.
- Add a GitHub Actions workflow that builds and publishes images to GHCR.
- Publish `latest`, semantic version tags, and commit SHA tags.
- Build and publish `linux/amd64` and `linux/arm64` images.
- Document self-hosting with Docker, Docker Compose, and persistent storage.
- Add a container smoke test that starts the image and checks `/api/state`.

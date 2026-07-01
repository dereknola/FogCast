<script lang="ts">
  import { onMount } from 'svelte';

  type ServerState = {
    activeMap: null | {
      id: string;
      name: string;
      width: number;
      height: number;
      url: string;
    };
    mask: {
      width: number;
      height: number;
    };
    playerView: {
      scale: number;
      offsetX: number;
      offsetY: number;
    };
    serverVersion: string;
  };

  type Tool = 'brush' | 'rectangle';
  type PaintMode = 'reveal' | 'shroud';
  type ControlsTab = 'map' | 'state';
  type StageTab = 'dm' | 'player';

  const MASK_WIDTH = 512;
  const MASK_HEIGHT = 512;
  const MAX_DM_FOG_ALPHA = 170;

  let status = $state(null as ServerState | null);
  let error = $state('');
  let info = $state('');
  let uploading = $state(false);
  let selectedFile = $state(null as File | null);
  let tool = $state('brush' as Tool);
  let mode = $state('reveal' as PaintMode);
  let brushSize = $state(36);
  let mapImageUrl = $state('');
  let panMode = $state(false);
  let viewScale = $state(1);
  let viewOffsetX = $state(0);
  let viewOffsetY = $state(0);
  let autoSync = $state(true);
  let hasPendingPush = $state(false);
  let controlsTab = $state('map' as ControlsTab);
  let stageTab = $state('dm' as StageTab);
  let showDirectionControls = $state(false);
  let autoShroudAll = $state(true);
  let playerViewScale = $state(1);
  let playerViewOffsetX = $state(0);
  let playerViewOffsetY = $state(0);

  let stageEl: HTMLDivElement | null = null;
  let overlayCanvas: HTMLCanvasElement | null = null;

  let isPointerDown = false;
  let activePointerId: number | null = null;
  let rectStart: { x: number; y: number } | null = null;
  let rectPreview: { x0: number; y0: number; x1: number; y1: number } | null = null;
  let panStart: { x: number; y: number } | null = null;
  let panOrigin: { x: number; y: number } | null = null;
  let socket: WebSocket | null = null;
  let reconnectTimer: ReturnType<typeof window.setTimeout> | null = null;

  const ALLOWED_UPLOAD_TYPES = new Set([
    'image/png',
    'image/jpeg',
    'image/webp',
    'image/gif'
  ]);

  const mask = new Uint8Array(MASK_WIDTH * MASK_HEIGHT);
  const overlayBuffer = new Uint8ClampedArray(MASK_WIDTH * MASK_HEIGHT * 4);
  const scratchCanvas = document.createElement('canvas');
  scratchCanvas.width = MASK_WIDTH;
  scratchCanvas.height = MASK_HEIGHT;

  onMount(() => {
    void loadState();
    connectSocket();
    syncCanvasSize();

    const observer = new ResizeObserver(() => {
      syncCanvasSize();
      renderOverlay();
    });

    if (stageEl) {
      observer.observe(stageEl);
    }

    return () => {
      observer.disconnect();
      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer);
      }
      socket?.close();
    };
  });

  function socketURL(path: string) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${protocol}//${window.location.host}${path}`;
  }

  function connectSocket() {
    if (socket && socket.readyState !== WebSocket.CLOSED) {
      return;
    }

    socket = new WebSocket(socketURL('/ws/dm'));
    socket.binaryType = 'arraybuffer';

    socket.onmessage = (event) => {
      if (!(event.data instanceof ArrayBuffer)) {
        return;
      }

      const incoming = new Uint8Array(event.data);
      if (incoming.length !== mask.length) {
        return;
      }

      mask.set(incoming);
      renderOverlay();
    };

    socket.onclose = () => {
      socket = null;
      reconnectTimer = window.setTimeout(() => {
        reconnectTimer = null;
        connectSocket();
      }, 1500);
    };
  }

  function sendMaskUpdate() {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      return;
    }

    socket.send(mask.slice());
  }

  function stageOrSyncMaskUpdate() {
    if (autoSync) {
      sendMaskUpdate();
      hasPendingPush = false;
      info = 'Mask synced to connected players.';
      return;
    }

    hasPendingPush = true;
    info = 'Mask changes staged. Press Push Update to send to players.';
  }

  async function pushUpdate() {
    clearMessage();

    try {
      const response = await fetch('/api/push', { method: 'POST' });
      if (!response.ok) {
        const message = await response.text();
        error = message || `Push update failed with ${response.status}.`;
        return;
      }

      sendMaskUpdate();
      hasPendingPush = false;
      info = 'Manual push sent to connected players.';
    } catch {
      error = 'Manual push failed. Check that the server is running.';
    }
  }

  function setAutoSync(enabled: boolean) {
    if (autoSync === enabled) {
      return;
    }

    autoSync = enabled;
    if (autoSync && hasPendingPush) {
      void pushUpdate();
      return;
    }

    info = autoSync
      ? 'Auto-Sync enabled. New edits will be sent immediately.'
      : 'Manual Push enabled. New edits are staged until pushed.';
  }

  function sendControl(type: 'reveal_all' | 'shroud_all') {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      return;
    }

    socket.send(JSON.stringify({ type }));
  }

  async function loadState() {
    error = '';
    info = '';

    try {
      const response = await fetch('/api/state');
      if (!response.ok) {
        error = `State request failed with ${response.status}`;
        return;
      }

      status = await response.json();
      mapImageUrl = status?.activeMap?.url ?? '';
      playerViewScale = status?.playerView?.scale ?? 1;
      playerViewOffsetX = status?.playerView?.offsetX ?? 0;
      playerViewOffsetY = status?.playerView?.offsetY ?? 0;
      info = 'Server state refreshed.';
    } catch {
      error = 'Unable to reach server state endpoint.';
    }

    renderOverlay();
  }

  function syncCanvasSize() {
    if (!stageEl || !overlayCanvas) {
      return;
    }

    const width = Math.max(1, Math.floor(stageEl.clientWidth));
    const height = Math.max(1, Math.floor(stageEl.clientHeight));

    if (overlayCanvas.width !== width) {
      overlayCanvas.width = width;
    }
    if (overlayCanvas.height !== height) {
      overlayCanvas.height = height;
    }
  }

  function renderOverlay() {
    if (!overlayCanvas) {
      return;
    }

    const ctx = overlayCanvas.getContext('2d');
    if (!ctx) {
      return;
    }

    for (let i = 0; i < mask.length; i += 1) {
      const maskValue = mask[i];
      const pixel = i * 4;

      overlayBuffer[pixel] = 15;
      overlayBuffer[pixel + 1] = 20;
      overlayBuffer[pixel + 2] = 31;
      overlayBuffer[pixel + 3] = Math.round(((255 - maskValue) / 255) * MAX_DM_FOG_ALPHA);
    }

    const image = new ImageData(overlayBuffer, MASK_WIDTH, MASK_HEIGHT);
    const scratchCtx = scratchCanvas.getContext('2d');
    if (!scratchCtx) {
      return;
    }

    scratchCtx.putImageData(image, 0, 0);

    ctx.clearRect(0, 0, overlayCanvas.width, overlayCanvas.height);
    ctx.imageSmoothingEnabled = true;
    ctx.drawImage(scratchCanvas, 0, 0, overlayCanvas.width, overlayCanvas.height);

    if (rectPreview) {
      drawRectPreview(ctx, rectPreview);
    }
  }

  function drawRectPreview(
    ctx: CanvasRenderingContext2D,
    preview: { x0: number; y0: number; x1: number; y1: number }
  ) {
    const x = Math.min(preview.x0, preview.x1) * overlayCanvas!.width;
    const y = Math.min(preview.y0, preview.y1) * overlayCanvas!.height;
    const w = Math.abs(preview.x1 - preview.x0) * overlayCanvas!.width;
    const h = Math.abs(preview.y1 - preview.y0) * overlayCanvas!.height;

    ctx.save();
    ctx.strokeStyle = mode === 'reveal' ? '#63d471' : '#ff6b6b';
    ctx.lineWidth = 2;
    ctx.setLineDash([8, 6]);
    ctx.strokeRect(x, y, w, h);
    ctx.restore();
  }

  function nudgePan(dx: number, dy: number) {
    viewOffsetX += dx;
    viewOffsetY += dy;
  }

  function resetView() {
    viewScale = 1;
    viewOffsetX = 0;
    viewOffsetY = 0;
  }

  function normalizePointer(event: PointerEvent) {
    if (!overlayCanvas) {
      return { x: 0, y: 0 };
    }

    const bounds = overlayCanvas.getBoundingClientRect();
    const x = clamp((event.clientX - bounds.left) / bounds.width, 0, 1);
    const y = clamp((event.clientY - bounds.top) / bounds.height, 0, 1);

    return { x, y };
  }

  function toMaskPoint(normX: number, normY: number) {
    return {
      x: Math.min(MASK_WIDTH - 1, Math.floor(normX * MASK_WIDTH)),
      y: Math.min(MASK_HEIGHT - 1, Math.floor(normY * MASK_HEIGHT))
    };
  }

  function clamp(value: number, min: number, max: number) {
    return Math.min(max, Math.max(min, value));
  }

  function paintBrush(normX: number, normY: number) {
    const center = toMaskPoint(normX, normY);
    const radius = Math.max(1, Math.floor(brushSize / 2));
    const value = mode === 'reveal' ? 255 : 0;

    const minY = Math.max(0, center.y - radius);
    const maxY = Math.min(MASK_HEIGHT - 1, center.y + radius);
    const minX = Math.max(0, center.x - radius);
    const maxX = Math.min(MASK_WIDTH - 1, center.x + radius);

    for (let y = minY; y <= maxY; y += 1) {
      for (let x = minX; x <= maxX; x += 1) {
        const dx = x - center.x;
        const dy = y - center.y;
        if (dx * dx + dy * dy > radius * radius) {
          continue;
        }

        mask[y * MASK_WIDTH + x] = value;
      }
    }

    renderOverlay();
  }

  function applyRect(
    start: { x: number; y: number },
    end: { x: number; y: number }
  ) {
    const value = mode === 'reveal' ? 255 : 0;
    const a = toMaskPoint(start.x, start.y);
    const b = toMaskPoint(end.x, end.y);

    const minX = Math.min(a.x, b.x);
    const maxX = Math.max(a.x, b.x);
    const minY = Math.min(a.y, b.y);
    const maxY = Math.max(a.y, b.y);

    for (let y = minY; y <= maxY; y += 1) {
      const row = y * MASK_WIDTH;
      for (let x = minX; x <= maxX; x += 1) {
        mask[row + x] = value;
      }
    }

    renderOverlay();
  }

  function onStagePointerDown(event: PointerEvent) {
    if (!overlayCanvas) {
      return;
    }

    isPointerDown = true;
    activePointerId = event.pointerId;
    overlayCanvas.setPointerCapture(event.pointerId);

    if (panMode) {
      panStart = { x: event.clientX, y: event.clientY };
      panOrigin = { x: viewOffsetX, y: viewOffsetY };
      return;
    }

    const norm = normalizePointer(event);
    if (tool === 'brush') {
      paintBrush(norm.x, norm.y);
      return;
    }

    rectStart = norm;
    rectPreview = { x0: norm.x, y0: norm.y, x1: norm.x, y1: norm.y };
    renderOverlay();
  }

  function onStagePointerMove(event: PointerEvent) {
    if (!isPointerDown || activePointerId !== event.pointerId) {
      return;
    }

    if (panMode && panStart && panOrigin) {
      viewOffsetX = panOrigin.x + (event.clientX - panStart.x);
      viewOffsetY = panOrigin.y + (event.clientY - panStart.y);
      return;
    }

    const norm = normalizePointer(event);
    if (tool === 'brush') {
      paintBrush(norm.x, norm.y);
      return;
    }

    if (!rectStart) {
      return;
    }

    rectPreview = { x0: rectStart.x, y0: rectStart.y, x1: norm.x, y1: norm.y };
    renderOverlay();
  }

  function onStagePointerUp(event: PointerEvent) {
    if (!isPointerDown || activePointerId !== event.pointerId) {
      return;
    }

    isPointerDown = false;
    activePointerId = null;

    if (overlayCanvas?.hasPointerCapture(event.pointerId)) {
      overlayCanvas.releasePointerCapture(event.pointerId);
    }

    if (panMode) {
      panStart = null;
      panOrigin = null;
      info = 'View adjusted.';
      return;
    }

    if (tool === 'rectangle' && rectStart) {
      const norm = normalizePointer(event);
      applyRect(rectStart, norm);
    }

    rectStart = null;
    rectPreview = null;
    stageOrSyncMaskUpdate();
    renderOverlay();
  }

  function revealAll() {
    mask.fill(255);
    info = 'All cells revealed.';
    if (autoSync) {
      sendControl('reveal_all');
    }
    stageOrSyncMaskUpdate();
    renderOverlay();
  }

  function shroudAll() {
    mask.fill(0);
    info = 'All cells shrouded.';
    if (autoSync) {
      sendControl('shroud_all');
    }
    stageOrSyncMaskUpdate();
    renderOverlay();
  }

  function clearMessage() {
    error = '';
    info = '';
  }

  function formatSelectedFilename(name: string, maxLength = 44) {
    if (name.length <= maxLength) {
      return name;
    }

    return `${name.slice(0, maxLength - 3)}...`;
  }

  function openPlayerView() {
    window.open('/player', '_blank', 'noopener,noreferrer');
  }

  async function syncPlayerView() {
    clearMessage();

    try {
      const response = await fetch('/api/player/view', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          scale: playerViewScale,
          offsetX: Math.round(playerViewOffsetX),
          offsetY: Math.round(playerViewOffsetY)
        })
      });

      if (!response.ok) {
        const message = await response.text();
        error = message || `Player view update failed with ${response.status}.`;
        return;
      }

      info = 'Player view updated.';
    } catch {
      error = 'Unable to sync player view.';
    }
  }

  function nudgePlayerView(dx: number, dy: number) {
    playerViewOffsetX += dx;
    playerViewOffsetY += dy;
    void syncPlayerView();
  }

  function resetPlayerView() {
    playerViewScale = 1;
    playerViewOffsetX = 0;
    playerViewOffsetY = 0;
    void syncPlayerView();
  }

  function onFileSelected(event: Event) {
    const target = event.currentTarget as HTMLInputElement;
    const file = target.files?.[0] ?? null;
    selectedFile = null;

    if (!file) {
      mapImageUrl = status?.activeMap?.url ?? '';
      return;
    }

    if (!ALLOWED_UPLOAD_TYPES.has(file.type)) {
      clearMessage();
      error = 'Unsupported file type. Use PNG, JPEG, WebP, or GIF.';
      target.value = '';
      mapImageUrl = status?.activeMap?.url ?? '';
      return;
    }

    selectedFile = file;

    clearMessage();
    info = `Uploading ${file.name}...`;
    void uploadMap();
  }

  async function uploadMap() {
    if (!selectedFile) {
      error = 'Select an image file before upload.';
      return;
    }

    clearMessage();
    uploading = true;

    try {
      const formData = new FormData();
      formData.append('map', selectedFile);
      formData.append('autoShroudAll', autoShroudAll ? 'true' : 'false');
      formData.append('autoSync', autoSync ? 'true' : 'false');

      const response = await fetch('/api/map', {
        method: 'POST',
        body: formData
      });

      if (!response.ok) {
        const message = await response.text();
        error = message || `Map upload failed with ${response.status}.`;
        return;
      }

      selectedFile = null;
      await loadState();
      if (autoSync) {
        hasPendingPush = false;
        info = 'Map uploaded and synced to players.';
      } else {
        hasPendingPush = true;
        info = 'New map staged. Press Push Update to send to players.';
      }
    } catch {
      error = 'Map upload failed. Check that the server is running.';
    } finally {
      uploading = false;
    }
  }
</script>

<main>
  <header class="hero">
    <p class="eyebrow">FogCast DM</p>
    <p>Upload a map, choose your tool, and paint the fog mask. Current edits render locally in the browser.</p>
  </header>

  <div class="layout">
    <section class="panel controls">
      <div class="tab-row" role="tablist" aria-label="DM controls tabs">
        <button
          type="button"
          role="tab"
          class:active={controlsTab === 'map'}
          aria-selected={controlsTab === 'map'}
          onclick={() => (controlsTab = 'map')}
        >
          Map
        </button>
        <button
          type="button"
          role="tab"
          class:active={controlsTab === 'state'}
          aria-selected={controlsTab === 'state'}
          onclick={() => (controlsTab = 'state')}
        >
          State
        </button>
      </div>

      {#if controlsTab === 'map'}
        <h2>Map</h2>
        <label class="label" for="map-file">Image file</label>
        <input id="map-file" type="file" accept="image/*" onchange={onFileSelected} />
        {#if selectedFile}
          <p class="file-meta">
            Selected: {formatSelectedFilename(selectedFile.name)} ({Math.max(1, Math.round(selectedFile.size / 1024))} KB)
          </p>
        {/if}

        <h2>Tools</h2>
        <div class="button-row">
          <button type="button" class:active={tool === 'brush'} onclick={() => (tool = 'brush')}>Brush</button>
          <button type="button" class:active={tool === 'rectangle'} onclick={() => (tool = 'rectangle')}>Rectangle</button>
        </div>

        <div class="button-row">
          <button type="button" class:active={mode === 'reveal'} onclick={() => (mode = 'reveal')}>Reveal</button>
          <button type="button" class:active={mode === 'shroud'} onclick={() => (mode = 'shroud')}>Shroud</button>
        </div>

        <h2>View</h2>
        <button type="button" class:active={panMode} onclick={() => (panMode = !panMode)}>
          {panMode ? 'Pan mode: on' : 'Pan mode: off'}
        </button>
        <label class="label" for="zoom-level">Zoom: {viewScale.toFixed(1)}x</label>
        <input
          id="zoom-level"
          type="range"
          min="1"
          max="3"
          step="0.1"
          bind:value={viewScale}
        />
        {#if showDirectionControls}
          <div class="arrow-row">
            <button type="button" class="arrow-button" aria-label="Pan left" onclick={() => nudgePan(-40, 0)}>&larr;</button>
            <button type="button" class="arrow-button" aria-label="Pan up" onclick={() => nudgePan(0, -40)}>&uarr;</button>
            <button type="button" class="arrow-button" aria-label="Pan down" onclick={() => nudgePan(0, 40)}>&darr;</button>
            <button type="button" class="arrow-button" aria-label="Pan right" onclick={() => nudgePan(40, 0)}>&rarr;</button>
          </div>
        {/if}
        <button type="button" onclick={resetView}>Reset view</button>

        <label class="label" for="brush-size">Brush size: {brushSize}px</label>
        <input
          id="brush-size"
          type="range"
          min="8"
          max="120"
          step="2"
          bind:value={brushSize}
          disabled={tool !== 'brush'}
        />

        <div class="button-row">
          <button type="button" onclick={revealAll}>Reveal all</button>
          <button type="button" onclick={shroudAll}>Shroud all</button>
        </div>

        <h2>Refresh mode</h2>
        <div class="button-row">
          <button type="button" class:active={autoSync} onclick={() => setAutoSync(true)}>Auto-Sync</button>
          <button type="button" class:active={!autoSync} onclick={() => setAutoSync(false)}>Manual Push</button>
        </div>
        {#if !autoSync}
          <button type="button" onclick={pushUpdate} disabled={!hasPendingPush}>Push Update</button>
        {/if}
      {/if}

      {#if controlsTab === 'state'}
        <h2>Server state</h2>
        {#if status}
          <dl>
            <div>
              <dt>Active map</dt>
              <dd>{status.activeMap ? status.activeMap.name : 'None loaded'}</dd>
            </div>
            <div>
              <dt>Server version</dt>
              <dd>{status.serverVersion}</dd>
            </div>
          </dl>
        {:else}
          <p>Loading server state...</p>
        {/if}

        <button type="button" onclick={loadState}>Refresh state</button>

        <h2>Display options</h2>
        <button
          type="button"
          class:active={showDirectionControls}
          onclick={() => (showDirectionControls = !showDirectionControls)}
        >
          Show Direction Controls: {showDirectionControls ? 'On' : 'Off'}
        </button>

        <button
          type="button"
          class:active={autoShroudAll}
          onclick={() => (autoShroudAll = !autoShroudAll)}
        >
          Auto Shroud All: {autoShroudAll ? 'On' : 'Off'}
        </button>
      {/if}

      {#if info}
        <p class="info">{info}</p>
      {/if}
      {#if error}
        <p class="error">{error}</p>
      {/if}
    </section>

    <section class="panel stage-panel">
      <div class="tab-row" role="tablist" aria-label="Stage tabs">
        <button
          type="button"
          role="tab"
          class:active={stageTab === 'dm'}
          aria-selected={stageTab === 'dm'}
          onclick={() => (stageTab = 'dm')}
        >
          DM View
        </button>
        <button
          type="button"
          role="tab"
          class:active={stageTab === 'player'}
          aria-selected={stageTab === 'player'}
          onclick={() => (stageTab = 'player')}
        >
          Player View
        </button>
      </div>

      <div class="stage-header">
        <h2>{stageTab === 'dm' ? 'DM View' : 'Player View'}</h2>
        <button type="button" onclick={openPlayerView}>Open Player Site</button>
      </div>

      {#if stageTab === 'player'}
        <div class="player-controls">
          <label class="label" for="player-zoom-level">Player zoom: {playerViewScale.toFixed(1)}x</label>
          <input
            id="player-zoom-level"
            type="range"
            min="1"
            max="3"
            step="0.1"
            bind:value={playerViewScale}
            onchange={() => void syncPlayerView()}
          />
          <div class="arrow-row">
            <button type="button" class="arrow-button" aria-label="Player pan left" onclick={() => nudgePlayerView(-40, 0)}>&larr;</button>
            <button type="button" class="arrow-button" aria-label="Player pan up" onclick={() => nudgePlayerView(0, -40)}>&uarr;</button>
            <button type="button" class="arrow-button" aria-label="Player pan down" onclick={() => nudgePlayerView(0, 40)}>&darr;</button>
            <button type="button" class="arrow-button" aria-label="Player pan right" onclick={() => nudgePlayerView(40, 0)}>&rarr;</button>
          </div>
          <button type="button" onclick={resetPlayerView}>Reset player view</button>
        </div>
      {/if}

      <div class="stage" bind:this={stageEl}>
        <div
          class="viewport"
          style={stageTab === 'dm'
            ? `transform: translate(${viewOffsetX}px, ${viewOffsetY}px) scale(${viewScale});`
            : `transform: translate(${playerViewOffsetX}px, ${playerViewOffsetY}px) scale(${playerViewScale});`}
        >
          {#if mapImageUrl}
            <img class="map" src={mapImageUrl} alt="Active map" draggable="false" />
          {:else}
            <div class="map fallback">No map loaded yet</div>
          {/if}

          <canvas
            bind:this={overlayCanvas}
            class="overlay"
            onpointerdown={onStagePointerDown}
            onpointermove={onStagePointerMove}
            onpointerup={onStagePointerUp}
            onpointercancel={onStagePointerUp}
            style={stageTab === 'player' ? 'pointer-events: none;' : ''}
          ></canvas>
        </div>
      </div>
      <p class="hint">
        {#if stageTab === 'dm'}
          Brush paints continuously. Rectangle applies on pointer release. Enable pan mode to drag the view.
        {:else}
          Player View controls what players see on the player site.
        {/if}
      </p>
    </section>
  </div>
</main>

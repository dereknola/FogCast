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
    serverVersion: string;
  };

  type Tool = 'brush' | 'rectangle';
  type PaintMode = 'reveal' | 'shroud';

  const MASK_WIDTH = 512;
  const MASK_HEIGHT = 512;

  let state = $state<ServerState | null>(null);
  let error = $state('');
  let info = $state('');
  let uploading = $state(false);
  let selectedFile = $state<File | null>(null);
  let tool = $state<Tool>('brush');
  let mode = $state<PaintMode>('reveal');
  let brushSize = $state(36);
  let mapImageUrl = $state('');

  let stageEl: HTMLDivElement | null = null;
  let overlayCanvas: HTMLCanvasElement | null = null;

  let isPointerDown = false;
  let activePointerId: number | null = null;
  let rectStart: { x: number; y: number } | null = null;
  let rectPreview: { x0: number; y0: number; x1: number; y1: number } | null = null;
  let previewObjectUrl = '';

  const mask = new Uint8Array(MASK_WIDTH * MASK_HEIGHT);
  const overlayBuffer = new Uint8ClampedArray(MASK_WIDTH * MASK_HEIGHT * 4);
  const scratchCanvas = document.createElement('canvas');
  scratchCanvas.width = MASK_WIDTH;
  scratchCanvas.height = MASK_HEIGHT;

  onMount(() => {
    void loadState();
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
      if (previewObjectUrl) {
        URL.revokeObjectURL(previewObjectUrl);
      }
    };
  });

  async function loadState() {
    error = '';
    info = '';

    try {
      const response = await fetch('/api/state');
      if (!response.ok) {
        error = `State request failed with ${response.status}`;
        return;
      }

      state = await response.json();
      mapImageUrl = state?.activeMap?.url ?? '';
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
      overlayBuffer[pixel + 3] = 255 - maskValue;
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

    if (tool === 'rectangle' && rectStart) {
      const norm = normalizePointer(event);
      applyRect(rectStart, norm);
    }

    rectStart = null;
    rectPreview = null;
    info = 'Mask updated locally. Server sync lands in the next backend milestone.';
    renderOverlay();
  }

  function revealAll() {
    mask.fill(255);
    info = 'All cells revealed locally.';
    renderOverlay();
  }

  function shroudAll() {
    mask.fill(0);
    info = 'All cells shrouded locally.';
    renderOverlay();
  }

  function clearMessage() {
    error = '';
    info = '';
  }

  function onFileSelected(event: Event) {
    const target = event.currentTarget as HTMLInputElement;
    const file = target.files?.[0] ?? null;
    selectedFile = file;

    if (!file) {
      return;
    }

    clearMessage();
    if (previewObjectUrl) {
      URL.revokeObjectURL(previewObjectUrl);
    }

    previewObjectUrl = URL.createObjectURL(file);
    mapImageUrl = previewObjectUrl;
    info = `Selected ${file.name}.`;
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

      const response = await fetch('/api/map', {
        method: 'POST',
        body: formData
      });

      if (!response.ok) {
        error = `Map upload not available yet (${response.status}).`;
        return;
      }

      info = 'Map uploaded successfully.';
      await loadState();
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
    <h1>Control surface</h1>
    <p>Upload a map, choose your tool, and paint the fog mask. Current edits render locally in the browser and are ready to wire to WebSocket sync.</p>
  </header>

  <div class="layout">
    <section class="panel controls">
      <h2>Map</h2>
      <label class="label" for="map-file">Image file</label>
      <input id="map-file" type="file" accept="image/*" onchange={onFileSelected} />
      <button type="button" onclick={uploadMap} disabled={uploading}>
        {uploading ? 'Uploading...' : 'Upload map'}
      </button>

      <h2>Tools</h2>
      <div class="button-row">
        <button type="button" class:active={tool === 'brush'} onclick={() => (tool = 'brush')}>Brush</button>
        <button type="button" class:active={tool === 'rectangle'} onclick={() => (tool = 'rectangle')}>Rectangle</button>
      </div>

      <div class="button-row">
        <button type="button" class:active={mode === 'reveal'} onclick={() => (mode = 'reveal')}>Reveal</button>
        <button type="button" class:active={mode === 'shroud'} onclick={() => (mode = 'shroud')}>Shroud</button>
      </div>

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

      <h2>Server state</h2>
      {#if state}
        <dl>
          <div>
            <dt>Active map</dt>
            <dd>{state.activeMap ? state.activeMap.name : 'None loaded'}</dd>
          </div>
          <div>
            <dt>Mask</dt>
            <dd>{state.mask.width} x {state.mask.height}</dd>
          </div>
          <div>
            <dt>Server version</dt>
            <dd>{state.serverVersion}</dd>
          </div>
        </dl>
      {:else}
        <p>Loading server state...</p>
      {/if}

      <button type="button" onclick={loadState}>Refresh state</button>

      {#if info}
        <p class="info">{info}</p>
      {/if}
      {#if error}
        <p class="error">{error}</p>
      {/if}
    </section>

    <section class="panel stage-panel">
      <h2>Map stage</h2>
      <div class="stage" bind:this={stageEl}>
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
        ></canvas>
      </div>
      <p class="hint">Brush paints continuously. Rectangle applies on pointer release. Reveal = visible map, shroud = hidden map.</p>
    </section>
  </div>
</main>

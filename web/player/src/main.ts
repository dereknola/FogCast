import './style.css';

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

const canvasEl = document.getElementById('fogcast-canvas');
const statusEl = document.getElementById('status');

if (!(canvasEl instanceof HTMLCanvasElement)) {
  throw new Error('Missing player canvas');
}
if (!statusEl) {
  throw new Error('Missing player status element');
}
const canvas = canvasEl;
const status = statusEl;
const ctx2d = canvas.getContext('2d');
if (!ctx2d) {
  throw new Error('Missing 2D context');
}
const ctx = ctx2d;

let loadedMap: HTMLImageElement | null = null;
let activeMapID = '';
let refreshing = false;

resizeCanvas();
window.addEventListener('resize', resizeCanvas);

void refreshState();
window.setInterval(() => {
  void refreshState();
}, 2000);

async function refreshState() {
  if (refreshing) {
    return;
  }
  refreshing = true;

  try {
    await loadState();
  } finally {
    refreshing = false;
  }
}

async function loadState() {
  const response = await fetch('/api/state');
  if (!response.ok) {
    status.textContent = `State request failed with ${response.status}`;
    return;
  }

  const state = (await response.json()) as ServerState;
  if (!state.activeMap) {
    loadedMap = null;
    activeMapID = '';
    render();
    status.textContent = `Waiting for a map. Mask ${state.mask.width} x ${state.mask.height}.`;
    return;
  }

  if (state.activeMap.id === activeMapID && loadedMap) {
    status.textContent = `Ready: ${state.activeMap.name}`;
    return;
  }

  try {
    loadedMap = await loadMapImage(state.activeMap.url, state.activeMap.id);
    activeMapID = state.activeMap.id;
    render();
    status.textContent = `Ready: ${state.activeMap.name}`;
  } catch {
    status.textContent = `Map load failed for ${state.activeMap.name}`;
  }
}

function resizeCanvas() {
  const scale = window.devicePixelRatio || 1;
  canvas.width = Math.floor(window.innerWidth * scale);
  canvas.height = Math.floor(window.innerHeight * scale);

  render();
}

function render() {
  ctx.fillStyle = '#05070d';
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  if (!loadedMap) {
    return;
  }

  const target = contain(loadedMap.width, loadedMap.height, canvas.width, canvas.height);
  ctx.drawImage(loadedMap, target.x, target.y, target.width, target.height);
}

function loadMapImage(url: string, revision: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.decoding = 'async';
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error('failed to load map image'));
    image.src = `${url}?v=${encodeURIComponent(revision)}`;
  });
}

function contain(sourceWidth: number, sourceHeight: number, maxWidth: number, maxHeight: number) {
  const sourceRatio = sourceWidth / sourceHeight;
  const maxRatio = maxWidth / maxHeight;

  if (sourceRatio > maxRatio) {
    const width = maxWidth;
    const height = Math.round(width / sourceRatio);
    return {
      x: 0,
      y: Math.floor((maxHeight - height) / 2),
      width,
      height
    };
  }

  const height = maxHeight;
  const width = Math.round(height * sourceRatio);
  return {
    x: Math.floor((maxWidth - width) / 2),
    y: 0,
    width,
    height
  };
}


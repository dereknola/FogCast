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

const canvas = document.getElementById('fogcast-canvas');
const status = document.getElementById('status');

if (!(canvas instanceof HTMLCanvasElement)) {
  throw new Error('Missing player canvas');
}
if (!status) {
  throw new Error('Missing player status element');
}

resizeCanvas();
window.addEventListener('resize', resizeCanvas);

void loadState();

async function loadState() {
  const response = await fetch('/api/state');
  if (!response.ok) {
    status.textContent = `State request failed with ${response.status}`;
    return;
  }

  const state = (await response.json()) as ServerState;
  status.textContent = state.activeMap
    ? `Ready: ${state.activeMap.name}`
    : `Waiting for a map. Mask ${state.mask.width} x ${state.mask.height}.`;
}

function resizeCanvas() {
  const scale = window.devicePixelRatio || 1;
  canvas.width = Math.floor(window.innerWidth * scale);
  canvas.height = Math.floor(window.innerHeight * scale);
}


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
  playerView: {
    scale: number;
    offsetX: number;
    offsetY: number;
  };
  serverVersion: string;
};

const DEFAULT_MASK_WIDTH = 512;
const DEFAULT_MASK_HEIGHT = 512;
const MASK_PATCH_MESSAGE_TYPE = 1;
const MAX_MASK_DIMENSION = 2048;

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
const webglContext = canvas.getContext('webgl');
if (!webglContext) {
  status.textContent = 'This device/browser does not support WebGL. FogCast player requires WebGL support.';
  throw new Error('Missing WebGL context');
}
const gl: WebGLRenderingContext = webglContext;

// Dirty-rectangle updates may have widths that are not multiples of 4 bytes.
// Use byte alignment 1 so tex(Sub)Image uploads are not row-corrupted.
gl.pixelStorei(gl.UNPACK_ALIGNMENT, 1);

const vertexSource = `
attribute vec2 a_position;
varying vec2 v_uv;

void main() {
  v_uv = (a_position + 1.0) * 0.5;
  gl_Position = vec4(a_position, 0.0, 1.0);
}
`;

const fragmentSource = `
precision mediump float;

uniform sampler2D u_map;
uniform sampler2D u_mask;
uniform vec2 u_map_scale;
uniform vec2 u_map_offset;
varying vec2 v_uv;

void main() {
  vec2 map_uv = (v_uv - u_map_offset) / u_map_scale;

  if (map_uv.x < 0.0 || map_uv.x > 1.0 || map_uv.y < 0.0 || map_uv.y > 1.0) {
    gl_FragColor = vec4(0.02, 0.03, 0.06, 1.0);
    return;
  }

  vec4 map_color = texture2D(u_map, vec2(map_uv.x, 1.0 - map_uv.y));
  float mask = texture2D(u_mask, vec2(map_uv.x, 1.0 - map_uv.y)).r;
  vec3 fog_color = vec3(0.02, 0.03, 0.06);
  vec3 color = mix(fog_color, map_color.rgb, mask);

  gl_FragColor = vec4(color, 1.0);
}
`;

const program = createProgram(gl, vertexSource, fragmentSource);
gl.useProgram(program);

const positionLoc = gl.getAttribLocation(program, 'a_position');
const mapScaleLoc = gl.getUniformLocation(program, 'u_map_scale');
const mapOffsetLoc = gl.getUniformLocation(program, 'u_map_offset');
const mapSamplerLoc = gl.getUniformLocation(program, 'u_map');
const maskSamplerLoc = gl.getUniformLocation(program, 'u_mask');

if (!mapScaleLoc || !mapOffsetLoc || !mapSamplerLoc || !maskSamplerLoc) {
  throw new Error('Missing shader uniforms');
}

const quad = gl.createBuffer();
if (!quad) {
  throw new Error('Failed to create quad buffer');
}

const mapTexture = createTexture(gl);
const maskTexture = createTexture(gl);

let maskWidth = DEFAULT_MASK_WIDTH;
let maskHeight = DEFAULT_MASK_HEIGHT;
let latestMask = new Uint8Array(maskWidth * maskHeight);
let mapImage: HTMLImageElement | null = null;
let activeMapID = '';
let playerViewScale = 1;
let playerViewOffsetX = 0;
let playerViewOffsetY = 0;
let refreshing = false;
let socket: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof window.setTimeout> | null = null;

setupGeometry();
updateMaskTexture(latestMask);
resizeCanvas();
connectSocket();
void refreshState();
window.setInterval(() => {
  void refreshState();
}, 2000);
window.addEventListener('resize', resizeCanvas);

function setupGeometry() {
  gl.bindBuffer(gl.ARRAY_BUFFER, quad);
  gl.bufferData(
    gl.ARRAY_BUFFER,
    new Float32Array([
      -1,
      -1,
      1,
      -1,
      -1,
      1,
      -1,
      1,
      1,
      -1,
      1,
      1
    ]),
    gl.STATIC_DRAW
  );

  gl.enableVertexAttribArray(positionLoc);
  gl.vertexAttribPointer(positionLoc, 2, gl.FLOAT, false, 0, 0);
}

function socketURL(path: string) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}${path}`;
}

function connectSocket() {
  if (socket && socket.readyState !== WebSocket.CLOSED) {
    return;
  }

  socket = new WebSocket(socketURL('/ws/player'));
  socket.binaryType = 'arraybuffer';

  socket.onopen = () => {
    status.textContent = status.textContent || 'Connected';
    void refreshState();
  };

  socket.onmessage = (event) => {
    if (!(event.data instanceof ArrayBuffer)) {
      return;
    }

    const incoming = new Uint8Array(event.data);
    if (incoming.length === latestMask.length) {
      latestMask.set(incoming);
      updateMaskTexture(latestMask);
      render();
      return;
    }

    const patch = decodeMaskPatch(incoming);
    if (!patch) {
      return;
    }

    for (let row = 0; row < patch.height; row += 1) {
      const srcStart = row * patch.width;
      const dstStart = (patch.y + row) * maskWidth + patch.x;
      latestMask.set(patch.data.subarray(srcStart, srcStart + patch.width), dstStart);
    }

    updateMaskTexturePatch(patch.x, patch.y, patch.width, patch.height, patch.data);
    render();
  };

  socket.onclose = () => {
    socket = null;
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = null;
      connectSocket();
    }, 1500);
  };
}

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
  const response = await fetch('/api/player/state');
  if (!response.ok) {
    status.textContent = `State request failed with ${response.status}`;
    return;
  }

  const state = (await response.json()) as ServerState;
  ensureMaskDimensions(state.mask.width, state.mask.height);
  playerViewScale = state.playerView?.scale ?? 1;
  playerViewOffsetX = state.playerView?.offsetX ?? 0;
  playerViewOffsetY = state.playerView?.offsetY ?? 0;

  if (!state.activeMap) {
    mapImage = null;
    activeMapID = '';
    render();
    status.textContent = 'Waiting for a map.';
    return;
  }

  if (state.activeMap.id === activeMapID && mapImage) {
    render();
    status.textContent = `Ready: ${state.activeMap.name}`;
    return;
  }

  try {
    mapImage = await loadMapImage(state.activeMap.url, state.activeMap.id);
    activeMapID = state.activeMap.id;
    updateMapTexture(mapImage);
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
  gl.viewport(0, 0, canvas.width, canvas.height);
  render();
}

function render() {
  gl.clearColor(0.02, 0.03, 0.06, 1);
  gl.clear(gl.COLOR_BUFFER_BIT);

  const mapWidth = mapImage?.width ?? 1;
  const mapHeight = mapImage?.height ?? 1;
  const frame = contain(mapWidth, mapHeight, canvas.width, canvas.height);

  const scaledWidth = frame.width * playerViewScale;
  const scaledHeight = frame.height * playerViewScale;
  const viewX = frame.x + playerViewOffsetX - (scaledWidth - frame.width) / 2;
  const viewY = frame.y + playerViewOffsetY - (scaledHeight - frame.height) / 2;

  const scaleX = Math.max(scaledWidth / canvas.width, 0.0001);
  const scaleY = Math.max(scaledHeight / canvas.height, 0.0001);
  const offsetX = viewX / canvas.width;
  // Shader Y coordinates are bottom-origin, while our view math is top-origin.
  // Convert the top-left viewport Y into the shader's bottom-edge offset.
  const offsetY = (canvas.height - viewY - scaledHeight) / canvas.height;

  gl.useProgram(program);

  gl.uniform2f(mapScaleLoc, scaleX, scaleY);
  gl.uniform2f(mapOffsetLoc, offsetX, offsetY);

  gl.activeTexture(gl.TEXTURE0);
  gl.bindTexture(gl.TEXTURE_2D, mapTexture);
  gl.uniform1i(mapSamplerLoc, 0);

  gl.activeTexture(gl.TEXTURE1);
  gl.bindTexture(gl.TEXTURE_2D, maskTexture);
  gl.uniform1i(maskSamplerLoc, 1);

  gl.drawArrays(gl.TRIANGLES, 0, 6);
}

function updateMapTexture(image: HTMLImageElement) {
  gl.bindTexture(gl.TEXTURE_2D, mapTexture);
  gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, image);
}

function updateMaskTexture(mask: Uint8Array) {
  gl.bindTexture(gl.TEXTURE_2D, maskTexture);
  gl.texImage2D(gl.TEXTURE_2D, 0, gl.LUMINANCE, maskWidth, maskHeight, 0, gl.LUMINANCE, gl.UNSIGNED_BYTE, mask);
}

function updateMaskTexturePatch(x: number, y: number, width: number, height: number, data: Uint8Array) {
  gl.bindTexture(gl.TEXTURE_2D, maskTexture);
  gl.texSubImage2D(gl.TEXTURE_2D, 0, x, y, width, height, gl.LUMINANCE, gl.UNSIGNED_BYTE, data);
}

function ensureMaskDimensions(width: number, height: number) {
  if (width <= 0 || height <= 0) {
    return;
  }

  if (width > MAX_MASK_DIMENSION || height > MAX_MASK_DIMENSION) {
    status.textContent = `Mask size ${width}x${height} is unsupported on this device (max ${MAX_MASK_DIMENSION}x${MAX_MASK_DIMENSION}).`;
    return;
  }

  if (width === maskWidth && height === maskHeight) {
    return;
  }

  maskWidth = width;
  maskHeight = height;
  latestMask = new Uint8Array(maskWidth * maskHeight);
  updateMaskTexture(latestMask);
}

function decodeMaskPatch(payload: Uint8Array) {
  const headerSize = 17;
  if (payload.length < headerSize || payload[0] !== MASK_PATCH_MESSAGE_TYPE) {
    return null;
  }

  const view = new DataView(payload.buffer, payload.byteOffset, payload.byteLength);
  const x = view.getUint32(1, true);
  const y = view.getUint32(5, true);
  const width = view.getUint32(9, true);
  const height = view.getUint32(13, true);

  if (width === 0 || height === 0 || x + width > maskWidth || y + height > maskHeight) {
    return null;
  }

  const expected = headerSize + width * height;
  if (payload.length !== expected) {
    return null;
  }

  return {
    x,
    y,
    width,
    height,
    data: payload.subarray(headerSize)
  };
}

function createTexture(context: WebGLRenderingContext) {
  const texture = context.createTexture();
  if (!texture) {
    throw new Error('Failed to create texture');
  }

  context.bindTexture(context.TEXTURE_2D, texture);
  context.texParameteri(context.TEXTURE_2D, context.TEXTURE_WRAP_S, context.CLAMP_TO_EDGE);
  context.texParameteri(context.TEXTURE_2D, context.TEXTURE_WRAP_T, context.CLAMP_TO_EDGE);
  context.texParameteri(context.TEXTURE_2D, context.TEXTURE_MIN_FILTER, context.LINEAR);
  context.texParameteri(context.TEXTURE_2D, context.TEXTURE_MAG_FILTER, context.LINEAR);

  context.texImage2D(
    context.TEXTURE_2D,
    0,
    context.RGBA,
    1,
    1,
    0,
    context.RGBA,
    context.UNSIGNED_BYTE,
    new Uint8Array([5, 7, 13, 255])
  );

  return texture;
}

function createProgram(context: WebGLRenderingContext, vertex: string, fragment: string) {
  const vertexShader = compileShader(context, context.VERTEX_SHADER, vertex);
  const fragmentShader = compileShader(context, context.FRAGMENT_SHADER, fragment);

  const shaderProgram = context.createProgram();
  if (!shaderProgram) {
    throw new Error('Failed to create shader program');
  }

  context.attachShader(shaderProgram, vertexShader);
  context.attachShader(shaderProgram, fragmentShader);
  context.linkProgram(shaderProgram);

  if (!context.getProgramParameter(shaderProgram, context.LINK_STATUS)) {
    const info = context.getProgramInfoLog(shaderProgram) || 'unknown linker error';
    throw new Error(`WebGL link error: ${info}`);
  }

  return shaderProgram;
}

function compileShader(context: WebGLRenderingContext, type: number, source: string) {
  const shader = context.createShader(type);
  if (!shader) {
    throw new Error('Failed to create shader');
  }

  context.shaderSource(shader, source);
  context.compileShader(shader);

  if (!context.getShaderParameter(shader, context.COMPILE_STATUS)) {
    const info = context.getShaderInfoLog(shader) || 'unknown compile error';
    throw new Error(`WebGL shader compile error: ${info}`);
  }

  return shader;
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

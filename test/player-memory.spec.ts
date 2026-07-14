import { test, expect } from '@playwright/test';
import { spawn, spawnSync, type ChildProcess } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

const ROOT = path.resolve(__dirname, '..');
const PORT = Number(process.env.FOGCAST_E2E_PORT ?? '18086');
const BASE_URL = `http://127.0.0.1:${PORT}`;
const DATA_DIR = fs.mkdtempSync(path.join(os.tmpdir(), 'fogcast-e2e-'));
const BINARY_PATH = process.env.FOGCAST_E2E_BINARY ?? path.join(ROOT, 'bin', 'fogcast');
const SHOULD_BUILD_BINARY = (process.env.FOGCAST_E2E_BUILD ?? '1') !== '0';
const SUMMARY_PATH = process.env.FOGCAST_E2E_SUMMARY_PATH ?? path.join(ROOT, 'test-results', 'e2e-player-memory-summary.json');

const LARGE_MAP_MAX_MB = Number(process.env.FOGCAST_E2E_GOBLIN_MAX_MB ?? '140');
const SMALL_MAP_MAX_MB = Number(process.env.FOGCAST_E2E_TOWN_MAX_MB ?? '120');

const GOBLIN_LAIR_PATH = path.join(ROOT, 'test', '119 - Goblin Lair-PC-grid.jpg');
const TOWN_NEIGHBORHOOD_PATH = path.join(ROOT, 'test', '4 - Town Neighborhood Map.jpg');

let serverProcess: ChildProcess | null = null;
let serverLogs = '';

type MapStageResult = {
  stage: string;
  mapId: string;
  mapName: string;
  samplesMb: number[];
  medianMb: number;
  peakMb: number;
  thresholdMb: number;
};

function ensureBinary() {
  if (!SHOULD_BUILD_BINARY && fs.existsSync(BINARY_PATH)) {
    return;
  }

  fs.mkdirSync(path.dirname(BINARY_PATH), { recursive: true });
  const build = spawnSync('go', ['build', '-o', BINARY_PATH, './cmd/fogcast'], {
    cwd: ROOT,
    stdio: 'inherit',
  });

  if (build.status !== 0) {
    throw new Error('Failed to build FogCast binary for E2E memory test.');
  }
}

async function waitForServerReady() {
  const deadline = Date.now() + 30000;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(`${BASE_URL}/api/state`);
      if (response.ok) {
        return;
      }
    } catch {
      // Retry until deadline.
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }

  throw new Error(`Server readiness check failed for ${BASE_URL}/api/state`);
}

async function uploadMap(mapPath: string) {
  const payload = fs.readFileSync(mapPath);
  const fileName = path.basename(mapPath);

  const form = new FormData();
  form.set('autoSync', 'true');
  form.set('autoShroudAll', 'true');
  form.set('map', new File([payload], fileName, { type: 'image/jpeg' }));

  const response = await fetch(`${BASE_URL}/api/map`, {
    method: 'POST',
    body: form,
  });

  if (!response.ok) {
    const body = await response.text();
    throw new Error(`Map upload failed for ${fileName}: HTTP ${response.status} ${body}`);
  }

  const uploaded = (await response.json()) as { id: string; name: string };
  return uploaded;
}

function median(values: number[]) {
  const sorted = [...values].sort((a, b) => a - b);
  const mid = Math.floor(sorted.length / 2);
  if (sorted.length % 2 === 0) {
    return (sorted[mid - 1] + sorted[mid]) / 2;
  }
  return sorted[mid];
}

test.beforeAll(async () => {
  if (!fs.existsSync(GOBLIN_LAIR_PATH)) {
    throw new Error(`Missing fixture map: ${GOBLIN_LAIR_PATH}`);
  }
  if (!fs.existsSync(TOWN_NEIGHBORHOOD_PATH)) {
    throw new Error(`Missing fixture map: ${TOWN_NEIGHBORHOOD_PATH}`);
  }

  ensureBinary();

  fs.mkdirSync(path.join(DATA_DIR, 'maps'), { recursive: true });

  const proc = spawn(BINARY_PATH, [], {
    cwd: ROOT,
    env: {
      ...process.env,
      FOGCAST_ADDR: `127.0.0.1:${PORT}`,
      FOGCAST_DATA_DIR: DATA_DIR,
      FOGCAST_STATIC_DIR: path.join(ROOT, 'static'),
      FOGCAST_CONTAINER: '0',
    },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  serverProcess = proc;

  if (!proc.stdout || !proc.stderr) {
    throw new Error('Failed to attach FogCast server logs for E2E test process.');
  }

  proc.stdout.on('data', (chunk: Buffer) => {
    serverLogs += chunk.toString();
  });
  proc.stderr.on('data', (chunk: Buffer) => {
    serverLogs += chunk.toString();
  });

  await waitForServerReady();
});

test.afterAll(async () => {
  if (serverProcess && !serverProcess.killed) {
    serverProcess.kill('SIGTERM');
  }

  await new Promise((resolve) => setTimeout(resolve, 250));

  fs.rmSync(DATA_DIR, { recursive: true, force: true });
});

test('player memory stays within thresholds when switching large/small maps', async ({ page }) => {
  const session = await page.context().newCDPSession(page);
  await session.send('Runtime.enable');

  await page.goto(`${BASE_URL}/player`, { waitUntil: 'domcontentloaded' });
  await expect(page.locator('#fogcast-canvas')).toBeVisible();

  const runStage = async (stageName: string, mapPath: string, thresholdMb: number): Promise<MapStageResult> => {
    const uploaded = await uploadMap(mapPath);

    await expect
      .poll(async () => {
        const stateResponse = await fetch(`${BASE_URL}/api/player/state`);
        if (!stateResponse.ok) {
          return '';
        }
        const state = (await stateResponse.json()) as { activeMap: null | { id: string } };
        return state.activeMap?.id ?? '';
      }, { timeout: 30000 })
      .toBe(uploaded.id);

    await expect
      .poll(async () => {
        const status = await page.locator('#status').textContent();
        return status ?? '';
      }, { timeout: 30000 })
      .toContain('Ready:');

    const samplesMb: number[] = [];
    for (let i = 0; i < 7; i += 1) {
      const heapUsage = await session.send('Runtime.getHeapUsage');
      let usedHeapBytes = heapUsage.usedSize ?? 0;
      if (!usedHeapBytes || usedHeapBytes <= 0) {
        const { metrics } = await session.send('Performance.getMetrics');
        usedHeapBytes = metrics.find((metric) => metric.name === 'JSHeapUsedSize')?.value ?? 0;
      }
      samplesMb.push(usedHeapBytes / 1024 / 1024);
      await page.waitForTimeout(250);
    }

    const stageMedianMb = median(samplesMb);
    const stagePeakMb = Math.max(...samplesMb);

    return {
      stage: stageName,
      mapId: uploaded.id,
      mapName: uploaded.name,
      samplesMb,
      medianMb: stageMedianMb,
      peakMb: stagePeakMb,
      thresholdMb,
    };
  };

  const goblinStage = await runStage('goblin-lair', GOBLIN_LAIR_PATH, LARGE_MAP_MAX_MB);
  const townStage = await runStage('town-neighborhood', TOWN_NEIGHBORHOOD_PATH, SMALL_MAP_MAX_MB);

  const summary = {
    result: 'pass',
    thresholds: {
      goblinLairMaxMb: LARGE_MAP_MAX_MB,
      townNeighborhoodMaxMb: SMALL_MAP_MAX_MB,
    },
    stages: [goblinStage, townStage],
  };

  fs.mkdirSync(path.dirname(SUMMARY_PATH), { recursive: true });
  fs.writeFileSync(SUMMARY_PATH, JSON.stringify(summary, null, 2));

  let failed = false;
  let failMessage = '';
  if (goblinStage.peakMb > LARGE_MAP_MAX_MB) {
    failed = true;
    failMessage += `Goblin Lair peak ${goblinStage.peakMb.toFixed(2)} MB exceeds ${LARGE_MAP_MAX_MB} MB. `;
  }
  if (townStage.peakMb > SMALL_MAP_MAX_MB) {
    failed = true;
    failMessage += `Town Neighborhood peak ${townStage.peakMb.toFixed(2)} MB exceeds ${SMALL_MAP_MAX_MB} MB. `;
  }

  if (failed) {
    const failedSummary = {
      ...summary,
      result: 'fail',
      reason: failMessage.trim(),
      serverLogs,
    };
    fs.writeFileSync(SUMMARY_PATH, JSON.stringify(failedSummary, null, 2));
    throw new Error(failMessage.trim());
  }
});

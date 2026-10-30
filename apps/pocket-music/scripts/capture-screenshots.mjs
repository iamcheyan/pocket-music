// Captures the documentation screenshot matrix from a running Pocket Music
// instance and audits each page for horizontal overflow at every viewport and
// DPI. Usage:
//
//   node scripts/capture-screenshots.mjs [--base http://127.0.0.1:8765] [--out ../docs/screenshots] [--audit-only]
//
// It drives headless Chrome over the DevTools protocol so the reported pixel
// size, the emulated viewport and the device scale factor all agree.
import { spawn } from 'node:child_process';
import { mkdtemp, mkdir, writeFile, rm } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname, resolve } from 'node:path';

const args = process.argv.slice(2);
const flag = (name, fallback) => {
  const index = args.indexOf(name);
  return index >= 0 ? args[index + 1] : fallback;
};
const base = flag('--base', process.env.PM_BASE || 'http://127.0.0.1:8765');
const outDir = resolve(flag('--out', join(dirname(new URL(import.meta.url).pathname), '..', 'docs', 'screenshots')));
const auditOnly = args.includes('--audit-only');
const port = Number(flag('--port', 9333));

class CDP {
  constructor(url) {
    this.url = url;
    this.nextId = 1;
    this.pending = new Map();
    this.sessions = new Map();
  }

  async connect() {
    this.socket = new WebSocket(this.url);
    this.socket.addEventListener('message', (event) => {
      const message = JSON.parse(event.data);
      if (message.id && this.pending.has(message.id)) {
        const { resolve: done, reject } = this.pending.get(message.id);
        this.pending.delete(message.id);
        if (message.error) reject(new Error(`${message.error.message}`));
        else done(message.result);
      }
    });
    await new Promise((done, fail) => {
      this.socket.addEventListener('open', done, { once: true });
      this.socket.addEventListener('error', fail, { once: true });
    });
  }

  send(method, params = {}, sessionId) {
    const id = this.nextId++;
    const payload = { id, method, params };
    if (sessionId) payload.sessionId = sessionId;
    this.socket.send(JSON.stringify(payload));
    return new Promise((done, reject) => this.pending.set(id, { resolve: done, reject }));
  }
}

async function chromeBinary() {
  for (const candidate of ['google-chrome', 'chromium', 'chromium-browser']) {
    const which = spawn('which', [candidate], { stdio: ['ignore', 'pipe', 'ignore'] });
    const code = await new Promise((done) => which.on('exit', done));
    if (code === 0) return candidate;
  }
  throw new Error('no Chrome/Chromium binary found');
}

const sleep = (ms) => new Promise((done) => setTimeout(done, ms));

async function waitForDebugger(userDataDir) {
  for (let attempt = 0; attempt < 120; attempt += 1) {
    const file = join(userDataDir, 'DevToolsActivePort');
    if (existsSync(file)) {
      const [debugPort] = (await import('node:fs')).readFileSync(file, 'utf8').split('\n');
      if (debugPort) return Number(debugPort);
    }
    await sleep(100);
  }
  throw new Error('Chrome did not expose a DevTools port');
}

async function seed(baseUrl) {
  const post = async (path, body) => {
    const response = await fetch(`${baseUrl}${path}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    if (!response.ok) throw new Error(`${path}: ${response.status}`);
    return response.json();
  };
  // A track is playing so the Now Playing, lyrics and player-bar states are real.
  await post('/api/play', { track_id: 'mock-011', queue: ['mock-021', 'mock-034', 'mock-047', 'mock-058'], mode: 'repeat_all' });
  await post('/api/player/seek', { position: 44 });
  const queue = await fetch(`${baseUrl}/api/player/state`).then((r) => r.json());
  return queue;
}

// Every capture is a URL plus a proof selector that must render text.
const SHOTS = [
  { name: 'home', url: '/?view=home', proof: '.hero h1', themes: ['dark', 'light'] },
  { name: 'library', url: '/?view=library&tab=songs', proof: '.track-row', themes: ['dark', 'light'] },
  { name: 'albums', url: '/?view=library&tab=albums', proof: '.card-album', themes: ['dark', 'light'] },
  { name: 'artists', url: '/?view=library&tab=artists', proof: '.card-artist', themes: ['dark'] },
  { name: 'favorites', url: '/?view=library&tab=favorites', proof: '.track-row', themes: ['dark'] },
  { name: 'genres', url: '/?view=library&tab=genres', proof: '.chip', themes: ['dark'] },
  { name: 'folders', url: '/?view=library&tab=folders', proof: '.folder-row', exists: true, themes: ['dark'] },
  { name: 'album-detail', url: '/?view=album&id=1', proof: '.detail-hero h1', themes: ['dark'] },
  { name: 'artist-detail', url: '/?view=artist&id=1', proof: '.artist-hero h1', themes: ['dark'] },
  { name: 'now-playing', url: '/?view=now', proof: '.now-detail h1', themes: ['dark', 'light'] },
  { name: 'queue', url: '/?view=queue', proof: '.track-row', themes: ['dark'] },
  { name: 'playlists', url: '/?view=playlists', proof: '.card', themes: ['dark'] },
  { name: 'playlist', url: '/?view=playlist&id=1', proof: '.playlist-cover', exists: true, themes: ['dark', 'light'] },
  { name: 'search', url: '/?view=search&q=Vesper', proof: '.track-row', themes: ['dark', 'light'] },
  { name: 'settings', url: '/?view=settings', proof: '.settings-panel', themes: ['dark', 'light'] },
  { name: 'stats', url: '/?view=stats', proof: '.stat-grid', themes: ['dark'] },
];

const AUDIT = `
(() => {
  const doc = document.documentElement;
  const vw = window.innerWidth;
  const offenders = [];
  for (const el of document.querySelectorAll('body *')) {
    const style = getComputedStyle(el);
    if (style.position === 'fixed' || style.display === 'none' || style.visibility === 'hidden') continue;
    const rect = el.getBoundingClientRect();
    if (rect.width === 0 || rect.height === 0) continue;
    // A closed off-canvas drawer is translated fully out of view: it cannot
    // scroll the document, so ignore it rather than reporting a false positive.
    if (rect.right <= 0) continue;
    // Content inside a horizontal scroller (tab rail, carousel) is reachable by
    // scrolling that element, so it is not a document-level layout overflow.
    let scoped = false;
    for (let node = el.parentElement; node && node !== document.body; node = node.parentElement) {
      const overflowX = getComputedStyle(node).overflowX;
      if ((overflowX === 'auto' || overflowX === 'scroll') && node.scrollWidth > node.clientWidth) {
        scoped = true;
        break;
      }
    }
    if (scoped) continue;
    if (rect.right > vw + 1 || rect.left < -1) {
      offenders.push((el.className && String(el.className).split(' ')[0]) || el.tagName.toLowerCase());
      if (offenders.length >= 6) break;
    }
  }
  return {
    scrollWidth: doc.scrollWidth,
    clientWidth: doc.clientWidth,
    innerWidth: vw,
    innerHeight: window.innerHeight,
    scale: window.devicePixelRatio,
    overflow: offenders,
  };
})()
`;

const main = async () => {
  const userDataDir = await mkdtemp(join(tmpdir(), 'pm-chrome-'));
  const binary = await chromeBinary();
  const chrome = spawn(
    binary,
    [
      '--headless=new',
      '--no-sandbox',
      '--disable-gpu',
      '--hide-scrollbars',
      '--force-color-profile=srgb',
      '--disable-lcd-text',
      '--no-first-run',
      '--disable-extensions',
      '--user-data-dir=' + userDataDir,
      '--remote-debugging-port=0',
      'about:blank',
    ],
    { stdio: 'ignore' },
  );

  try {
    const debugPort = await waitForDebugger(userDataDir);
    const version = await fetch(`http://127.0.0.1:${debugPort}/json/version`).then((r) => r.json());
    const cdp = new CDP(version.webSocketDebuggerUrl);
    await cdp.connect();

    if (!auditOnly) {
      await mkdir(outDir, { recursive: true });
    }
    const playback = await seed(base);
    console.log(`seeded playback: ${playback.track?.title ?? 'none'} with ${playback.queue?.length ?? 0} queued`);

    const results = [];
    const plan = [];
    const add = (label, w, h, names, themes = ['dark']) => {
      for (const name of names) {
        const shot = SHOTS.find((item) => item.name === name);
        if (!shot) throw new Error(`unknown view ${name}`);
        for (const theme of themes) {
          if (theme === 'light' && !shot.themes.includes('light')) continue;
          plan.push({ shot, size: { w, h, scale: 1, label }, theme });
        }
      }
    };
    const all = SHOTS.map((shot) => shot.name);
    add('1280x720', 1280, 720, all);
    add('1280x720', 1280, 720, ['home', 'library', 'albums', 'now-playing', 'playlist', 'search', 'settings'], ['light']);
    add('1920x1080', 1920, 1080, ['home', 'now-playing']);
    add('720x720', 720, 720, ['home', 'now-playing', 'settings', 'albums']);
    add('720x720', 720, 720, ['home'], ['light']);
    add('640x480', 640, 480, ['home', 'library', 'now-playing']);
    add('480x320', 480, 320, ['home', 'library', 'now-playing', 'settings']);
    add('480x320', 480, 320, ['home'], ['light']);
    add('320x240', 320, 240, ['home', 'library', 'now-playing', 'queue']);
    add('320x240', 320, 240, ['home'], ['light']);

    const { targetId } = await cdp.send('Target.createTarget', { url: 'about:blank' });
    const { sessionId } = await cdp.send('Target.attachToTarget', { targetId, flatten: true });
    await cdp.send('Page.enable', {}, sessionId);
    await cdp.send('Runtime.enable', {}, sessionId);

    for (const item of plan) {
      const { shot, size, theme } = item;
      const url = `${base}${shot.url}${shot.url.includes('?') ? '&' : '?'}theme=${theme}&lang=en`;
      await cdp.send('Emulation.setDeviceMetricsOverride', { width: size.w, height: size.h, deviceScaleFactor: size.scale, mobile: false }, sessionId);
      await cdp.send('Page.navigate', { url }, sessionId);
      await sleep(900);
      // Wait for the proof element and the API-driven content behind it.
      let proof = null;
      for (let attempt = 0; attempt < 40; attempt += 1) {
        const expression = shot.exists
          ? `(() => { const el = document.querySelector(${JSON.stringify(shot.proof)}); return el ? 'present' : null })()`
          : `(() => { const el = document.querySelector(${JSON.stringify(shot.proof)}); return el ? (el.textContent || '').trim().slice(0, 60) : null })()`;
        const { result } = await cdp.send('Runtime.evaluate', { expression, returnByValue: true }, sessionId);
        proof = result.value;
        if (proof) break;
        await sleep(120);
      }
      const { result: audit } = await cdp.send('Runtime.evaluate', { expression: AUDIT, returnByValue: true }, sessionId);
      const file = `${shot.name}-${size.label}-${theme}.png`;
      if (!auditOnly) {
        const { data } = await cdp.send('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false }, sessionId);
        await writeFile(join(outDir, file), Buffer.from(data, 'base64'));
      }
      results.push({ file, view: shot.name, size: size.label, theme, proof, audit: audit.value });
      const bad = audit.value.scrollWidth > audit.value.clientWidth + 1 || audit.value.overflow.length > 0;
      console.log(`${bad ? 'OVERFLOW' : 'ok      '} ${size.label.padEnd(9)} ${theme.padEnd(5)} ${shot.name.padEnd(14)} proof=${proof ? 'yes' : 'NO'} ${bad ? JSON.stringify(audit.value) : ''}`);
    }

    // Device pixel ratio coverage: the layout must not depend on 1x rendering.
    for (const scale of [1.5, 2, 3]) {
      await cdp.send('Emulation.setDeviceMetricsOverride', { width: 1280, height: 720, deviceScaleFactor: scale, mobile: false }, sessionId);
      await cdp.send('Page.navigate', { url: `${base}/?view=home&theme=dark&lang=en` }, sessionId);
      await sleep(900);
      const { result: audit } = await cdp.send('Runtime.evaluate', { expression: AUDIT, returnByValue: true }, sessionId);
      results.push({ file: `home-1280x720-dark-${scale}x.png`, view: 'home', size: `1280x720-${scale}x`, theme: 'dark', proof: 'dpi', audit: audit.value });
      if (!auditOnly) {
        const { data } = await cdp.send('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false }, sessionId);
        await writeFile(join(outDir, `home-1280x720-dark-${scale}x.png`), Buffer.from(data, 'base64'));
      }
      console.log(`dpi ${scale}x  css=${audit.value.innerWidth}x${audit.value.innerHeight} dpr=${audit.value.scale} scrollWidth=${audit.value.scrollWidth} overflow=${audit.value.overflow.length}`);
    }

    await cdp.send('Target.closeTarget', { targetId });
    if (!auditOnly) {
      await writeFile(join(outDir, 'capture-report.json'), JSON.stringify(results, null, 2) + '\n');
    }
    const failures = results.filter((item) => !item.proof || item.audit.scrollWidth > item.audit.clientWidth + 1 || item.audit.overflow.length > 0);
    console.log(`\ncaptured ${results.length} pages, ${failures.length} layout failures`);
    if (failures.length > 0) {
      for (const failure of failures) console.log('FAIL', failure.size, failure.theme, failure.view, JSON.stringify(failure.audit.overflow));
      process.exitCode = 1;
    }
  } finally {
    chrome.kill('SIGKILL');
    for (let attempt = 0; attempt < 10; attempt += 1) {
      try {
        await rm(userDataDir, { recursive: true, force: true });
        break;
      } catch {
        await sleep(200);
      }
    }
  }
};

await main();

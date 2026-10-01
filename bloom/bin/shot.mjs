#!/usr/bin/env node
// Headless full-page screenshots of the local web app (light + dark) or of artboard HTML files, over Chrome CDP.
// No npm deps (Node >= 22 global WebSocket).
//
// App screens (logs in with a real OTP read from the local ritme_dev DB — no bypass):
//   node bloom/bin/shot.mjs --out docs/qa/bloom/B-N1-06 --mobile 09900000001 /fa /fa/calendar
// Artboards (no login), for side-by-side comparison:
//   node bloom/bin/shot.mjs --out docs/qa/bloom/B-N1-06/artboards --files docs/design/night-bloom/b1-cycle-log-analysis/nbl_Home.dc.html
// Options: --themes light,dark (default both; ignored with --files) --width 390 --height 844 --wait 5000
//          --base http://localhost:3000 --api http://127.0.0.1:8020/api/v1 --db ritme_dev --viewport (no full page)
//          --token <jwt> (skip OTP login)
//          --click 'sel||text=Label||text*=Part' (after load, click each step in order — CSS selector, exact
//          or partial button/link/tab text — waiting 1s between; `wait=3000` pauses longer) --name <suffix>
//          (file = <slug>.<suffix>.<theme>.png) --fake-media (Chrome's fake microphone + auto-granted permission,
//          for the voice recorder, B-N3-05)
// Admin panel (admin-web; desktop 1440×900, theme key ritme_admin_theme, tall-viewport capture — no clip, so RTL
// pages are not cut off). Signs in through the admin API on the same origin (session + CSRF cookies):
//   node bloom/bin/shot.mjs --admin --out docs/qa/bloom/B-N2-09 --admin-email qa@ritme.local --admin-password '…' /plus/plans
//   --base http://localhost:3001 (default with --admin) --admin-api /api/admin/v1 (path on --base, or a full URL)
import { spawn, execFileSync } from 'node:child_process';
import { mkdirSync, writeFileSync, mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve, basename } from 'node:path';

const args = process.argv.slice(2);
const opt = { themes: 'light,dark', width: '390', height: '844', wait: '5000', base: 'http://localhost:3000',
  api: 'http://127.0.0.1:8020/api/v1', db: 'ritme_dev' };
const flags = new Set(); const targets = [];
for (let i = 0; i < args.length; i++) {
  const a = args[i];
  if (['--files', '--viewport', '--admin', '--fake-media'].includes(a)) flags.add(a.slice(2));
  else if (a.startsWith('--')) opt[a.slice(2)] = args[++i];
  else targets.push(a);
}
const ADMIN = flags.has('admin');
if (ADMIN) {
  if (!args.includes('--width')) opt.width = '1440';
  if (!args.includes('--height')) opt.height = '900';
  if (!args.includes('--base')) opt.base = 'http://localhost:3001';
}
if (!opt.out || !targets.length) { console.error('usage: shot.mjs --out DIR [--mobile M | --token T | --files] targets…'); process.exit(2); }
mkdirSync(opt.out, { recursive: true });
const FILES = flags.has("files");
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const ROOT = resolve(new URL('../..', import.meta.url).pathname);

async function login(mobile) {
  const post = (p, b) => fetch(`${opt.api}${p}`, { method: 'POST', headers: { 'Content-Type': 'application/json', Accept: 'application/json' }, body: JSON.stringify(b) }).then((r) => r.json());
  await post('/auth/send-otp', { mobile });
  const code = execFileSync('docker', ['compose', '-f', join(ROOT, 'backend-go/docker-compose.test.yml'), 'exec', '-T', 'mariadb', 'mariadb',
    '-uritme', '-pritme', opt.db, '-N', '-e', `select code from otp_verifications where mobile='${mobile.replace(/\D/g, '')}' order by id desc limit 1`]).toString().trim();
  const res = await post('/auth/verify-otp', { mobile, code });
  const t = res?.data?.access_token; if (!t) throw new Error('login failed: ' + JSON.stringify(res).slice(0, 200));
  return t;
}

const token = opt.token || (opt.mobile && !flags.has('files') ? await login(opt.mobile) : null);
const port = 9300 + Math.floor(Math.random() * 600);
const chrome = spawn('/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', ['--headless=new', `--remote-debugging-port=${port}`,
  `--user-data-dir=${mkdtempSync(join(tmpdir(), 'shot-'))}`, '--hide-scrollbars', '--allow-file-access-from-files',
  ...(flags.has('fake-media') ? ['--use-fake-ui-for-media-stream', '--use-fake-device-for-media-stream'] : []), 'about:blank'], { stdio: 'ignore' });
let list; for (let i = 0; i < 60 && !list; i++) { try { list = await (await fetch(`http://127.0.0.1:${port}/json`)).json(); } catch { await sleep(250); } }
const ws = new WebSocket(list.find((t) => t.type === 'page').webSocketDebuggerUrl); await new Promise((r) => ws.addEventListener('open', r));
let id = 0; const pend = new Map(); const errors = [];
ws.addEventListener('message', (e) => { const m = JSON.parse(e.data); if (m.id && pend.has(m.id)) { pend.get(m.id)(m); pend.delete(m.id); }
  if (m.method === 'Runtime.exceptionThrown') errors.push(m.params.exceptionDetails?.exception?.description?.split('\n')[0]);
  if (m.method === 'Network.responseReceived' && m.params.response.url.includes('/api/') && m.params.response.status >= 400) errors.push(`${m.params.response.status} ${m.params.response.url}`); });
const send = (method, params = {}) => new Promise((r) => { const i = ++id; pend.set(i, r); ws.send(JSON.stringify({ id: i, method, params })); });
const ev = async (x) => (await send('Runtime.evaluate', { expression: x, awaitPromise: true, returnByValue: true })).result?.result?.value;
await send('Network.enable'); await send('Runtime.enable');
const MOBILE = !FILES && !ADMIN;
await send('Emulation.setDeviceMetricsOverride', { width: +opt.width, height: +opt.height, deviceScaleFactor: 2, mobile: MOBILE });

async function capture(file) {
  let clip;
  if (!flags.has('viewport')) {
    // Full page: grow the viewport to the tallest scroll container so inner scrollers are captured too.
    const h = await ev(`Math.max(document.documentElement.scrollHeight, ...[...document.querySelectorAll('*')].filter(e=>{const s=getComputedStyle(e);return /(auto|scroll)/.test(s.overflowY)&&e.scrollHeight>e.clientHeight}).map(e=>e.scrollHeight+e.getBoundingClientRect().top))`);
    const height = Math.min(Math.max(+opt.height, Math.ceil(h || 0)), 12000);
    await send('Emulation.setDeviceMetricsOverride', { width: +opt.width, height, deviceScaleFactor: 2, mobile: MOBILE }); await sleep(800);
    // Admin: the viewport is now as tall as the page — capture it as is (a clip from x=0 cuts RTL layouts).
    if (!ADMIN) clip = { x: 0, y: 0, width: +opt.width, height, scale: 1 };
  }
  const s = await send('Page.captureScreenshot', { format: 'png', ...(clip ? { clip, captureBeyondViewport: true } : {}) });
  writeFileSync(file, Buffer.from(s.result.data, 'base64'));
  await send('Emulation.setDeviceMetricsOverride', { width: +opt.width, height: +opt.height, deviceScaleFactor: 2, mobile: MOBILE });
}

const slug = (p) => (p.replace(/^\/+|\/+$/g, '').replace(/[^\w.-]+/g, '_') || 'root');
if (flags.has('files')) {
  for (const f of targets) {
    await send('Page.navigate', { url: 'file://' + resolve(f) }); await sleep(+opt.wait / 2);
    const out = join(opt.out, basename(f).replace(/\.dc\.html$|\.html$/, '') + '.png'); await capture(out); console.log('✔', out);
  }
} else {
  const base = opt.base;
  await send('Page.navigate', { url: ADMIN ? `${base}/login` : base }); await sleep(2500);
  if (token && !ADMIN) await ev(`localStorage.setItem('ritme-install-dismissed','1'); localStorage.setItem('ritme_token',${JSON.stringify(token)}); document.cookie='ritme_auth=1; path=/'; true`);
  if (ADMIN && opt['admin-email']) {
    // Same-origin login from the page, so the browser keeps the session + CSRF cookies the panel expects.
    const apiBase = opt['admin-api'] || '/api/admin/v1';
    const res = await ev(`fetch(${JSON.stringify(apiBase.replace(/\/+$/, '') + '/auth/login')}, { method: 'POST', credentials: 'include',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ email: ${JSON.stringify(opt['admin-email'])}, password: ${JSON.stringify(opt['admin-password'] ?? '')} }) })
      .then(async (r) => r.status + ' ' + ((await r.json().catch(() => ({})))?.data?.admin?.email ?? ''))`);
    if (!String(res).startsWith('200')) throw new Error('admin login failed: ' + res);
  }
  const themeKey = ADMIN ? 'ritme_admin_theme' : 'ritme_theme';
  for (const theme of opt.themes.split(',')) {
    await send('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: theme }] });
    await ev(`try{localStorage.setItem(${JSON.stringify(themeKey)},${JSON.stringify(theme)})}catch(e){}; document.documentElement.dataset.theme=${JSON.stringify(theme)}; true`);
    for (const p of targets) {
      errors.length = 0;
      await send('Page.navigate', { url: base + p }); await sleep(+opt.wait);
      for (const step of (opt.click ? opt.click.split('||') : [])) {
        if (/^wait=\d+$/.test(step)) { await sleep(+step.slice(5)); continue; }
        const r = await ev(`(() => { const s = ${JSON.stringify(step)}; let el;
          const pool = () => [...document.querySelectorAll('button,a,[role=tab],[role=button]')];
          if (s.startsWith('text=')) el = pool().find((e) => e.textContent.trim() === s.slice(5));
          else if (s.startsWith('text*=')) el = pool().find((e) => e.textContent.includes(s.slice(6)));
          else el = document.querySelector(s);
          if (!el) return 'missing ' + s; el.scrollIntoView({ block: 'center' }); el.click(); return 'ok'; })()`);
        if (r !== 'ok') console.log('   click:', r);
        await sleep(1000);
      }
      const out = join(opt.out, `${slug(p)}${opt.name ? '.' + opt.name : ''}.${theme}.png`); await capture(out);
      const info = await ev(`location.pathname + ' | ' + document.documentElement.dataset.theme + ' | ' + document.body.innerText.slice(0,60).replace(/\\s+/g,' ')`);
      console.log('✔', out, '→', info, errors.length ? '\n   errors: ' + [...new Set(errors)].join(' | ') : '');
    }
  }
}
ws.close(); chrome.kill(); process.exit(0);

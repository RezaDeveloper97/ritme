#!/usr/bin/env node
// Stage smoke over Chrome CDP: light + dark full-page screenshots of https://stage.ritmeapp.ir with a per-page record
// of 4xx/5xx responses, console errors and uncaught exceptions (summary.json + summary.md in --out).
// No npm deps (Node >= 22 global WebSocket). Reusable by every release task (B-N2-11 onwards).
//
// SECRETS: none live in this file or in its output. At runtime it reads `/root/ritme-stage-credentials.txt` over ssh
// (gate Basic auth + admin login) into memory only, trades the Basic auth for the nginx gate cookie `ritme_stage`, and
// reads OTP codes READ-ONLY from the stage MariaDB (`ritme-stage-mysql-1`, DB from the container env) — stage has no
// test-OTP mode. Phone numbers are masked in the page before every capture and in the logs (--no-mask to disable).
// Never point it at production.
//
// App screens, existing user (signs in through the real OTP flow):
//   node bloom/bin/stage-smoke.mjs --out docs/qa/bloom/n2-stage --mobile 09900000001 --prefix A /fa/home /fa/calendar
// Fresh user (random unused 0990… number, OTP sign-in; prints the masked number; onboarding is NOT completed — pass
// onboarding routes, or script it with the exported helpers):
//   node bloom/bin/stage-smoke.mjs --out DIR --new-user /fa/onboarding/name
// Admin panel (admin-web on /panel, admin creds from the same credentials file, desktop 1440×900):
//   node bloom/bin/stage-smoke.mjs --admin --out DIR /plus/subscriptions /plus/plans
// Options: --themes light,dark  --width 390  --height 844  --wait 5000  --viewport (no full page)  --prefix NAME
//          --base https://stage.ritmeapp.ir  --ssh root@89.251.8.115  --db-container ritme-stage-mysql-1
//          --token <jwt> (skip OTP)  --no-mask  --headful
// Output: <out>/<prefix>-<route>.<theme>.png and <out>/summary.{json,md} (merged across runs into the same --out).
//
// Scripted flows (clicks, forms): import the helpers instead of the CLI —
//   import { stage, openBrowser, Summary } from './bloom/bin/stage-smoke.mjs';
//   const s = await stage(); const tok = await s.login('0990…');           // gate cookie + OTP sign-in
//   const b = await openBrowser({ cookie: s.cookie }); await b.setToken(tok);
//   await b.goto('/fa/plus'); await b.click('ادامه'); await b.shot(dir, 'plus', 'light', summary);
import { spawn, execFileSync } from 'node:child_process';
import { mkdirSync, writeFileSync, readFileSync, existsSync, mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

export const DEFAULTS = { base: 'https://stage.ritmeapp.ir', ssh: 'root@89.251.8.115', dbContainer: 'ritme-stage-mysql-1' };
export const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
export const maskMobile = (m) => String(m).replace(/^(\d{4})\d+(\d{2})$/, '$1•••••$2');

/** Stage access: credentials (memory only), gate cookie, OTP sign-in, read-only SQL. */
export async function stage({ base = DEFAULTS.base, ssh = DEFAULTS.ssh, dbContainer = DEFAULTS.dbContainer } = {}) {
  const raw = execFileSync('ssh', ['-o', 'ConnectTimeout=15', ssh, 'cat /root/ritme-stage-credentials.txt']).toString();
  const lines = raw.split('\n');
  const gateLine = lines.find((l) => /basic/i.test(l)) ?? lines[0];
  const [gateUser, gatePass] = gateLine.slice(gateLine.lastIndexOf(':') + 1).trim().split(/\s*\/\s*/);
  const adm = lines.find((l) => /admin/i.test(l))?.match(/(\S+@\S+)\s*\/\s*(\S+)/);
  const creds = { admin: adm ? { email: adm[1], password: adm[2] } : null };
  const res = await fetch(base + '/', { redirect: 'manual', headers: { Authorization: 'Basic ' + Buffer.from(`${gateUser}:${gatePass}`).toString('base64') } });
  const sc = (res.headers.getSetCookie?.() ?? []).find((c) => c.startsWith('ritme_stage='));
  if (!sc) throw new Error(`gate cookie not issued (HTTP ${res.status})`);
  const cookie = { name: 'ritme_stage', value: sc.split(';')[0].slice('ritme_stage='.length) };
  const cookieHeader = `${cookie.name}=${cookie.value}`;

  /** Read-only SQL against the stage DB (SELECT/SHOW only). Returns rows of tab-separated columns. */
  const sql = (q) => {
    if (!/^\s*(select|show)\b/i.test(q) || /;\s*\S/.test(q)) throw new Error('read-only single SELECT/SHOW only');
    const inner = `mariadb -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" -N -e ${JSON.stringify(q)}`;
    const out = execFileSync('ssh', [ssh, `docker exec ${dbContainer} sh -c ${shQuote(inner)}`]).toString().trim();
    return out ? out.split('\n').map((r) => r.split('\t')) : [];
  };
  const api = async (method, path, body, token) => {
    const r = await fetch(`${base}/api/v1${path}`, { method, headers: { 'Content-Type': 'application/json', Accept: 'application/json',
      'Accept-Language': 'fa', Cookie: cookieHeader, ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: body === undefined ? undefined : JSON.stringify(body) });
    const text = await r.text(); let json; try { json = JSON.parse(text); } catch { json = { raw: text.slice(0, 300) }; }
    return { status: r.status, json };
  };
  const readOtp = async (mobile, after = 0) => {
    const m = String(mobile).replace(/\D/g, '');
    for (let i = 0; i < 20; i++) {
      const row = sql(`select id, code from otp_verifications where mobile='${m}' order by id desc limit 1`)[0];
      if (row && +row[0] > after) return row[1];
      await sleep(500);
    }
    throw new Error('no OTP row for ' + maskMobile(m));
  };
  const lastOtpId = (mobile) => +(sql(`select coalesce(max(id),0) from otp_verifications where mobile='${String(mobile).replace(/\D/g, '')}'`)[0]?.[0] ?? 0);
  const login = async (mobile) => {
    const before = lastOtpId(mobile);
    const s = await api('POST', '/auth/send-otp', { mobile });
    if (s.status >= 400) throw new Error(`send-otp ${s.status}: ${JSON.stringify(s.json).slice(0, 200)}`);
    const code = await readOtp(mobile, before);
    const v = await api('POST', '/auth/verify-otp', { mobile, code });
    const t = v.json?.data?.access_token; if (!t) throw new Error(`verify-otp ${v.status}`);
    return t;
  };
  /** A random 0990xxxxxxx number with no user and no OTP row yet. */
  const freshMobile = (prefix = '0990') => {
    for (;;) {
      const m = prefix + String(Math.floor(Math.random() * 10 ** (11 - prefix.length))).padStart(11 - prefix.length, '0');
      const n = sql(`select (select count(*) from users where mobile='${m}') + (select count(*) from otp_verifications where mobile='${m}')`)[0]?.[0];
      if (n === '0') return m;
    }
  };
  return { base, cookie, cookieHeader, creds, sql, api, readOtp, lastOtpId, login, freshMobile };
}
const shQuote = (s) => `'${s.replace(/'/g, `'\\''`)}'`;

/** Collects per-page findings; merged into <out>/summary.{json,md}. */
export class Summary {
  constructor(out) { this.out = out; this.file = join(out, 'summary.json'); this.rows = existsSync(this.file) ? JSON.parse(readFileSync(this.file, 'utf8')) : []; }
  add(row) { this.rows = this.rows.filter((r) => !(r.shot === row.shot)); this.rows.push(row); this.save(); }
  save() {
    mkdirSync(this.out, { recursive: true });
    writeFileSync(this.file, JSON.stringify(this.rows, null, 2));
    const md = ['| Page | Theme | Landed | Screenshot | 4xx/5xx | Console / exceptions |', '|---|---|---|---|---|---|',
      ...this.rows.map((r) => `| \`${r.route}\` | ${r.theme} | \`${r.landed}\` | ${r.shot} | ${r.network.join('<br>') || 'clean'} | ${r.console.join('<br>').replace(/\|/g, '\\|') || 'clean'} |`)];
    writeFileSync(join(this.out, 'summary.md'), md.join('\n') + '\n');
  }
}

/** Headless Chrome with gate cookie, theme emulation, error capture and small DOM helpers. */
export async function openBrowser({ cookie, base = DEFAULTS.base, width = 390, height = 844, mobile = true, headful = false, mask = true } = {}) {
  const port = 9300 + Math.floor(Math.random() * 600);
  const chrome = spawn('/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', [...(headful ? [] : ['--headless=new']),
    `--remote-debugging-port=${port}`, `--user-data-dir=${mkdtempSync(join(tmpdir(), 'stage-smoke-'))}`, '--hide-scrollbars', 'about:blank'], { stdio: 'ignore' });
  let list; for (let i = 0; i < 80 && !list; i++) { try { list = await (await fetch(`http://127.0.0.1:${port}/json`)).json(); } catch { await sleep(250); } }
  const ws = new WebSocket(list.find((t) => t.type === 'page').webSocketDebuggerUrl); await new Promise((r) => ws.addEventListener('open', r));
  let id = 0; const pend = new Map(); const network = []; const consoleErr = []; const host = new URL(base).host;
  ws.addEventListener('message', (e) => {
    const m = JSON.parse(e.data); if (m.id && pend.has(m.id)) { pend.get(m.id)(m); pend.delete(m.id); }
    if (m.method === 'Runtime.exceptionThrown') consoleErr.push('exception: ' + (m.params.exceptionDetails?.exception?.description ?? m.params.exceptionDetails?.text ?? '').split('\n')[0]);
    if (m.method === 'Runtime.consoleAPICalled' && m.params.type === 'error') consoleErr.push('console.error: ' + m.params.args.map((a) => a.value ?? a.description ?? '').join(' ').split('\n')[0].slice(0, 200));
    if (m.method === 'Network.responseReceived' && m.params.response.status >= 400) {
      const u = new URL(m.params.response.url); if (u.host === host) network.push(`${m.params.response.status} ${m.params.type === 'Document' ? 'doc ' : ''}${u.pathname}`);
    }
  });
  const send = (method, params = {}) => new Promise((r) => { const i = ++id; pend.set(i, r); ws.send(JSON.stringify({ id: i, method, params })); });
  const ev = async (x) => { const r = await send('Runtime.evaluate', { expression: x, awaitPromise: true, returnByValue: true }); return r.result?.result?.value; };
  await send('Network.enable'); await send('Runtime.enable'); await send('Page.enable');
  if (cookie) await send('Network.setCookie', { ...cookie, domain: host, path: '/', secure: true, httpOnly: false });
  const metrics = (h = height) => send('Emulation.setDeviceMetricsOverride', { width, height: h, deviceScaleFactor: 2, mobile });
  await metrics();

  const b = {
    send, ev, network, consoleErr, base,
    reset() { network.length = 0; consoleErr.length = 0; },
    async goto(path, wait = 4500) { await send('Page.navigate', { url: path.startsWith('http') ? path : base + path }); await sleep(wait); },
    path: () => ev('location.pathname + location.search'),
    text: () => ev('document.body.innerText'),
    async setTheme(theme, key = 'ritme_theme') {
      await send('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: theme }] });
      await ev(`try{localStorage.setItem(${JSON.stringify(key)},${JSON.stringify(theme)})}catch(e){}; document.documentElement.dataset.theme=${JSON.stringify(theme)}; true`);
    },
    async setToken(token) {
      if ((await ev('location.origin')) !== new URL(base).origin) await b.goto('/fa/splash', 2500);
      await ev(`localStorage.setItem('ritme-install-dismissed','1'); localStorage.setItem('ritme_token',${JSON.stringify(token)}); document.cookie='ritme_auth=1; path=/'; true`);
    },
    /** Clicks the smallest visible button/link/label/radio whose text (or aria-label) contains `text`. */
    async click(text, { exact = false, nth = 0 } = {}) {
      const ok = await ev(`(() => { const t = ${JSON.stringify(text)};
        const els = [...document.querySelectorAll('button,a,label,[role=button],[role=radio],[role=tab],[role=switch],[role=menuitem],[role=option],[role=checkbox],input[type=checkbox]')]
          .filter((e) => { const r = e.getBoundingClientRect(); const s = ((e.innerText || '') + ' ' + (e.getAttribute('aria-label') || '')).trim();
            return r.width > 0 && r.height > 0 && !e.disabled && (${exact} ? (e.innerText||'').trim() === t || e.getAttribute('aria-label') === t : s.includes(t)); })
          .sort((a, b) => (a.innerText || '').length - (b.innerText || '').length);
        const el = els[${nth}]; if (!el) return false; el.scrollIntoView({ block: 'center' }); el.click(); return true; })()`);
      if (!ok) throw new Error(`click: "${text}" not found on ${await b.path()}`);
      await sleep(900);
    },
    async clickSel(sel, nth = 0) {
      const ok = await ev(`(() => { const el = [...document.querySelectorAll(${JSON.stringify(sel)})].filter(e=>!e.disabled)[${nth}]; if (!el) return false; el.scrollIntoView({block:'center'}); el.click(); return true; })()`);
      if (!ok) throw new Error(`clickSel: ${sel} not found on ${await b.path()}`);
      await sleep(900);
    },
    /** Focuses `sel` and types like a user (Input.insertText → React onChange fires). */
    async type(sel, value, { clear = true } = {}) {
      const ok = await ev(`(() => { const el = document.querySelector(${JSON.stringify(sel)}); if (!el) return false; el.scrollIntoView({block:'center'}); el.focus(); ${clear ? 'el.select && el.select();' : ''} return true; })()`);
      if (!ok) throw new Error(`type: ${sel} not found on ${await b.path()}`);
      await send('Input.insertText', { text: String(value) }); await sleep(400);
    },
    async waitFor(pred, timeout = 15000) {
      const t0 = Date.now();
      while (Date.now() - t0 < timeout) { if (await ev(`(() => { try { return !!(${pred}); } catch { return false; } })()`)) return true; await sleep(300); }
      return false;
    },
    has: (text) => ev(`document.body.innerText.includes(${JSON.stringify(text)})`),
    /** Masks Iranian mobile numbers (latin or Persian digits) in visible text and input values. */
    mask: () => ev(`(() => { const re = /(?:۰|0)?(۹|9)([۰-۹0-9][\\s\\u200c•]*){9}/g; const f = (s) => s.replace(re, (m) => m.slice(0, 4) + '•••••' + m.slice(-2));
      const w = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT); let n; while ((n = w.nextNode())) { if (re.test(n.nodeValue)) n.nodeValue = f(n.nodeValue); re.lastIndex = 0; }
      document.querySelectorAll('input').forEach((i) => { if (re.test(i.value)) { i.value = f(i.value); } re.lastIndex = 0; });
      const em = /[\\w.+-]+@([\\w-]+\\.)+[a-z]{2,}/gi; const w2 = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
      while ((n = w2.nextNode())) { if (em.test(n.nodeValue)) n.nodeValue = n.nodeValue.replace(em, (m) => '•••@' + m.split('@')[1]); em.lastIndex = 0; }
      return true; })()`),
    /** Full-page (inner scrollers included) or viewport capture to `file`. */
    async capture(file, { viewport = false, clip = mobile } = {}) {
      if (mask) await b.mask();
      let c;
      if (!viewport) {
        const h = await ev(`Math.max(document.documentElement.scrollHeight, ...[...document.querySelectorAll('*')].filter(e=>{const s=getComputedStyle(e);return /(auto|scroll)/.test(s.overflowY)&&e.scrollHeight>e.clientHeight}).map(e=>e.scrollHeight+e.getBoundingClientRect().top))`);
        const hh = Math.min(Math.max(height, Math.ceil(h || 0)), 12000);
        await metrics(hh); await sleep(800); if (mask) await b.mask();
        if (clip) c = { x: 0, y: 0, width, height: hh, scale: 1 };
      }
      const s = await send('Page.captureScreenshot', { format: 'png', ...(c ? { clip: c, captureBeyondViewport: true } : {}) });
      writeFileSync(file, Buffer.from(s.result.data, 'base64'));
      if (!viewport) await metrics();
    },
    /** Capture + record the page's network/console findings since the last reset() into `summary`. */
    async shot(out, name, theme, summary, opts = {}) {
      mkdirSync(out, { recursive: true });
      const file = `${name}.${theme}.png`; await b.capture(join(out, file), opts);
      const row = { route: opts.route ?? name, theme, landed: await b.path(), shot: file, network: [...new Set(network)], console: [...new Set(consoleErr)] };
      summary?.add(row); b.reset();
      console.log(`${row.network.length || row.console.length ? '!' : '✔'} ${file} → ${row.landed}${row.network.length ? '  net: ' + row.network.join(', ') : ''}${row.console.length ? '  console: ' + row.console.join(' | ') : ''}`);
      return row;
    },
    /** admin-web: same-origin session login from the page (keeps session + CSRF cookies). Returns the CSRF token. */
    async adminLogin({ email, password }, apiBase = '/api/admin/v1') {
      if (!(await ev('location.href')).includes('/panel')) await b.goto('/panel/login', 3000);
      const r = await ev(`fetch(${JSON.stringify(apiBase + '/auth/login')}, { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify({ email: ${JSON.stringify(email)}, password: ${JSON.stringify(password)} }) }).then(async (r) => ({ s: r.status, j: await r.json().catch(() => ({})) }))`);
      if (r?.s !== 200) throw new Error('admin login failed: HTTP ' + r?.s);
      return r.j?.data?.csrf_token ?? null;
    },
    close() { try { ws.close(); } catch { /* */ } chrome.kill(); },
  };
  return b;
}

export const slug = (p) => (p.replace(/^\/+|\/+$/g, '').replace(/[?=&]/g, '_').replace(/[^\w.-]+/g, '_') || 'root');

// ---------------------------------------------------------------- CLI
async function main() {
  const args = process.argv.slice(2);
  const opt = { themes: 'light,dark', width: '390', height: '844', wait: '5000', base: DEFAULTS.base, ssh: DEFAULTS.ssh, 'db-container': DEFAULTS.dbContainer };
  const flags = new Set(); const targets = [];
  for (let i = 0; i < args.length; i++) {
    const a = args[i];
    if (['--admin', '--viewport', '--new-user', '--no-mask', '--headful'].includes(a)) flags.add(a.slice(2));
    else if (a.startsWith('--')) opt[a.slice(2)] = args[++i];
    else targets.push(a);
  }
  if (!opt.out || !targets.length) { console.error('usage: stage-smoke.mjs --out DIR [--mobile M | --new-user | --token T | --admin] routes…  (see header)'); process.exit(2); }
  const ADMIN = flags.has('admin');
  if (ADMIN) { if (!args.includes('--width')) opt.width = '1440'; if (!args.includes('--height')) opt.height = '900'; }
  const s = await stage({ base: opt.base, ssh: opt.ssh, dbContainer: opt['db-container'] });
  let token = opt.token ?? null;
  if (!ADMIN && !token) {
    const m = flags.has('new-user') ? s.freshMobile() : opt.mobile;
    if (m) { token = await s.login(m); console.log('signed in', maskMobile(m)); }
  }
  const b = await openBrowser({ cookie: s.cookie, base: opt.base, width: +opt.width, height: +opt.height, mobile: !ADMIN, headful: flags.has('headful'), mask: !flags.has('no-mask') });
  const summary = new Summary(opt.out);
  const prefix = opt.prefix ? opt.prefix + '-' : (ADMIN ? 'ADM-' : '');
  try {
    if (ADMIN) {
      if (!s.creds.admin) throw new Error('no admin line in the credentials file');
      await b.goto('/panel/login', 3000); await b.adminLogin(s.creds.admin);
    } else if (token) await b.setToken(token);
    else await b.goto('/fa/splash', 2500);
    for (const theme of opt.themes.split(',')) {
      await b.setTheme(theme, ADMIN ? 'ritme_admin_theme' : 'ritme_theme');
      for (const p of targets) {
        b.reset();
        await b.goto(ADMIN && !p.startsWith('/panel') ? '/panel' + p : p, +opt.wait);
        await b.shot(opt.out, prefix + slug(p), theme, summary, { route: p, viewport: flags.has('viewport'), clip: !ADMIN });
      }
    }
  } finally { b.close(); }
  console.log('summary →', join(opt.out, 'summary.md'));
  process.exit(0);
}
if (import.meta.url === pathToFileURL(process.argv[1] ?? '').href) main().catch((e) => { console.error(String(e?.message ?? e)); process.exit(1); });

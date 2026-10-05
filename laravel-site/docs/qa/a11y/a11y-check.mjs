#!/usr/bin/env node
/**
 * a11y-check — repeatable WCAG 2.1 AA sweep of every public template (L9-03).
 *
 *   node docs/qa/a11y/a11y-check.mjs --base http://127.0.0.1:8000                  # every page, 390 + 1440
 *   node docs/qa/a11y/a11y-check.mjs --base http://127.0.0.1:8000 --only shop,cart --widths 390
 *   node docs/qa/a11y/a11y-check.mjs --base … --json docs/qa/a11y/results.json      # machine-readable summary
 *   AXE_PATH=/path/to/axe.min.js node docs/qa/a11y/a11y-check.mjs --base …
 *
 * Drives headless system Chrome over CDP (helpers from tools/critical.mjs, Node built-ins only). Per page and width:
 *  1. axe-core (WCAG 2.0/2.1 A + AA rules, best practices, `target-size` = 24 px) injected from a LOCAL file —
 *     never a CDN. Resolved from AXE_PATH, ./node_modules/axe-core or the monorepo's ../frontend/node_modules
 *     (the site does not add axe-core as a dependency). Without axe the custom checks still run.
 *     Text axe cannot judge (gradient / image backgrounds) is measured in pixels: glyphs hidden, line boxes captured,
 *     the 10th percentile of background pixels must reach 4.5:1 (3:1 large).
 *  2. Custom checks: `lang="fa"` + `dir="rtl"`, exactly one <h1>, no skipped heading level, one <main>, a skip link
 *     as the first tab stop whose target exists, every aria-invalid control described by a non-empty error, and a
 *     keyboard walk: Tab through the page (max 250 stops), every stop must be visible (non-zero box, not inside a
 *     hidden/inert/aria-hidden subtree) and show a focus indicator (style change versus the blurred state).
 *  3. Reduced motion: with `prefers-reduced-motion: reduce` no running motion animation/transition may last > 10 ms.
 *  4. Keyboard walkthroughs of the widgets (SCENARIOS): menu, accordion, calculators, filters, variants, cart stepper,
 *     join stepper, gallery lightbox.
 * Flows: `cart`/`checkout` add a product to the session cart first; `*-errors` submit the empty form so the error
 * state (aria-invalid / aria-describedby) is audited.
 * No page may request anything outside 127.0.0.1 (external requests are blocked and reported).
 *
 * Results: docs/qa/a11y/README.md. Exit: 0 = no violations · 1 = violations or failed walkthrough · 2 = setup error.
 */
import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { inflateSync } from 'node:zlib';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { Cdp, launchChrome, withTab } from '../../../tools/critical.mjs';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');

/** Pages: path, or discover (first link matching `pattern` on `from`), optional flow. */
const PAGES = [
    { key: 'home', path: '/' },
    { key: 'cycle', path: '/cycle' }, { key: 'ttc', path: '/ttc' }, { key: 'pregnancy', path: '/pregnancy' },
    { key: 'postpartum', path: '/postpartum' }, { key: 'menopause', path: '/menopause' }, { key: 'teen', path: '/teen' },
    { key: 'services', path: '/services' }, { key: 'plus', path: '/plus' }, { key: 'tools', path: '/tools' },
    { key: 'about', path: '/about' }, { key: 'social', path: '/social-responsibility' },
    { key: 'privacy', path: '/privacy' }, { key: 'terms', path: '/terms' }, { key: 'faq', path: '/faq' },
    { key: 'contact', path: '/contact' }, { key: 'contact-errors', path: '/contact', flow: 'submit-empty', form: 'main form' },
    { key: 'blog', path: '/blog' },
    { key: 'blog-category', discover: { from: '/blog', pattern: /href="(?:https?:\/\/[^/"]+)?(\/blog\/category\/[^"?#]+)"/ } },
    { key: 'post', discover: { from: '/blog', pattern: /href="(?:https?:\/\/[^/"]+)?(\/blog\/(?!category\/|tag\/|author\/|feed)[^"/?#]+)"/ } },
    { key: 'blog-tag', discover: { from: '@post', fallback: '/blog/tag/breastfeeding', pattern: /href="(?:https?:\/\/[^/"]+)?(\/blog\/tag\/[^"?#]+)"/ } },
    { key: 'blog-author', discover: { from: '@post', pattern: /href="(?:https?:\/\/[^/"]+)?(\/blog\/author\/[^"?#]+)"/ } },
    { key: 'search', path: '/search?q=%D8%AF%D8%B1%D8%AF' }, { key: 'search-empty', path: '/search' },
    { key: 'shop', path: '/shop' },
    { key: 'shop-category', discover: { from: '/shop', pattern: /href="(?:https?:\/\/[^/"]+)?(\/shop\/category\/[^"?#]+)"/ } },
    { key: 'product', discover: { from: '/shop', pattern: /href="(?:https?:\/\/[^/"]+)?(\/shop\/product\/[^"?#]+)"/ } },
    { key: 'buyable', discover: { from: '/shop/category/baby', buyable: true, pattern: /href="(?:https?:\/\/[^/"]+)?(\/shop\/product\/[^"?#]+)"/g } },
    { key: 'cart-empty', path: '/shop/cart' },
    { key: 'cart', path: '/shop/cart', flow: 'add-to-cart' },
    { key: 'checkout', path: '/shop/checkout', flow: 'add-to-cart' },
    { key: 'checkout-errors', path: '/shop/checkout', flow: 'add-to-cart submit-empty', form: 'form[action$="/shop/checkout"]' },
    { key: 'directory', path: '/directory' },
    { key: 'directory-city', discover: { from: '/directory', pattern: /href="(?:https?:\/\/[^/"]+)?(\/directory\/(?!place\/|business|join|booked)[^"/?#]+)"/ } },
    { key: 'directory-category', discover: { from: '/directory', pattern: /href="(?:https?:\/\/[^/"]+)?(\/directory\/(?!place\/)[^"/?#]+\/[^"/?#]+)"/ } },
    { key: 'place', discover: { from: '/directory', pattern: /href="(?:https?:\/\/[^/"]+)?(\/directory\/place\/[^"?#]+)"/ } },
    { key: 'business', path: '/directory/business' },
    { key: 'join', path: '/directory/join' },
    { key: 'join-errors', path: '/directory/join', flow: 'submit-empty', form: 'main form' },
    { key: 'join-done', path: '/directory/join/done' },
    { key: 'offline', path: '/offline' },
    { key: '404', path: '/this-page-does-not-exist', status: 404 },
];

/** The forms' time trap (FormTimer::MIN_SECONDS = 3) answers a faster submit like a bot: wait past it. */
const FormTimerWait = 3500;
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

function args(argv) {
    const o = { base: 'http://127.0.0.1:8000', widths: [390, 1440], only: null, json: null };
    for (let i = 0; i < argv.length; i++) {
        const a = argv[i];
        if (a === '--base') o.base = argv[++i].replace(/\/+$/, '');
        else if (a === '--widths') o.widths = argv[++i].split(',').map(Number);
        else if (a === '--only') o.only = new Set(argv[++i].split(','));
        else if (a === '--json') o.json = argv[++i];
        else throw new Error(`unknown argument ${a}`);
    }
    return o;
}

function axeSource() {
    const candidates = [
        process.env.AXE_PATH,
        resolve(ROOT, 'node_modules/axe-core/axe.min.js'),
        resolve(ROOT, '../frontend/node_modules/axe-core/axe.min.js'),
        resolve(ROOT, '../admin-web/node_modules/axe-core/axe.min.js'),
    ].filter(Boolean);
    const hit = candidates.find((p) => existsSync(p));
    return hit ? { path: hit, src: readFileSync(hit, 'utf8') } : null;
}

async function resolvePaths(base, pages) {
    const cache = new Map();
    const html = async (path) => {
        if (!cache.has(path)) cache.set(path, await (await fetch(base + path)).text().catch(() => ''));
        return cache.get(path);
    };
    const resolved = new Map();
    for (const p of pages) {
        if (p.path) { resolved.set(p.key, p.path); continue; }
        if (p.discover.buyable) {
            const links = [...(await html(p.discover.from)).matchAll(p.discover.pattern)].map((m) => m[1]);
            let hit = null;
            for (const link of [...new Set(links)].slice(0, 20)) {
                if (/<form id="buy" method="post"/.test(await html(link))) { hit = link; break; }
            }
            resolved.set(p.key, hit);
            continue;
        }
        const from = p.discover.from.startsWith('@') ? resolved.get(p.discover.from.slice(1)) : p.discover.from;
        const m = from ? p.discover.pattern.exec(await html(from)) : null;
        resolved.set(p.key, m ? m[1] : (p.discover.fallback && (await fetch(base + p.discover.fallback)).ok ? p.discover.fallback : null));
    }
    return resolved;
}

/* In-page custom checks (runs in the document). */
const CUSTOM = String.raw`(() => {
    const out = [];
    const add = (id, msg, target = '') => out.push({ id, msg, target });
    const html = document.documentElement;
    if (html.lang !== 'fa') add('lang', 'html lang is "' + html.lang + '"');
    if (html.dir !== 'rtl') add('dir', 'html dir is "' + html.dir + '"');
    const h1 = document.querySelectorAll('h1');
    if (h1.length !== 1) add('one-h1', h1.length + ' <h1> elements');
    let prev = 0;
    for (const h of document.querySelectorAll('h1,h2,h3,h4,h5,h6')) {
        if (h.closest('[aria-hidden="true"],[hidden],template')) continue;
        const lvl = +h.tagName[1];
        if (prev && lvl > prev + 1) add('heading-skip', 'h' + prev + ' → h' + lvl + ': ' + h.textContent.trim().slice(0, 40));
        prev = lvl;
    }
    const mains = document.querySelectorAll('main');
    if (mains.length !== 1) add('one-main', mains.length + ' <main>');
    const skip = document.querySelector('body a[href^="#"]');
    if (!skip || !document.getElementById(skip.getAttribute('href').slice(1))) add('skip-link', 'no skip link with an existing target');
    // Error state: every aria-invalid control names its error text through aria-describedby (3.3.1 / 3.3.3).
    for (const f of document.querySelectorAll('[aria-invalid="true"]')) {
        const ids = (f.getAttribute('aria-describedby') || '').split(/\s+/).filter(Boolean);
        const text = ids.map((id) => document.getElementById(id)?.textContent.trim() || '').join('');
        if (!text) add('error-described', 'aria-invalid control without a described error text', f.name || f.id);
    }
    out.invalid = document.querySelectorAll('[aria-invalid="true"]').length;
    return { list: out, invalid: out.invalid, cartLines: document.querySelectorAll('[data-cart-body] form[data-cart-form]').length };
})()`;

const STATE = String.raw`(async () => {
    const frame = () => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)));
    const el = document.activeElement;
    if (!el || el === document.body) return null;
    const r = el.getBoundingClientRect();
    const cs = getComputedStyle(el);
    const hiddenAncestor = el.closest('[hidden],[inert],[aria-hidden="true"]');
    const sel = el.tagName.toLowerCase() + (el.id ? '#' + el.id : '') + (el.getAttribute('name') ? '[name=' + el.getAttribute('name') + ']' : '')
        + (el.getAttribute('href') ? '[href=' + el.getAttribute('href').slice(0, 60) + ']' : '');
    const label = (el.getAttribute('aria-label') || el.textContent || el.value || '').trim().replace(/\s+/g, ' ').slice(0, 40);
    // Focus indicator = a visible style difference between the focused and the unfocused state of the element, its
    // label, parent or next sibling (wrapper / peer rings): outline, box-shadow, border colour or background.
    const boxes = [el, el.closest('label'), el.parentElement, el.nextElementSibling].filter(Boolean);
    const snap = () => boxes.map((b) => { const c = getComputedStyle(b);
        return [c.outlineStyle !== 'none' && parseFloat(c.outlineWidth) > 0 ? c.outlineStyle + c.outlineWidth + c.outlineColor : '', c.boxShadow, c.borderColor, c.backgroundColor, c.textDecorationLine].join('|'); });
    const on = snap();
    el.blur();
    await frame(); // let (reduced-motion, 0.01 ms) transitions settle
    const off = snap();
    el.focus();
    await frame();
    const changed = on.map((v, i) => v !== off[i]);
    const ring = changed.some(Boolean);
    const borderOnly = ring && boxes.every((b, i) => !changed[i] || on[i].split('|').slice(0, 2).join('|') === off[i].split('|').slice(0, 2).join('|'));
    // Inline links inside running text are exempt from target sizing (WCAG 2.5.5/2.5.8 "inline" exception).
    const inline = el.tagName === 'A' && cs.display === 'inline' && (el.parentElement?.textContent.trim().length ?? 0) > el.textContent.trim().length + 10;
    const visible = r.width > 0 && r.height > 0 && cs.visibility !== 'hidden' && !hiddenAncestor;
    return { sel, label, ring, borderOnly, visible, inline, x: r.x, y: r.y + scrollY, w: r.width, h: r.height };
})()`;

async function keyboardWalk(s) {
    const issues = [];
    const stops = [];
    const press = async (key, code, keyCode, shift = false) => {
        const mods = shift ? 8 : 0;
        await s('Input.dispatchKeyEvent', { type: 'keyDown', key, code, windowsVirtualKeyCode: keyCode, modifiers: mods });
        await s('Input.dispatchKeyEvent', { type: 'keyUp', key, code, windowsVirtualKeyCode: keyCode, modifiers: mods });
    };
    // Reset the sequential focus navigation starting point to the top of the document (error pages focus a field).
    await s('Runtime.evaluate', { expression: `(() => { const t = document.createElement('span'); t.tabIndex = -1; document.body.prepend(t); t.focus(); t.blur(); t.remove(); window.scrollTo(0, 0); })()` });
    const seen = new Set();
    for (let i = 0; i < 250; i++) {
        await press('Tab', 'Tab', 9);
        const { result } = await s('Runtime.evaluate', { expression: STATE, returnByValue: true, awaitPromise: true });
        const st = result.value;
        if (!st) break; // back at the document
        const id = `${st.sel}@${Math.round(st.x)},${Math.round(st.y)}`;
        if (seen.has(id)) break; // wrapped around
        seen.add(id);
        stops.push(st);
        if (!st.visible) issues.push({ id: 'focus-hidden', msg: `focus lands on an invisible element «${st.label}»`, target: st.sel });
        else if (!st.ring) issues.push({ id: 'focus-visible', msg: `no visible focus indicator on «${st.label}»`, target: st.sel });
        else if (st.borderOnly) issues.push({ id: 'focus-weak', msg: `focus shown only by a border/background colour change on «${st.label}» (add an outline/ring)`, target: st.sel });
    }
    const first = stops[0];
    if (!first || !/^a\[href=#/.test(first.sel.replace(/^a#[^[]+/, 'a'))) issues.push({ id: 'skip-first', msg: `first tab stop is ${first?.sel ?? 'nothing'}, not the skip link` });
    // Info (not a failure): targets under the task's 44 px comfort size; axe `target-size` enforces the 24 px minimum.
    const small = [...new Set(stops.filter((t) => t.visible && !t.inline && Math.min(t.w, t.h) < 44).map((t) => `${t.label || t.sel} ${Math.round(t.w)}×${Math.round(t.h)}`))];
    return { issues, stops: stops.length, small };
}

const MOTION = String.raw`(() => document.getAnimations().filter((a) => {
    const t = a.effect && a.effect.getTiming ? a.effect.getTiming() : {};
    const d = typeof t.duration === 'number' ? t.duration : 0;
    const motion = !a.transitionProperty || /^(transform|translate|scale|rotate|top|right|bottom|left|inset|margin|width|height|offset)/.test(a.transitionProperty);
    return motion && a.playState === 'running' && (d > 10 || t.iterations === Infinity);
}).map((a) => (a.animationName || a.transitionProperty || 'animation') + ' on ' + (a.effect && a.effect.target ? a.effect.target.tagName.toLowerCase() + '.' + String(a.effect.target.className).split(' ').slice(0, 3).join('.') : '?')))()`;


/* ---------- Pixel contrast for text axe cannot judge (gradient / image backgrounds) ---------- */

/** Minimal PNG decoder (8-bit RGB/RGBA, non-interlaced — what Chrome's captureScreenshot emits). */
function decodePng(buf) {
    let pos = 8, width = 0, height = 0, type = 0;
    const idat = [];
    while (pos < buf.length) {
        const len = buf.readUInt32BE(pos), kind = buf.toString('ascii', pos + 4, pos + 8), data = buf.subarray(pos + 8, pos + 8 + len);
        if (kind === 'IHDR') { width = data.readUInt32BE(0); height = data.readUInt32BE(4); type = data[9]; }
        else if (kind === 'IDAT') idat.push(data);
        pos += 12 + len;
    }
    const bpp = type === 6 ? 4 : 3, stride = width * bpp, raw = inflateSync(Buffer.concat(idat));
    const out = Buffer.alloc(height * stride);
    for (let y = 0; y < height; y++) {
        const f = raw[y * (stride + 1)], line = raw.subarray(y * (stride + 1) + 1, (y + 1) * (stride + 1));
        for (let x = 0; x < stride; x++) {
            const a = x >= bpp ? out[y * stride + x - bpp] : 0, b = y ? out[(y - 1) * stride + x] : 0, c = x >= bpp && y ? out[(y - 1) * stride + x - bpp] : 0;
            let v = line[x];
            if (f === 1) v += a; else if (f === 2) v += b; else if (f === 3) v += (a + b) >> 1;
            else if (f === 4) { const p = a + b - c, pa = Math.abs(p - a), pb = Math.abs(p - b), pc = Math.abs(p - c); v += pa <= pb && pa <= pc ? a : pb <= pc ? b : c; }
            out[y * stride + x] = v & 255;
        }
    }
    return { width, height, bpp, data: out };
}

const lum = (r, g, b) => [r, g, b].map((v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; })
    .reduce((acc, v, i) => acc + v * [0.2126, 0.7152, 0.0722][i], 0);

async function pixelContrast(s, selectors) {
    const out = [];
    for (const sel of [...new Set(selectors)].slice(0, 40)) {
        const { result } = await s('Runtime.evaluate', { returnByValue: true, expression: `(() => {
            const el = document.querySelector(${JSON.stringify(sel)});
            if (!el || el.closest('[aria-hidden="true"]')) return null; // decorative mockups: incidental text (1.4.3 exception)
            const clip = [el, ...el.querySelectorAll('*')].some((n) => getComputedStyle(n).backgroundClip === 'text');
            if (clip) return { skip: 'background-clip:text' };
            el.scrollIntoView({ block: 'center' });
            const cs = getComputedStyle(el);
            const m = cs.color.match(/[\\d.]+/g).map(Number);
            const size = parseFloat(cs.fontSize), bold = +cs.fontWeight >= 700;
            // Measure under the glyphs only (one clip per line box of a text Range), not the padding/border box.
            const range = document.createRange(); range.selectNodeContents(el);
            const rects = [...range.getClientRects()].filter((q) => q.width > 0 && q.height > 0).slice(0, 6)
                .map((q) => ({ x: q.left + scrollX, y: q.top + scrollY, width: q.width, height: q.height }));
            if (!rects.length) return null;
            // Inline text shares its line boxes with sibling text: hide the glyphs of the whole block around it.
            let block = el;
            while (block.parentElement && getComputedStyle(block).display.startsWith('inline')) block = block.parentElement;
            // Hide the glyphs through the CSSOM (allowed by the page's CSP, unlike an injected <style>); restored below.
            window.__a11yRestore = [block, ...block.querySelectorAll('*')].map((n) => {
                const prev = n.getAttribute('style');
                n.style.setProperty('transition', 'none', 'important');
                n.style.setProperty('color', 'transparent', 'important');
                n.style.setProperty('-webkit-text-fill-color', 'transparent', 'important');
                n.style.setProperty('text-shadow', 'none', 'important');
                n.style.setProperty('text-decoration-color', 'transparent', 'important');
                if (n instanceof SVGElement) n.style.setProperty('visibility', 'hidden', 'important');
                return [n, prev];
            });
            return { rects, rgb: m.slice(0, 3), large: size >= 24 || (bold && size >= 18.66) };
        })()` });
        const box = result.value;
        if (box?.skip) { out.push({ sel, ratio: 21, need: 0, skip: box.skip }); continue; }
        if (!box) continue;
        await sleep(30);
        const pngs = [];
        for (const rect of box.rects) {
            const shot = await s('Page.captureScreenshot', { format: 'png', captureBeyondViewport: true, clip: { ...rect, width: Math.min(rect.width, 1600), height: Math.min(rect.height, 400), scale: 1 } });
            pngs.push(decodePng(Buffer.from(shot.data, 'base64')));
        }
        await s('Runtime.evaluate', { expression: `(window.__a11yRestore || []).forEach(([n, prev]) => prev === null ? n.removeAttribute('style') : n.setAttribute('style', prev))` });
        await sleep(80); // restored colours transition back in; let them settle before the next read
        const fg = lum(...box.rgb), ratios = [];
        for (const png of pngs) for (let i = 0; i < png.data.length; i += png.bpp * 3) {
            const bg = lum(png.data[i], png.data[i + 1], png.data[i + 2]);
            const hex = '#' + [0, 1, 2].map((k) => png.data[i + k].toString(16).padStart(2, '0')).join('');
            ratios.push([(Math.max(fg, bg) + 0.05) / (Math.min(fg, bg) + 0.05), hex]);
        }
        ratios.sort((a, b) => a[0] - b[0]);
        const at = ratios[Math.floor(ratios.length * 0.1)] ?? [21, ''];
        const fgHex = '#' + box.rgb.map((v) => v.toString(16).padStart(2, '0')).join('');
        out.push({ sel, ratio: at[0], need: box.large ? 3 : 4.5, fg: fgHex, bg: at[1] });
    }
    return out;
}

async function runFlow(s, state, base, page, productPath) {
    for (const step of (page.flow ?? '').split(' ').filter(Boolean)) {
        if (step === 'add-to-cart') {
            await state.navigate(base + productPath);
            await s('Runtime.evaluate', { expression: `(() => { const f = [...document.querySelectorAll('form')].find((f) => /cart/.test(f.action) && f.method.toLowerCase() === 'post'); if (f) { f.submit(); return 1; } return 0; })()` });
            await sleep(400);
            await state.idle();
        }
    }
    await state.navigate(base + state.path);
    if ((page.flow ?? '').includes('submit-empty')) {
        // Real browser validation would block an empty submit; turn it off so the server-side error state renders.
        await s('Runtime.evaluate', { expression: `(() => { const f = document.querySelector(${JSON.stringify(page.form)}); if (!f) return 0; f.noValidate = true; f.querySelectorAll('input:not([type=hidden]):not([type=radio]):not([type=checkbox]),textarea').forEach((i) => { i.value = ''; }); setTimeout(() => HTMLFormElement.prototype.submit.call(f), ${FormTimerWait}); return 1; })()` });
        await sleep(FormTimerWait + 300);
        await sleep(500);
        await state.idle();
    }
}

/* ---------- Keyboard walkthroughs of the interactive widgets (390 px, touch emulation off) ---------- */

const KEYS = {
    Tab: ['Tab', 9], Enter: ['Enter', 13], Escape: ['Escape', 27], ' ': ['Space', 32],
    ArrowLeft: ['ArrowLeft', 37], ArrowRight: ['ArrowRight', 39], ArrowDown: ['ArrowDown', 40],
};

/** Each scenario: page (key of PAGES), steps run in the page; returns a list of failed expectations. */
const SCENARIOS = [
    { name: 'skip link → main', page: 'home', run: async (k) => {
        await k.press('Tab');
        await k.expect(`document.activeElement.getAttribute('href') === '#main'`, 'first Tab focuses the skip link');
        await k.press('Enter');
        await k.expect(`document.activeElement.id === 'main'`, 'Enter moves focus to <main>');
    } },
    { name: 'mobile menu', page: 'home', run: async (k) => {
        await k.focus('[data-module~="menu"]');
        await k.press('Enter');
        await k.expect(`document.querySelector('[data-module~="menu"]').getAttribute('aria-expanded') === 'true' && !document.getElementById('site-menu').hidden`, 'Enter opens the menu (aria-expanded=true)');
        await k.press('Tab');
        await k.expect(`document.getElementById('site-menu').contains(document.activeElement)`, 'next Tab enters the menu');
        await k.press('Escape');
        await k.expect(`document.getElementById('site-menu').hidden && document.activeElement.matches('[data-module~="menu"]')`, 'Esc closes and returns focus to the burger');
    } },
    { name: 'accordion (details)', page: 'faq', run: async (k) => {
        await k.focus('details:not([open]) > summary');
        await k.press('Enter');
        await k.expect(`document.activeElement.parentElement.open === true`, 'Enter opens a closed question');
        await k.press(' ');
        await k.expect(`document.activeElement.parentElement.open === false`, 'Space closes it again');
    } },
    { name: 'faq search filter', page: 'faq', run: async (k) => {
        await k.focus('#faq-search');
        await k.type('zzzz');
        await k.expect(`!document.querySelector('[data-faq-empty]') || !document.querySelector('[data-faq-empty]').hidden`, 'typing a miss shows the empty note');
    } },
    { name: 'calculator', page: 'tools', run: async (k) => {
        await k.focus('#due-date-cycle');
        await k.eval(`document.querySelector('#due-date-cycle').value = '99'`);
        await k.press('Enter');
        await k.expect(`document.activeElement.id === 'due-date-cycle' && document.activeElement.getAttribute('aria-invalid') === 'true' && document.getElementById('due-date-error').textContent.trim() !== ''`, 'Enter with an out-of-range cycle: error text, aria-invalid, focus stays on the field');
        await k.eval(`document.querySelector('#due-date-cycle').value = '28'`);
        await k.press('Enter');
        await k.expect(`!document.querySelector('#due-date [data-calc-result]').hidden && document.querySelector('#due-date [data-calc-result]').closest('[aria-live]') !== null`, 'Enter with valid input shows the result inside a live region');
    } },
    { name: 'shop filters', page: 'shop-category', run: async (k) => {
        await k.focus('form[role="search"] input[type="checkbox"], section[aria-labelledby="filters-title"] input[type="checkbox"]');
        const before = await k.eval(`document.activeElement.checked`);
        await k.press(' ');
        await k.expect(`document.activeElement.checked === ${!before}`, 'Space toggles a filter checkbox');
    } },
    { name: 'product variant radios', page: 'buyable', run: async (k) => {
        await k.focus('#buy input[type="radio"]:checked:not(:disabled)');
        const before = await k.eval(`document.activeElement.value`);
        await k.press('ArrowLeft');
        await k.expect(`document.activeElement.type === 'radio' && (document.activeElement.value !== ${JSON.stringify(before)} || document.querySelectorAll('#buy input[name="' + document.activeElement.name + '"]:not(:disabled)').length === 1)`, 'arrow keys move between variant options');
    } },
    { name: 'cart quantity stepper', page: 'cart', run: async (k) => {
        await k.focus('[data-cart-body] form[data-cart-form] button[name="quantity"]:not([disabled])');
        const label = await k.eval(`document.activeElement.getAttribute('aria-label')`);
        const qty = await k.eval(`document.querySelector('[data-cart-body] form[data-cart-form] b').textContent.trim()`);
        await k.press('Enter');
        await k.wait(1200);
        await k.expect(`document.querySelector('[data-cart-body] form[data-cart-form] b').textContent.trim() !== ${JSON.stringify(qty)}`, 'Enter changes the quantity');
        await k.expect(`document.activeElement.getAttribute('aria-label') === ${JSON.stringify(label)} || document.activeElement.matches('[data-cart-status], h1')`, 'focus returns to the same button (or the status line)');
    } },
    { name: 'join stepper', page: 'join', run: async (k) => {
        await k.focus('[data-stepper-next]');
        await k.press('Enter');
        await k.expect(`document.activeElement.matches('input, select, textarea') && !document.activeElement.checkValidity()`, 'Enter on «ذخیره و ادامه» with empty fields focuses the first invalid field');
    } },
    { name: 'place gallery lightbox', page: 'place', run: async (k) => {
        if (!(await k.eval(`!!document.querySelector('[data-gallery-dialog]')`))) return k.skip('no photos on the sample place');
        await k.focus('[data-gallery-open]');
        await k.press('Enter');
        await k.expect(`document.querySelector('[data-gallery-dialog]').open`, 'Enter opens the lightbox');
        const counter = await k.eval(`document.querySelector('[data-gallery-counter]').textContent`);
        await k.press('ArrowLeft');
        await k.expect(`document.querySelector('[data-gallery-counter]').textContent !== ${JSON.stringify(counter)}`, '← shows the next photo (RTL)');
        await k.press('Tab');
        await k.expect(`document.querySelector('[data-gallery-dialog]').contains(document.activeElement)`, 'Tab stays inside the modal dialog');
        await k.press('Escape');
        await k.expect(`!document.querySelector('[data-gallery-dialog]').open && document.activeElement.matches('[data-gallery-open]')`, 'Esc closes and returns focus to the opener');
    } },
];

async function runScenarios(cdp, base, paths, only) {
    const results = [];
    for (const sc of SCENARIOS) {
        if (only && !only.has(sc.page)) continue;
        const path = paths.get(sc.page);
        if (!path) { results.push({ name: sc.name, fails: ['no sample page'] }); continue; }
        const page = PAGES.find((p) => p.key === sc.page);
        const res = await withTab(cdp, async (s, state) => {
            await s('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
            state.path = path;
            await runFlow(s, state, base, { ...page, flow: (page.flow ?? '').replace('submit-empty', '') }, paths.get('buyable') ?? paths.get('product'));
            await s('Runtime.evaluate', { expression: `new Promise((r) => setTimeout(r, 400))`, awaitPromise: true }); // lazy modules
            const fails = [];
            let skipped = null;
            const evaluate = async (expr) => (await s('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true })).result.value;
            const k = {
                eval: evaluate,
                wait: (ms) => sleep(ms),
                skip: (why) => { skipped = why; },
                focus: async (sel) => { if (!(await evaluate(`(() => { const e = document.querySelector(${JSON.stringify(sel)}); if (!e) return false; e.focus(); return document.activeElement === e; })()`))) fails.push(`cannot focus ${sel}`); },
                press: async (key) => {
                    const [code, kc] = KEYS[key];
                    await s('Input.dispatchKeyEvent', { type: 'keyDown', key, code, windowsVirtualKeyCode: kc, ...(key.length === 1 ? { text: key } : key === 'Enter' ? { text: '\r' } : {}) });
                    await s('Input.dispatchKeyEvent', { type: 'keyUp', key, code, windowsVirtualKeyCode: kc });
                    await sleep(250);
                },
                type: async (text) => { await s('Input.insertText', { text }); await sleep(300); },
                expect: async (expr, label) => { if (!(await evaluate(`(() => { try { return !!(${expr}); } catch { return false; } })()`))) fails.push(label); },
            };
            await sc.run(k);
            return { fails, skipped };
        });
        results.push({ name: sc.name, path, ...res });
        console.log(`${res.fails.length ? '✘' : res.skipped ? '–' : '✔'} keyboard: ${sc.name}${res.skipped ? ` (skipped: ${res.skipped})` : ''}`);
        for (const f of res.fails) console.log(`    - ${f}`);
    }
    return results;
}

async function auditPage(cdp, base, page, path, width, axe, productPath) {
    return withTab(cdp, async (s, state) => {
        const mobile = width < 768;
        await s('Emulation.setDeviceMetricsOverride', { width, height: mobile ? 844 : 900, deviceScaleFactor: 1, mobile });
        if (mobile) await s('Emulation.setTouchEmulationEnabled', { enabled: true });
        await s('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: 'light' }, { name: 'prefers-reduced-motion', value: 'reduce' }] });
        state.path = path;
        await runFlow(s, state, base, page, productPath);
        const status = state.status;
        const issues = [];
        let review = 0;
        if (status !== (page.status ?? 200) && !page.flow?.includes('submit-empty')) issues.push({ id: 'status', msg: `HTTP ${status}` });
        for (const url of state.external) issues.push({ id: 'external-request', msg: url });

        const motion = await s('Runtime.evaluate', { expression: MOTION, returnByValue: true });
        for (const m of motion.result.value) issues.push({ id: 'reduced-motion', msg: `animation runs with reduced motion: ${m}` });
        if (axe) {
            await s('Runtime.evaluate', { expression: axe.src });
            const run = `axe.run(document, { runOnly: { type: 'tag', values: ['wcag2a','wcag2aa','wcag21a','wcag21aa','best-practice'] },
                rules: { 'target-size': { enabled: true } }, resultTypes: ['violations', 'incomplete'] })
                .then((r) => ({
                    violations: r.violations.map((v) => ({ id: v.id, impact: v.impact, help: v.help, nodes: v.nodes.length,
                        targets: v.nodes.slice(0, 4).map((n) => n.target.join(' ') + (n.any[0] ? ' — ' + n.any[0].message : '')) })),
                    review: r.incomplete.filter((v) => v.id === 'color-contrast').flatMap((v) => v.nodes.map((n) => n.target[0])),
                }))`;
            const { result, exceptionDetails } = await s('Runtime.evaluate', { expression: run, awaitPromise: true, returnByValue: true });
            if (exceptionDetails) issues.push({ id: 'axe-error', msg: exceptionDetails.text });
            else {
                const measured = await pixelContrast(s, result.value.review);
                review = measured.length;
                for (const m of measured.filter((x) => x.ratio < x.need)) {
                    issues.push({ id: 'pixel-contrast', msg: `${m.ratio.toFixed(2)}:1 < ${m.need}:1 over a gradient/image (text ${m.fg} on ${m.bg}, worst 10% of background pixels)`, target: m.sel });
                }
            }
            if (!exceptionDetails) for (const v of result.value.violations) issues.push({ id: `axe:${v.id}`, msg: `${v.help} (${v.impact}, ${v.nodes} nodes)`, target: v.targets.join(' | ') });
        }
        const custom = await s('Runtime.evaluate', { expression: CUSTOM, returnByValue: true });
        issues.push(...custom.result.value.list);
        const probe = { invalid: custom.result.value.invalid, cartLines: custom.result.value.cartLines };
        const walk = await keyboardWalk(s);
        issues.push(...walk.issues);
        return { key: page.key, path, width, status, stops: walk.stops, contrastReview: review, probe, small: walk.small, issues };
    });
}

async function main() {
    const o = args(process.argv.slice(2));
    const axe = axeSource();
    if (!axe) console.warn('! axe-core not found (set AXE_PATH) — custom checks only');
    else console.log(`axe-core: ${axe.path}`);
    try { await fetch(o.base + '/'); } catch { console.error(`✘ site does not answer at ${o.base}`); return 2; }
    const pages = PAGES.filter((p) => !o.only || o.only.has(p.key));
    const paths = await resolvePaths(o.base, PAGES);
    const chrome = await launchChrome();
    const cdp = await Cdp.connect(chrome.wsUrl);
    const rows = [];
    try {
        for (const page of pages) {
            const path = paths.get(page.key);
            if (!path) { rows.push({ key: page.key, path: null, width: null, issues: [{ id: 'not-found', msg: 'no sample URL' }] }); continue; }
            for (const width of o.widths) {
                const row = await auditPage(cdp, o.base, page, path, width, axe, paths.get('buyable') ?? paths.get('product'));
                rows.push(row);
                const mark = row.issues.length ? '✘' : '✔';
                console.log(`${mark} ${page.key.padEnd(18)} ${String(width).padStart(4)}  ${path}  (${row.stops} tab stops, ${row.contrastReview} gradient-contrast nodes measured by pixels${row.probe?.invalid ? `, ${row.probe.invalid} invalid fields` : ''}${row.probe?.cartLines ? `, ${row.probe.cartLines} cart forms` : ''})`);
                for (const i of row.issues) console.log(`    - ${i.id}: ${i.msg}${i.target ? `\n        ${i.target}` : ''}`);
            }
        }
        var keyboard = await runScenarios(cdp, o.base, paths, o.only);
    } finally {
        cdp.close();
        await chrome.close();
    }
    if (o.json) writeFileSync(resolve(o.json), `${JSON.stringify({ base: o.base, axe: axe ? 'axe-core' : null, rows, keyboard }, null, 2)}\n`);
    const bad = rows.filter((r) => r.issues.length);
    const badKeys = keyboard.filter((r) => r.fails.length);
    console.log(bad.length ? `✘ ${bad.length}/${rows.length} page×width runs with issues` : `✔ ${rows.length} page×width runs clean`);
    console.log(badKeys.length ? `✘ ${badKeys.length}/${keyboard.length} keyboard walkthroughs failed` : `✔ ${keyboard.length} keyboard walkthroughs pass`);
    return bad.length || badKeys.length ? 1 : 0;
}

main().then((c) => process.exit(c), (e) => { console.error(`✘ a11y-check: ${e.stack ?? e.message}`); process.exit(2); });

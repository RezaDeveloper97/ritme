#!/usr/bin/env node
/**
 * style2tw — first-pass converter from the design export's inline styles to Tailwind v4 utilities.
 *
 *   node tools/style2tw.mjs design/html/cycle.html [--section N] [--no-icons] > /tmp/cycle.blade.php
 *
 * - Tokens come from resources/css/app.css (@theme): colours, text sizes, leading, radii, shadows.
 * - The design page is RTL, so physical left/right become logical end/start (ms/me, ps/pe, start/end, border-s/e).
 * - Responsive behaviour: the simple `.rt-page [style*="…"]` rules of design/html/assets/ritme.css are parsed and
 *   applied with the same substring + source-order semantics, emitted as max-xl/max-lg/max-sm variants. The few
 *   structural selectors (header nav, flex wrap, footer row, footer grid brand cell) are encoded by hand below.
 * - Links to `*.html` become `{{ route('…') }}` from the URL map in docs/AUDIT.md §7.
 * - Inline icons become `<x-icon>` when resources/svg/icons exists (L0-07); SVG colour attributes become classes.
 * - A report (unmapped declarations, off-palette colours, arbitrary-value counts) goes to stderr.
 *
 * No dependencies beyond Node built-ins. Exports its building blocks for tests (tests/tools).
 */
import { readFileSync, existsSync, readdirSync } from 'node:fs';
import { dirname, join, resolve, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');

/* ------------------------------------------------------------------------------------------------ *
 * Small helpers
 * ------------------------------------------------------------------------------------------------ */

/** Round to 2 decimals and drop trailing zeros: 52.800000000000004 → "52.8". */
export function fmt(n) {
    return String(Math.round(n * 100) / 100);
}

/** Encode a raw CSS value for use inside Tailwind's `[...]`. */
export function arb(value) {
    return value.trim().replace(/_/g, '\\_').replace(/\s*,\s*/g, ',').replace(/\s+/g, '_');
}

/** Split on a separator at depth 0 (outside parentheses and quotes). */
export function splitTop(str, sep) {
    const out = [];
    let depth = 0;
    let quote = null;
    let cur = '';
    for (const ch of str) {
        if (quote) {
            if (ch === quote) quote = null;
        } else if (ch === '"' || ch === "'") {
            quote = ch;
        } else if (ch === '(') {
            depth++;
        } else if (ch === ')') {
            depth--;
        } else if (depth === 0 && (sep === ' ' ? /\s/.test(ch) : ch === sep)) {
            if (sep !== ' ' || cur !== '') out.push(cur);
            cur = '';
            continue;
        }
        cur += ch;
    }
    if (cur !== '' || sep !== ' ') out.push(cur);
    return out.map((s) => s.trim()).filter(Boolean);
}

/** Parse `a:b;c:d` into ordered [{prop, value}] (lower-cased props, `!important` dropped). */
export function parseStyle(style) {
    return splitTop(style, ';')
        .map((d) => {
            const i = d.indexOf(':');
            if (i < 0) return null;
            return {
                prop: d.slice(0, i).trim().toLowerCase(),
                value: d
                    .slice(i + 1)
                    .replace(/!important/i, '')
                    .trim(),
            };
        })
        .filter((d) => d && d.prop && d.value !== '');
}

/** Parse a length: {n, unit} | null. Unitless 0 → px. */
export function parseLength(v) {
    const m = /^(-?\d*\.?\d+)(px|%|rem|em|vh|vw)?$/.exec(String(v).trim());
    if (!m) return null;
    const n = parseFloat(m[1]);
    return { n, unit: m[2] ?? (n === 0 ? 'px' : '') };
}

/* ------------------------------------------------------------------------------------------------ *
 * Colours
 * ------------------------------------------------------------------------------------------------ */

/** Parse #rgb/#rrggbb/#rrggbbaa/rgb()/rgba() → {r,g,b,a} | null. */
export function parseColor(str) {
    const s = String(str).trim().toLowerCase();
    let m = /^#([0-9a-f]{3,8})$/.exec(s);
    if (m) {
        let h = m[1];
        if (h.length === 3 || h.length === 4) h = [...h].map((c) => c + c).join('');
        if (h.length !== 6 && h.length !== 8) return null;
        return {
            r: parseInt(h.slice(0, 2), 16),
            g: parseInt(h.slice(2, 4), 16),
            b: parseInt(h.slice(4, 6), 16),
            a: h.length === 8 ? parseInt(h.slice(6, 8), 16) / 255 : 1,
        };
    }
    m = /^rgba?\(([^)]+)\)$/.exec(s);
    if (m) {
        const parts = m[1].split(/[\s,/]+/).filter(Boolean);
        if (parts.length < 3) return null;
        const [r, g, b] = parts.slice(0, 3).map(Number);
        let a = parts[3] === undefined ? 1 : parts[3].endsWith('%') ? parseFloat(parts[3]) / 100 : Number(parts[3]);
        if ([r, g, b, a].some((x) => Number.isNaN(x))) return null;
        return { r, g, b, a };
    }
    return null;
}

const hex2 = (n) => n.toString(16).padStart(2, '0');
export const colorKey = (c) => `#${hex2(c.r)}${hex2(c.g)}${hex2(c.b)}`;
const COLOR_RE = /#[0-9a-fA-F]{3,8}\b|rgba?\([^)]*\)/g;

/* ------------------------------------------------------------------------------------------------ *
 * Theme (parsed from resources/css/app.css @theme)
 * ------------------------------------------------------------------------------------------------ */

export function loadTheme(cssPath = join(ROOT, 'resources/css/app.css')) {
    const css = readFileSync(cssPath, 'utf8');
    const block = /@theme\s*\{([\s\S]*?)\n\}/.exec(css)?.[1] ?? '';
    const theme = { colors: new Map(), text: new Map(), leading: new Map(), radius: new Map(), shadow: new Map() };
    const rem = (v) => {
        const l = parseLength(v);
        return l ? (l.unit === 'rem' ? l.n * 16 : l.n) : null;
    };
    for (const [, name, raw] of block.matchAll(/--([a-z0-9-]+)\s*:\s*([^;]+);/g)) {
        const value = raw.trim();
        if (value === 'initial') continue;
        if (name.startsWith('color-')) {
            const c = parseColor(value);
            const key = c && colorKey(c);
            if (key && !theme.colors.has(key)) theme.colors.set(key, name.slice(6));
        } else if (name.startsWith('text-') && !name.includes('--')) {
            theme.text.set(fmt(rem(value)), name.slice(5));
        } else if (name.startsWith('leading-')) {
            theme.leading.set(fmt(Number(value)), name.slice(8));
        } else if (name.startsWith('radius-')) {
            theme.radius.set(fmt(rem(value)), name.slice(7));
        } else if (name.startsWith('shadow-')) {
            theme.shadow.set(normalizeShadow(value), name.slice(7));
        }
    }
    return theme;
}

export function normalizeShadow(v) {
    return v
        .replace(COLOR_RE, (c) => {
            const p = parseColor(c);
            return p ? `rgba(${p.r},${p.g},${p.b},${fmt(p.a)})` : c;
        })
        .replace(/\s+/g, ' ')
        .trim();
}

/* Background white is a "surface" (cards, bands); everywhere else it is plain white. */
const COLOR_PREFER = { bg: { '#ffffff': 'surface' } };

/* ------------------------------------------------------------------------------------------------ *
 * Declaration mapper
 * ------------------------------------------------------------------------------------------------ */

export class Report {
    constructor() {
        this.unmapped = new Map();
        this.offPalette = new Map();
        this.arbitrary = 0;
        this.utilities = 0;
        this.declarations = 0;
        this.elements = 0;
        this.variants = { 'max-xl': 0, 'max-lg': 0, 'max-sm': 0 };
        this.links = 0;
        this.unresolvedLinks = new Map();
        this.dynamicLinks = new Map();
        this.icons = 0;
        this.iconSet = 'absent';
        this.svgColors = 0;
        this.legacyClasses = new Set();
    }

    bump(map, key) {
        map.set(key, (map.get(key) ?? 0) + 1);
    }
}

const FONT_WEIGHT = { 100: 'thin', 200: 'extralight', 300: 'light', 400: 'normal', 500: 'medium', 600: 'semibold', 700: 'bold', 800: 'extrabold', 900: 'black', normal: 'normal', bold: 'bold' };
const DISPLAY = { flex: 'flex', 'inline-flex': 'inline-flex', grid: 'grid', 'inline-grid': 'inline-grid', block: 'block', 'inline-block': 'inline-block', inline: 'inline', none: 'hidden', contents: 'contents' };
const ALIGN = { center: 'center', 'flex-start': 'start', start: 'start', 'flex-end': 'end', end: 'end', baseline: 'baseline', stretch: 'stretch' };
const JUSTIFY = { center: 'center', 'flex-start': 'start', start: 'start', 'flex-end': 'end', end: 'end', 'space-between': 'between', 'space-around': 'around', 'space-evenly': 'evenly' };
const SIMPLE = {
    position: { relative: 'relative', absolute: 'absolute', fixed: 'fixed', sticky: 'sticky', static: 'static' },
    overflow: { hidden: 'overflow-hidden', auto: 'overflow-auto', scroll: 'overflow-scroll', visible: 'overflow-visible' },
    'overflow-x': { auto: 'overflow-x-auto', hidden: 'overflow-x-hidden' },
    'overflow-y': { auto: 'overflow-y-auto', hidden: 'overflow-y-hidden' },
    'flex-direction': { column: 'flex-col', row: 'flex-row', 'column-reverse': 'flex-col-reverse', 'row-reverse': 'flex-row-reverse' },
    'flex-wrap': { wrap: 'flex-wrap', nowrap: 'flex-nowrap' },
    'flex-grow': { 1: 'grow', 0: 'grow-0' },
    'flex-shrink': { 0: 'shrink-0', 1: 'shrink' },
    cursor: { pointer: 'cursor-pointer', default: 'cursor-default' },
    'list-style': { none: 'list-none' },
    'box-sizing': { 'border-box': 'box-border', 'content-box': 'box-content' },
    'white-space': { nowrap: 'whitespace-nowrap', normal: 'whitespace-normal', 'pre-line': 'whitespace-pre-line' },
    'align-self': { 'flex-start': 'self-start', 'flex-end': 'self-end', center: 'self-center', stretch: 'self-stretch' },
    outline: { none: 'outline-none' },
    'text-decoration': { 'line-through': 'line-through', underline: 'underline', none: 'no-underline' },
    'border-collapse': { collapse: 'border-collapse' },
    'backdrop-filter': { 'blur(8px)': 'backdrop-blur-sm', 'blur(4px)': 'backdrop-blur-xs', 'blur(12px)': 'backdrop-blur-md' },
    'transform-origin': { 'right top': 'origin-top-right', 'top right': 'origin-top-right', 'left top': 'origin-top-left', 'top left': 'origin-top-left', 'top center': 'origin-top', 'center top': 'origin-top', center: 'origin-center' },
    // Physical → logical (the page is dir="rtl": right = start, left = end).
    'text-align': { center: 'text-center', right: 'text-start', left: 'text-end', start: 'text-start', end: 'text-end', justify: 'text-justify' },
    'vertical-align': { middle: 'align-middle', top: 'align-top' },
    'pointer-events': { none: 'pointer-events-none' },
};
const FRACTIONS = { 50: '1/2', 25: '1/4', 75: '3/4', 20: '1/5', 40: '2/5', 60: '3/5', 80: '4/5' };

export class Mapper {
    constructor(theme, report = new Report()) {
        this.theme = theme;
        this.report = report;
    }

    /* --- primitives ---------------------------------------------------------------------------- */

    arbitrary(cls) {
        this.report.arbitrary++;
        return cls;
    }

    /** Spacing scale key for a px value (2px grid, plus `px`), else null. */
    static spacing(px) {
        if (px === 0) return '0';
        if (px === 1) return 'px';
        if (Number.isInteger(px) && px % 2 === 0) return fmt(px / 4);
        return null;
    }

    /** `p`, `-mt`, `w` … + CSS length → utility. */
    length(prefix, value, { auto = true } = {}) {
        const v = String(value).trim();
        if (v === 'auto' && auto) return `${prefix}-auto`;
        const l = parseLength(v);
        if (!l) return this.arbitrary(`${prefix}-[${arb(this.tokenize(v))}]`);
        if (l.unit === 'px') {
            const key = Mapper.spacing(Math.abs(l.n));
            if (key !== null) return l.n < 0 ? `-${prefix}-${key}` : `${prefix}-${key}`;
            return this.arbitrary(`${prefix}-[${fmt(l.n)}px]`);
        }
        if (l.unit === '%') {
            if (l.n === 100) return `${prefix}-full`;
            const f = FRACTIONS[Math.abs(l.n)];
            if (f) return l.n < 0 ? `-${prefix}-${f}` : `${prefix}-${f}`;
        }
        return this.arbitrary(`${prefix}-[${fmt(l.n)}${l.unit}]`);
    }

    /** Colour utility (`bg`, `text`, `border-b`, `fill`…) for a CSS colour value. */
    color(prefix, value) {
        const v = String(value).trim().toLowerCase();
        if (v === 'transparent') return `${prefix}-transparent`;
        if (v === 'currentcolor') return `${prefix}-current`;
        if (v === 'inherit') return `${prefix}-inherit`;
        const c = parseColor(v);
        if (!c) return null;
        const key = colorKey(c);
        const name = COLOR_PREFER[prefix]?.[key] ?? this.theme.colors.get(key);
        const alpha = c.a < 1 ? `/${Math.round(c.a * 100)}` : '';
        if (name) return `${prefix}-${name}${alpha}`;
        this.report.bump(this.report.offPalette, key.toUpperCase());
        return this.arbitrary(`${prefix}-[${c.a < 1 ? key + hex2(Math.round(c.a * 255)) : key}]`);
    }

    /** Replace palette colours inside a raw value with theme variables (for arbitrary values). */
    tokenize(value) {
        return value.replace(COLOR_RE, (raw) => {
            const c = parseColor(raw);
            if (!c) return raw;
            const key = colorKey(c);
            const name = this.theme.colors.get(key);
            if (!name) {
                this.report.bump(this.report.offPalette, key.toUpperCase());
                return raw;
            }
            if (c.a >= 1) return `var(--color-${name})`;
            return `color-mix(in srgb, var(--color-${name}) ${Math.round(c.a * 100)}%, transparent)`;
        });
    }

    /** Escape hatch: arbitrary property, recorded as unmapped. */
    unmapped(prop, value) {
        this.report.bump(this.report.unmapped, `${prop}:${value}`);
        return this.arbitrary(`[${prop}:${arb(this.tokenize(value))}]`);
    }

    /* --- box model ----------------------------------------------------------------------------- */

    static expandBox(value) {
        const p = splitTop(value, ' ');
        const [t, r = t, b = t, l = r] = p;
        return [t, r, b, l];
    }

    /** padding/margin shorthand → p/px/py/pt/ps… (right = start, left = end). */
    box(base, value) {
        const [t, r, b, l] = Mapper.expandBox(value);
        if (t === r && r === b && b === l) return [this.length(base, t)];
        const out = [];
        if (t === b) out.push(this.length(`${base}y`, t));
        else out.push(this.length(`${base}t`, t), this.length(`${base}b`, b));
        if (r === l) out.push(this.length(`${base}x`, r));
        else out.push(this.length(`${base}s`, r), this.length(`${base}e`, l));
        return out;
    }

    radiusKey(value) {
        const l = parseLength(value);
        if (!l) return null;
        if (l.unit === 'px') {
            if (l.n === 0) return 'none';
            if (l.n >= 999) return 'full';
            return this.theme.radius.get(fmt(l.n)) ?? null;
        }
        if (l.unit === '%' && l.n === 50) return 'full';
        return null;
    }

    radius(value, el) {
        const parts = Mapper.expandBox(value);
        const [tl, tr, br, bl] = parts;
        const one = (side, v) => {
            const k = this.radiusKey(v);
            const p = side ? `rounded-${side}` : 'rounded';
            return k ? `${p}-${k}` : this.arbitrary(`${p}-[${fmt(parseLength(v)?.n ?? 0)}px]`);
        };
        if (tl === tr && tr === br && br === bl) {
            const l = parseLength(tl);
            // Pills/circles: radius == height / 2 (and not narrower than tall).
            if (l && l.unit === 'px' && el?.height && !this.radiusKey(tl)?.match(/^(none|full)$/)) {
                const h = el.height;
                if (l.n * 2 === h && (el.width === undefined || el.width >= h)) return ['rounded-full'];
            }
            return [one('', tl)];
        }
        // RTL corners: top-right = start-start, top-left = start-end, bottom-right = end-start, bottom-left = end-end.
        if (tl === tr && bl === br) return [one('t', tl), one('b', bl)];
        if (tr === br && tl === bl) return [one('s', tr), one('e', tl)];
        return [one('ss', tr), one('se', tl), one('es', br), one('ee', bl)];
    }

    /** border / border-<side> shorthand. `side` is a Tailwind side suffix ('', 't', 'b', 's', 'e'). */
    border(side, value) {
        const pre = side ? `border-${side}` : 'border';
        const v = value.trim().toLowerCase();
        if (v === 'none' || v === '0') return [`${pre}-0`];
        let width = null;
        let style = null;
        let color = null;
        for (const part of splitTop(value, ' ')) {
            if (/^(solid|dashed|dotted|double|none)$/i.test(part)) style = part.toLowerCase();
            else if (parseLength(part)) width = part;
            else color = part;
        }
        const out = [];
        const w = width ? parseLength(width) : { n: 1, unit: 'px' };
        if (w.unit === 'px' && w.n === 1) out.push(pre);
        else if (w.unit === 'px' && Number.isInteger(w.n)) out.push(`${pre}-${w.n}`);
        else out.push(this.arbitrary(`${pre}-[${fmt(w.n)}${w.unit}]`));
        if (style && style !== 'solid') out.push(style === 'none' ? `${pre}-0` : `border-${style}`);
        if (color) out.push(this.color(pre, color) ?? this.unmapped(`${side ? 'border-' + side : 'border'}-color`, color));
        return out;
    }

    /* --- backgrounds --------------------------------------------------------------------------- */

    background(value, isImage = false) {
        const v = value.trim();
        if (/^none$/i.test(v)) return ['bg-none'];
        const color = this.color('bg', v);
        if (color && !isImage) return [color];
        if (/^radial-gradient\(\s*circle at 78% 18%/.test(v) && v.includes('185,166,255,.35')) return ['bg-hero-glow'];
        const native = this.nativeLinear(v);
        if (native) return native;
        if (/^(repeating-)?(linear|radial|conic)-gradient\(.*\)$/.test(v) && splitTop(v, ' ').length === 1) {
            return [this.arbitrary(`bg-[${arb(this.tokenize(v))}]`)];
        }
        return [this.unmapped(isImage ? 'background-image' : 'background', v)];
    }

    /** linear-gradient(<angle>|to <side>, c1, [c2,] c3) with plain palette stops → bg-linear-* from/via/to. */
    nativeLinear(v) {
        const m = /^linear-gradient\((.*)\)$/.exec(v);
        if (!m) return null;
        const parts = splitTop(m[1], ',');
        let dir;
        const first = parts[0];
        if (/^-?\d+deg$/.test(first)) dir = String(parseInt(first, 10));
        else if (/^to (left|right|top|bottom)$/.test(first)) dir = `to-${{ left: 'l', right: 'r', top: 't', bottom: 'b' }[first.slice(3)]}`;
        else return null;
        const stops = parts.slice(1);
        if (stops.length < 2 || stops.length > 3) return null;
        const colors = stops.map((s) => (splitTop(s, ' ').length === 1 ? s : null));
        if (colors.includes(null)) return null;
        const inPalette = (c) => c.toLowerCase() === 'transparent' || (parseColor(c) && this.theme.colors.has(colorKey(parseColor(c))));
        if (!colors.every(inPalette)) return null;
        const names = ['from', ...(colors.length === 3 ? ['via'] : []), 'to'];
        const cls = colors.map((c, i) => this.color(names[i], c));
        // `/srgb` keeps the browser's default sRGB interpolation (Tailwind defaults to oklab).
        return [`bg-linear-${dir}/srgb`, ...cls];
    }

    /* --- main entry ---------------------------------------------------------------------------- */

    /**
     * Map ordered declarations to utilities.
     * @param {{prop:string,value:string}[]} decls
     * @param {{baseTransforms?:string[], skipFontSize?:boolean}} ctx
     */
    map(decls, ctx = {}) {
        const out = [];
        const last = new Map(decls.map((d) => [d.prop, d.value]));
        const el = { width: pxOf(last.get('width')), height: pxOf(last.get('height')) };
        const done = new Set();

        // width + height with the same scale value → size-N
        if (last.has('width') && last.has('height') && last.get('width') === last.get('height')) {
            const s = this.length('size', last.get('width'));
            out.push(s);
            done.add('width').add('height');
        }

        for (const { prop, value } of decls) {
            if (done.has(prop) && (prop === 'width' || prop === 'height')) continue;
            if (last.get(prop) !== value) continue; // a later declaration of the same property wins
            const res = this.one(prop, value, el, ctx);
            for (const c of [res].flat()) if (c) out.push(c);
        }
        this.report.utilities += out.length;
        return [...new Set(out)];
    }

    one(prop, value, el, ctx) {
        const v = value.trim();
        const lv = v.toLowerCase();
        if (SIMPLE[prop]?.[lv] !== undefined) return SIMPLE[prop][lv];
        switch (prop) {
            case 'display':
                return DISPLAY[lv] ?? this.unmapped(prop, v);
            case 'align-items':
                return ALIGN[lv] ? `items-${ALIGN[lv]}` : this.unmapped(prop, v);
            case 'justify-content':
                return JUSTIFY[lv] ? `justify-${JUSTIFY[lv]}` : this.unmapped(prop, v);
            case 'color':
                return this.color('text', v) ?? this.unmapped(prop, v);
            case 'background':
            case 'background-color':
                return this.background(v);
            case 'background-image':
                return this.background(v, true);
            case 'accent-color':
                return this.color('accent', v) ?? this.unmapped(prop, v);
            case 'font-size': {
                if (ctx.skipFontSize) return null;
                const l = parseLength(v);
                if (l?.unit === 'px') {
                    const t = this.theme.text.get(fmt(l.n));
                    return t ? `text-${t}` : this.arbitrary(`text-[${fmt(l.n)}px]`);
                }
                return this.unmapped(prop, v);
            }
            case 'font-weight':
                return FONT_WEIGHT[lv] ? `font-${FONT_WEIGHT[lv]}` : this.unmapped(prop, v);
            case 'font-family':
                if (/^lalezar/i.test(v)) return 'font-display';
                if (/^vazirmatn/i.test(v)) return 'font-sans';
                return this.unmapped(prop, v);
            case 'line-height': {
                const t = this.theme.leading.get(fmt(Number(v)));
                if (t) return `leading-${t}`;
                return Number.isNaN(Number(v)) ? this.unmapped(prop, v) : this.arbitrary(`leading-[${v.replace(/^0\./, '.')}]`);
            }
            case 'opacity':
                return Number.isNaN(Number(v)) ? this.unmapped(prop, v) : `opacity-${Math.round(Number(v) * 100)}`;
            case 'padding':
            case 'margin':
                return this.box(prop[0], v);
            case 'padding-top':
            case 'padding-bottom':
            case 'margin-top':
            case 'margin-bottom':
                return this.length(`${prop[0]}${prop.endsWith('top') ? 't' : 'b'}`, v);
            case 'padding-right':
            case 'margin-right':
                return this.length(`${prop[0]}s`, v);
            case 'padding-left':
            case 'margin-left':
                return this.length(`${prop[0]}e`, v);
            case 'width':
                return this.length('w', v);
            case 'height':
                return this.length('h', v);
            case 'min-width':
                return this.length('min-w', v);
            case 'min-height':
                return this.length('min-h', v);
            case 'max-width':
                return v === 'none' ? 'max-w-none' : this.length('max-w', v);
            case 'max-height':
                return this.length('max-h', v);
            case 'flex-basis':
                return this.length('basis', v);
            case 'top':
            case 'bottom':
                return this.length(prop, v);
            case 'right':
                return this.length('start', v);
            case 'left':
                return this.length('end', v);
            case 'inset':
                return this.length('inset', v);
            case 'gap': {
                const [row, col] = splitTop(v, ' ');
                if (col === undefined || col === row) return this.length('gap', row);
                return [this.length('gap-y', row), this.length('gap-x', col)];
            }
            case 'row-gap':
                return this.length('gap-y', v);
            case 'column-gap':
                return this.length('gap-x', v);
            case 'flex': {
                if (/^1 1 0(%|px)?$/.test(v) || v === '1') return 'flex-1';
                if (v === 'none') return 'flex-none';
                if (v === 'auto') return 'flex-auto';
                return this.arbitrary(`flex-[${arb(v)}]`);
            }
            case 'z-index':
                return /^\d+$/.test(v) ? `z-${v}` : this.unmapped(prop, v);
            case 'grid-template-columns':
            case 'grid-template-rows': {
                const p = prop.endsWith('columns') ? 'grid-cols' : 'grid-rows';
                const rep = /^repeat\((\d+),\s*minmax\(0,\s*1fr\)\)$/.exec(v);
                if (rep) return `${p}-${rep[1]}`;
                if (/^minmax\(0,\s*1fr\)$/.test(v)) return `${p}-1`;
                if (/^(1fr)( 1fr)*$/.test(v)) return `${p}-${v.split(' ').length}`;
                return this.arbitrary(`${p}-[${arb(v)}]`);
            }
            case 'grid-column':
            case 'grid-row': {
                const p = prop === 'grid-column' ? 'col' : 'row';
                if (/^1 \/ -1$/.test(v)) return `${p}-span-full`;
                const m = /^span (\d+)$/.exec(v);
                return m ? `${p}-span-${m[1]}` : this.arbitrary(`${p}-[${arb(v)}]`);
            }
            case 'border-radius':
                return this.radius(v, el);
            case 'border':
                return this.border('', v);
            case 'border-top':
                return this.border('t', v);
            case 'border-bottom':
                return this.border('b', v);
            case 'border-right':
                return this.border('s', v);
            case 'border-left':
                return this.border('e', v);
            case 'border-color':
                return this.color('border', v) ?? this.unmapped(prop, v);
            case 'box-shadow': {
                if (lv === 'none') return 'shadow-none';
                const named = this.theme.shadow.get(normalizeShadow(v));
                if (named) return `shadow-${named}`;
                const ring = /^0 0 0 (\d+)px (.+)$/.exec(v);
                if (ring) {
                    const c = this.color('ring', ring[2]);
                    if (c) return [`ring-${ring[1]}`, c];
                }
                return this.arbitrary(`shadow-[${arb(this.tokenize(v))}]`);
            }
            case 'transform':
                return this.transform(v, ctx);
            case 'font':
                return lv === 'inherit' ? '[font:inherit]' : this.unmapped(prop, v);
            default:
                return this.unmapped(prop, v);
        }
    }

    transform(v, ctx) {
        if (v === 'none') {
            const kinds = ctx.baseTransforms?.length ? ctx.baseTransforms : ['transform'];
            return kinds.map((k) => `${k}-none`);
        }
        const out = [];
        for (const fn of splitTop(v, ' ')) {
            const m = /^(\w+)\((.*)\)$/.exec(fn);
            if (!m) return this.unmapped('transform', v);
            const [, name, args] = m;
            const a = args.split(',').map((s) => s.trim());
            if (name === 'scale' && a.length === 1 && !Number.isNaN(Number(a[0]))) out.push(`scale-${Math.round(Number(a[0]) * 100)}`);
            else if (name === 'rotate' && /^-?\d+deg$/.test(a[0])) {
                const d = parseInt(a[0], 10);
                out.push(d < 0 ? `-rotate-${-d}` : `rotate-${d}`);
            } else if (name === 'translate' || name === 'translateX' || name === 'translateY') {
                const axes = name === 'translate' ? ['x', 'y'] : [name.slice(-1).toLowerCase()];
                a.forEach((val, i) => out.push(this.length(`translate-${axes[i]}`, val, { auto: false })));
            } else return this.unmapped('transform', v);
        }
        return out;
    }
}

function pxOf(v) {
    const l = v && parseLength(v);
    return l && l.unit === 'px' ? l.n : undefined;
}

/** Which Tailwind transform families a base `transform` value uses (for resetting with `*-none`). */
export function transformKinds(value) {
    if (!value) return [];
    return [...new Set([...value.matchAll(/(scale|rotate|translate)/g)].map((m) => m[1]))];
}

/* ------------------------------------------------------------------------------------------------ *
 * Responsive rules from ritme.css
 * ------------------------------------------------------------------------------------------------ */

const MEDIA = { 1180: 'max-xl', 1024: 'max-lg', 700: 'max-sm' };

/** Parse the simple `.rt-page [style*="…"]{…}` rules (in source order) per media query. */
export function loadResponsiveRules(cssPath = join(ROOT, 'design/html/assets/ritme.css')) {
    const css = readFileSync(cssPath, 'utf8');
    const rules = [];
    const mediaRe = /@media\s*\(max-width:\s*(\d+)px\)\s*\{/g;
    let m;
    while ((m = mediaRe.exec(css))) {
        const variant = MEDIA[m[1]];
        let depth = 1;
        let i = mediaRe.lastIndex;
        const start = i;
        while (depth && i < css.length) {
            if (css[i] === '{') depth++;
            else if (css[i] === '}') depth--;
            i++;
        }
        const body = css.slice(start, i - 1);
        for (const [, selectors, decls] of body.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
            const subs = [];
            let simple = true;
            for (const sel of splitTop(selectors.trim(), ',')) {
                const sm = /^\.rt-page \[style\*="([^"]+)"\]$/.exec(sel.trim());
                if (!sm) simple = false;
                else subs.push(sm[1]);
            }
            if (simple && variant) rules.push({ variant, subs, decls: parseStyle(decls) });
        }
    }
    return rules;
}

/* ------------------------------------------------------------------------------------------------ *
 * Tolerant HTML tree (enough for the generated design export)
 * ------------------------------------------------------------------------------------------------ */

const VOID = new Set(['area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input', 'link', 'meta', 'source', 'track', 'wbr']);
const RAW = new Set(['script', 'style']);

export function parseHtml(html) {
    const root = { type: 'root', children: [] };
    const stack = [root];
    const re = /<!--[\s\S]*?-->|<![^>]*>|<\/([a-zA-Z][\w:-]*)\s*>|<([a-zA-Z][\w:-]*)((?:[^>"']|"[^"]*"|'[^']*')*)>/g;
    let pos = 0;
    let m;
    const top = () => stack[stack.length - 1];
    const add = (node) => {
        node.parent = top();
        top().children.push(node);
    };
    while ((m = re.exec(html))) {
        if (m.index > pos) add({ type: 'text', text: html.slice(pos, m.index) });
        pos = re.lastIndex;
        if (m[0].startsWith('<!--')) add({ type: 'comment', text: m[0] });
        else if (m[0].startsWith('<!')) add({ type: 'doctype', text: m[0] });
        else if (m[1]) {
            const tag = m[1].toLowerCase();
            const idx = stack.map((n) => n.tag).lastIndexOf(tag);
            if (idx > 0) stack.length = idx;
        } else {
            const tag = m[2].toLowerCase();
            let rawAttrs = m[3];
            const selfClosing = /\/\s*$/.test(rawAttrs);
            if (selfClosing) rawAttrs = rawAttrs.replace(/\/\s*$/, '');
            const node = { type: 'element', tag, name: m[2], attrs: parseAttrs(rawAttrs), children: [], selfClosing };
            add(node);
            if (RAW.has(tag)) {
                const end = html.toLowerCase().indexOf(`</${tag}`, pos);
                const stop = end < 0 ? html.length : end;
                if (stop > pos) node.children.push({ type: 'raw', text: html.slice(pos, stop), parent: node });
                const close = end < 0 ? html.length : html.indexOf('>', end) + 1;
                pos = close;
                re.lastIndex = close;
            } else if (!selfClosing && !VOID.has(tag)) {
                stack.push(node);
            }
        }
    }
    if (pos < html.length) add({ type: 'text', text: html.slice(pos) });
    return root;
}

function parseAttrs(raw) {
    const attrs = [];
    for (const [, name, dq, sq, uq] of raw.matchAll(/([^\s"'>/=]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+)))?/g)) {
        attrs.push([name, dq ?? sq ?? uq ?? null]);
    }
    return attrs;
}

export const getAttr = (node, name) => node.attrs?.find(([n]) => n.toLowerCase() === name)?.[1] ?? null;

export function setAttr(node, name, value) {
    const a = node.attrs.find(([n]) => n.toLowerCase() === name);
    if (value === undefined) node.attrs = node.attrs.filter(([n]) => n.toLowerCase() !== name);
    else if (a) a[1] = value;
    else node.attrs.push([name, value]);
}

const elementChildren = (node) => node.children.filter((c) => c.type === 'element');
const hasAncestor = (node, tag) => {
    for (let p = node.parent; p; p = p.parent) if (p.tag === tag) return true;
    return false;
};

/** Blade-safe text: `{{`, `{!!` and directive-looking `@word` must not be compiled. */
export function bladeText(s) {
    return s
        .replace(/\{\{/g, '@{{')
        .replace(/\{!!/g, '@{!!')
        .replace(/(^|[^\w@{])@(?=[A-Za-z])/g, '$1@@');
}

export function serialize(node) {
    switch (node.type) {
        case 'root':
            return node.children.map(serialize).join('');
        case 'text':
            return bladeText(node.text);
        case 'raw':
        case 'comment':
        case 'doctype':
            return node.text;
        case 'blade':
            return node.text;
        default: {
            const attrs = node.attrs
                .map(([n, v]) => (v === null ? ` ${n}` : ` ${n}="${n.startsWith(':') || n === 'href' || n === 'action' ? v : bladeText(v)}"`))
                .join('');
            if (node.selfClosing) return `<${node.name}${attrs}/>`;
            if (VOID.has(node.tag)) return `<${node.name}${attrs}>`;
            return `<${node.name}${attrs}>${node.children.map(serialize).join('')}</${node.name}>`;
        }
    }
}

/* ------------------------------------------------------------------------------------------------ *
 * URL map + icons
 * ------------------------------------------------------------------------------------------------ */

/* Routes in the URL map that take one slug parameter (the demo slug is the last path segment). */
const SLUG_ROUTES = new Set(['blog.show', 'directory.place', 'shop.category', 'shop.product']);

export function loadUrlMap(auditPath = join(ROOT, 'docs/AUDIT.md')) {
    if (!existsSync(auditPath)) return new Map();
    const json = /```json urlmap\s*([\s\S]*?)```/.exec(readFileSync(auditPath, 'utf8'))?.[1];
    const map = new Map();
    for (const e of json ? JSON.parse(json) : []) map.set(e.design, e);
    return map;
}

/** `cycle.html#x` → `{{ route('stage.cycle') }}#x`; null when not a design page link. */
export function routeFor(href, urlMap, report) {
    const m = /^([a-z0-9-]+\.html)(#.*)?$/i.exec(href);
    if (!m) return null;
    const e = urlMap.get(m[1]);
    if (!e) {
        report?.bump(report.unresolvedLinks, m[1]);
        return null;
    }
    let expr;
    if (e.route.includes('{')) {
        // Only meaningful with an id (booking/order confirmation) → the list page (AUDIT §7).
        const list = `${e.name.split('.')[0]}.index`;
        report?.bump(report.dynamicLinks, `${m[1]} → ${list}`);
        expr = `route('${list}')`;
    } else if (SLUG_ROUTES.has(e.name)) {
        expr = `route('${e.name}', '${e.route.split('/').pop()}')`;
        report?.bump(report.dynamicLinks, `${m[1]} → ${expr} (demo slug)`);
    } else {
        expr = `route('${e.name}')`;
    }
    report && report.links++;
    return `{{ ${expr} }}${m[2] ?? ''}`;
}

const GEOMETRY = ['d', 'cx', 'cy', 'r', 'rx', 'ry', 'x', 'y', 'x1', 'y1', 'x2', 'y2', 'width', 'height', 'points'];

/** Stable key of an SVG's drawing (tags + geometry attributes), used to recognise icons. */
export function svgKey(svgNode) {
    const parts = [];
    const walk = (n) => {
        for (const c of n.children ?? []) {
            if (c.type !== 'element') continue;
            if (c.tag !== 'title' && c.tag !== 'desc') {
                const geo = GEOMETRY.map((g) => [g, getAttr(c, g)])
                    .filter(([, v]) => v !== null)
                    .map(([g, v]) => `${g}=${v.replace(/\s+/g, ' ').trim()}`);
                parts.push(`${c.tag}(${geo.join(',')})`);
            }
            walk(c);
        }
    };
    walk(svgNode);
    return parts.join('|');
}

export function loadIcons(dir = join(ROOT, 'resources/svg/icons')) {
    const icons = new Map();
    if (!existsSync(dir)) return null;
    for (const f of readdirSync(dir)) {
        if (!f.endsWith('.svg')) continue;
        const tree = parseHtml(readFileSync(join(dir, f), 'utf8'));
        const svg = findFirst(tree, (n) => n.tag === 'svg');
        if (svg) icons.set(svgKey(svg), f.slice(0, -4));
    }
    return icons;
}

function findFirst(node, pred) {
    for (const c of node.children ?? []) {
        if (c.type !== 'element') continue;
        if (pred(c)) return c;
        const f = findFirst(c, pred);
        if (f) return f;
    }
    return null;
}

/* ------------------------------------------------------------------------------------------------ *
 * Element conversion
 * ------------------------------------------------------------------------------------------------ */

const ORDER = ['max-xl', 'max-lg', 'max-sm'];
const SVG_COLOR_ATTRS = { fill: 'fill', stroke: 'stroke' };

export class Converter {
    constructor({ theme = loadTheme(), rules = loadResponsiveRules(), urlMap = loadUrlMap(), icons = loadIcons(), report = new Report() } = {}) {
        this.report = report;
        this.mapper = new Mapper(theme, report);
        this.rules = rules;
        this.urlMap = urlMap;
        this.icons = icons;
        report.iconSet = icons ? `${icons.size} icons` : 'absent';
    }

    /** Classes for one element: [base…, max-xl:…, max-lg:…, max-sm:…]. */
    classesFor(node) {
        const style = getAttr(node, 'style') ?? '';
        const decls = parseStyle(style);
        const base = decls.length ? this.mapper.map(decls) : [];
        this.report.declarations += decls.length;

        // 1) ritme.css simple rules: substring match on the raw style, later rule wins per property.
        const merged = { 'max-xl': new Map(), 'max-lg': new Map(), 'max-sm': new Map() };
        if (style) {
            for (const r of this.rules) {
                if (r.subs.some((s) => style.includes(s))) for (const d of r.decls) merged[r.variant].set(d.prop, d.value);
            }
        }
        const baseFont = decls.findLast?.((d) => d.prop === 'font-size') ?? [...decls].reverse().find((d) => d.prop === 'font-size');
        const baseFontToken = baseFont && this.mapper.theme.text.get(fmt(parseLength(baseFont.value)?.n ?? -1));
        const ctx = {
            baseTransforms: transformKinds(decls.find((d) => d.prop === 'transform')?.value),
            // Display tokens (text-d-*) already step down at ≤1024/≤700 in app.css.
            skipFontSize: Boolean(baseFontToken?.startsWith('d-')),
        };
        const variants = {};
        for (const v of ORDER) {
            const list = [...merged[v]].map(([prop, value]) => ({ prop, value }));
            variants[v] = list.length ? this.mapper.map(list, ctx) : [];
        }

        // 2) Structural selectors from ritme.css that are not plain [style*=] rules.
        const isMainNav = node.tag === 'nav' && getAttr(node, 'aria-label') === 'منوی اصلی';
        if (isMainNav) variants['max-xl'].push('gap-3.5');
        if (node.tag === 'a' && node.parent?.tag === 'nav' && getAttr(node.parent, 'aria-label') === 'منوی اصلی') variants['max-xl'].push('text-base');
        if (isMainNav && hasAncestor(node, 'header')) variants['max-lg'].push('hidden');
        if (node.tag === 'div' && node.parent?.tag === 'header') {
            const divs = elementChildren(node.parent).filter((c) => c.tag === 'div');
            if (divs[divs.length - 1] === node) variants['max-lg'].push('hidden');
        }
        if (style.includes('display:flex') && !style.includes('flex-direction:column') && !style.includes('flex-wrap:nowrap') && node.tag !== 'nav' && node.tag !== 'header') {
            variants['max-lg'].push('flex-wrap');
        }
        if (node.tag === 'div' && (node.parent.origStyle ?? '').includes('grid-template-columns:1.3fr repeat(5,') && elementChildren(node.parent)[0] === node) {
            variants['max-lg'].push('col-span-full');
        }
        if (style.includes('justify-content:space-between') && hasAncestor(node, 'footer')) variants['max-sm'].push('flex-col', 'gap-3', 'items-start');

        // Drop variant classes identical to a base class (only safe when no wider variant overrides it).
        const out = [...base];
        let wider = false;
        for (const v of ORDER) {
            const list = [...new Set(variants[v])].filter((c) => wider || !base.includes(c));
            if (list.length) wider = true;
            this.report.variants[v] += list.length;
            out.push(...list.map((c) => `${v}:${c}`));
        }
        return out;
    }

    convertElement(node) {
        node.origStyle = getAttr(node, 'style') ?? '';
        if (node.origStyle) this.report.elements++;
        const cls = this.classesFor(node);
        const existing = (getAttr(node, 'class') ?? '').split(/\s+/).filter(Boolean);
        for (const c of existing) if (c.startsWith('rt-')) this.report.legacyClasses.add(c);

        // SVG presentation colours → fill-*/stroke-* classes (class beats attribute; no hex in markup).
        for (const [attr, prefix] of Object.entries(SVG_COLOR_ATTRS)) {
            const val = getAttr(node, attr);
            if (val && parseColor(val)) {
                const c = this.mapper.color(prefix, val);
                if (c) {
                    cls.push(c);
                    setAttr(node, attr, undefined);
                    this.report.svgColors++;
                }
            }
        }
        const stop = getAttr(node, 'stop-color');
        if (stop && parseColor(stop)) {
            cls.push(this.mapper.arbitrary(`[stop-color:${arb(this.mapper.tokenize(stop))}]`));
            setAttr(node, 'stop-color', undefined);
            this.report.svgColors++;
        }

        setAttr(node, 'style', undefined);
        const all = [...new Set([...existing, ...cls])];
        if (all.length) setAttr(node, 'class', all.join(' '));

        for (const attr of ['href', 'action']) {
            const v = getAttr(node, attr);
            const r = v && routeFor(v, this.urlMap, this.report);
            if (r) setAttr(node, attr, r);
        }
    }

    /** Replace a recognised inline icon with <x-icon>; returns true when replaced. */
    iconize(node) {
        if (!this.icons || node.tag !== 'svg') return false;
        const name = this.icons.get(svgKey(node));
        if (!name) return false;
        const cls = [];
        const w = getAttr(node, 'width');
        if (w && w === (getAttr(node, 'height') ?? w)) cls.push(this.mapper.length('size', `${w}px`));
        const stroke = getAttr(node, 'stroke');
        const fill = getAttr(node, 'fill');
        const color = stroke && stroke !== 'none' ? stroke : fill && fill !== 'none' ? fill : null;
        if (color && parseColor(color)) cls.push(this.mapper.color('text', color));
        if (getAttr(node, 'style')) cls.push(...this.classesFor(node));
        node.type = 'blade';
        node.text = `<x-icon name="${name}"${cls.length ? ` class="${cls.join(' ')}"` : ''}/>`;
        this.report.icons++;
        return true;
    }

    walk(node) {
        for (const c of node.children ?? []) {
            if (c.type !== 'element') continue;
            if (this.iconize(c)) continue;
            this.convertElement(c);
            this.walk(c);
        }
    }

    /** Convert a whole document; returns Blade for <body> content (or the N-th section of .rt-page). */
    convert(html, { section = null, source = '' } = {}) {
        const tree = parseHtml(html);
        const body = findFirst(tree, (n) => n.tag === 'body') ?? tree;
        // Section selection uses the original tree (context like footer/header must exist), so convert all first.
        this.walk(body);
        let target = body;
        let label = '';
        if (section !== null) {
            const page = findFirst(body, (n) => (getAttr(n, 'class') ?? '').split(/\s+/).includes('rt-page'));
            const kids = page ? elementChildren(page) : [];
            target = kids[section - 1];
            if (!target) throw new Error(`--section ${section}: .rt-page has ${kids.length} sections`);
            label = ` (section ${section}/${kids.length})`;
        }
        const inner = section !== null ? serialize(target) : target.children.map(serialize).join('');
        return `{{-- style2tw: ${source}${label} — generated first pass; extract components by hand. --}}\n${inner.trim()}\n`;
    }
}

/* ------------------------------------------------------------------------------------------------ *
 * Report + CLI
 * ------------------------------------------------------------------------------------------------ */

export function formatReport(r, source) {
    const top = (map, n = 40) =>
        [...map]
            .sort((a, b) => b[1] - a[1])
            .slice(0, n)
            .map(([k, c]) => `    ${String(c).padStart(4)}  ${k}`)
            .join('\n') || '    (none)';
    return [
        `style2tw report — ${source}`,
        `  elements with style: ${r.elements}, declarations: ${r.declarations}`,
        `  utilities: ${r.utilities} (incl. variants); variants max-xl ${r.variants['max-xl']} / max-lg ${r.variants['max-lg']} / max-sm ${r.variants['max-sm']}`,
        `  arbitrary values: ${r.arbitrary}`,
        `  links → route(): ${r.links}`,
        `  links needing review (dynamic/demo slug):\n${top(r.dynamicLinks)}`,
        `  unresolved .html links:\n${top(r.unresolvedLinks)}`,
        `  icons → <x-icon>: ${r.icons} (icon set: ${r.iconSet})`,
        `  svg colour attributes → classes: ${r.svgColors}`,
        `  unmapped declarations (kept as arbitrary properties):\n${top(r.unmapped)}`,
        `  colours outside the palette:\n${top(r.offPalette)}`,
        `  legacy rt-* classes kept (styled by the layout, not Tailwind): ${[...r.legacyClasses].join(', ') || '(none)'}`,
        '',
    ].join('\n');
}

const USAGE = `Usage: node tools/style2tw.mjs <design/html/page.html> [--section N] [--no-icons] [--quiet]

Converts inline styles to Tailwind v4 utilities (project theme tokens, RTL logical utilities, ritme.css responsive
rules as max-xl/max-lg/max-sm variants) and prints Blade for the <body> content to stdout.
  --section N   only the N-th top-level section (1-based child of .rt-page)
  --no-icons    keep inline SVG icons even when resources/svg/icons exists
  --quiet       no report on stderr
See tools/README.md.`;

export function main(argv) {
    const args = argv.slice(2);
    if (!args.length || args.includes('--help') || args.includes('-h')) {
        process.stdout.write(`${USAGE}\n`);
        return args.length ? 0 : 1;
    }
    const file = args.find((a) => !a.startsWith('--') && !/^\d+$/.test(a));
    const si = args.indexOf('--section');
    const section = si >= 0 ? Number(args[si + 1]) : null;
    if (!file || !existsSync(file) || (si >= 0 && !(section >= 1))) {
        process.stderr.write(`${USAGE}\n`);
        return 1;
    }
    const conv = new Converter({ icons: args.includes('--no-icons') ? null : loadIcons() });
    const source = relative(ROOT, resolve(file));
    process.stdout.write(conv.convert(readFileSync(file, 'utf8'), { section, source }));
    if (!args.includes('--quiet')) process.stderr.write(formatReport(conv.report, source));
    return 0;
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
    process.exitCode = main(process.argv);
}

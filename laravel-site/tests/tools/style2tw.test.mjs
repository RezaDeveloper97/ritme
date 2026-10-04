// node --test tests/tools — unit tests for tools/style2tw.mjs (declaration mapper, RTL, gradients, responsive rules).
import { test, describe } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, existsSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { Converter, Mapper, Report, loadTheme, loadResponsiveRules, loadUrlMap, parseStyle, parseColor, routeFor, splitTop, arb } from '../../tools/style2tw.mjs';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const theme = loadTheme();
const rules = loadResponsiveRules();

const map = (style, ctx) => new Mapper(theme).map(parseStyle(style), ctx);
const convert = (html) => new Converter({ theme, rules, urlMap: loadUrlMap(), icons: null }).convert(`<body>${html}</body>`).split('\n').slice(1).join('\n').trim();
const classesOf = (html) => /class="([^"]*)"/.exec(convert(html))?.[1].split(' ') ?? [];

describe('parsing helpers', () => {
    test('splitTop respects parentheses', () => {
        assert.deepEqual(splitTop('repeat(5,minmax(0,1fr)),a', ','), ['repeat(5,minmax(0,1fr))', 'a']);
        assert.deepEqual(splitTop('0 0 0 14px rgba(1, 2, 3, .4)', ' '), ['0', '0', '0', '14px', 'rgba(1, 2, 3, .4)']);
    });

    test('parseStyle keeps order, lower-cases props, drops !important', () => {
        assert.deepEqual(parseStyle('Color:#FFF;padding:1px 2px !important;;'), [
            { prop: 'color', value: '#FFF' },
            { prop: 'padding', value: '1px 2px' },
        ]);
    });

    test('parseColor handles hex alpha and rgba', () => {
        assert.deepEqual(parseColor('#6E54F022'), { r: 110, g: 84, b: 240, a: 34 / 255 });
        assert.deepEqual(parseColor('rgba(34,26,61,.82)'), { r: 34, g: 26, b: 61, a: 0.82 });
        assert.equal(parseColor('transparent'), null);
    });

    test('arb encodes spaces and keeps commas tight', () => {
        assert.equal(arb('circle, #fff 0 25%'), 'circle,#fff_0_25%');
    });
});

describe('theme', () => {
    test('reads tokens from app.css', () => {
        assert.equal(theme.colors.get('#6e54f0'), 'primary');
        assert.equal(theme.text.get('14'), 'base');
        assert.equal(theme.text.get('68'), 'd-4xl');
        assert.equal(theme.radius.get('40'), '7xl');
        assert.equal(theme.leading.get('1.9'), 'relaxed');
    });
});

describe('declaration mapper', () => {
    test('padding shorthand: 1–4 values, scale and arbitrary', () => {
        assert.deepEqual(map('padding:12px'), ['p-3']);
        assert.deepEqual(map('padding:0 18px'), ['py-0', 'px-4.5']);
        assert.deepEqual(map('padding:64px 120px 32px'), ['pt-16', 'pb-8', 'px-30']);
        assert.deepEqual(map('padding:1px 2px 3px 4px'), ['pt-px', 'pb-[3px]', 'ps-0.5', 'pe-1']);
        assert.deepEqual(map('padding:52.800000000000004px 20px'), ['py-[52.8px]', 'px-5']);
    });

    test('RTL: left/right become logical end/start', () => {
        assert.deepEqual(map('margin-left:8px'), ['me-2']);
        assert.deepEqual(map('margin-right:8px'), ['ms-2']);
        assert.deepEqual(map('margin:0 4px 0 12px'), ['my-0', 'ms-1', 'me-3']);
        assert.deepEqual(map('position:absolute;left:10px;right:-10px'), ['absolute', 'end-2.5', '-start-2.5']);
        assert.deepEqual(map('left:50%'), ['end-1/2']);
        assert.deepEqual(map('text-align:right'), ['text-start']);
        assert.deepEqual(map('text-align:left'), ['text-end']);
        assert.deepEqual(map('border-right:2px solid #E7E1F4'), ['border-s-2', 'border-s-line']);
        assert.deepEqual(map('border-left:1px solid #E7E1F4'), ['border-e', 'border-e-line']);
        assert.deepEqual(map('border-radius:20px 20px 0 0'), ['rounded-t-3xl', 'rounded-b-none']);
        assert.deepEqual(map('border-radius:0 14px 14px 0'), ['rounded-s-lg', 'rounded-e-none']);
    });

    test('colours map to tokens with opacity modifiers; off-palette colours are flagged', () => {
        const report = new Report();
        const m = new Mapper(theme, report);
        assert.deepEqual(m.map(parseStyle('color:#B8AED6;background:#6E54F022')), ['text-on-night-muted', 'bg-primary/13']);
        assert.deepEqual(m.map(parseStyle('background:#FFFFFF;color:#FFFFFF')), ['bg-surface', 'text-white']);
        assert.deepEqual(m.map(parseStyle('background:rgba(34,26,61,.82)')), ['bg-night-card/82']);
        assert.deepEqual(m.map(parseStyle('background:#F1EDFB')), ['bg-[#f1edfb]']);
        assert.equal(report.offPalette.get('#F1EDFB'), 1);
        assert.deepEqual(m.map(parseStyle('background:transparent;color:inherit')), ['bg-transparent', 'text-inherit']);
    });

    test('typography uses the theme scale, arbitrary otherwise', () => {
        assert.deepEqual(map('font-size:14px;font-weight:800;line-height:1.9'), ['text-base', 'font-extrabold', 'leading-relaxed']);
        assert.deepEqual(map('font-size:12.5px;line-height:1.7'), ['text-[12.5px]', 'leading-[1.7]']);
        assert.deepEqual(map('font-family:Lalezar,Vazirmatn,sans-serif;font-size:60px'), ['font-display', 'text-d-3xl']);
    });

    test('flex, grid, sizing', () => {
        assert.deepEqual(map('display:flex;flex-direction:column;align-items:flex-start;justify-content:space-between;gap:28px 20px'), [
            'flex',
            'flex-col',
            'items-start',
            'justify-between',
            'gap-y-7',
            'gap-x-5',
        ]);
        assert.deepEqual(map('display:grid;grid-template-columns:repeat(3,minmax(0,1fr))'), ['grid', 'grid-cols-3']);
        assert.deepEqual(map('grid-template-columns:1fr 1fr'), ['grid-cols-2']);
        assert.deepEqual(map('grid-template-columns:2fr 1fr'), ['grid-cols-[2fr_1fr]']);
        assert.deepEqual(map('width:40px;height:40px'), ['size-10']);
        assert.deepEqual(map('width:255px;height:527px;flex:1 1 0;flex-shrink:0'), ['w-[255px]', 'h-[527px]', 'flex-1', 'shrink-0']);
        assert.deepEqual(map('flex:2 1 0'), ['flex-[2_1_0]']);
        assert.deepEqual(map('max-width:760px;width:100%'), ['max-w-190', 'w-full']);
    });

    test('borders, radii and pills', () => {
        assert.deepEqual(map('border:1.5px solid #E7E1F4'), ['border-[1.5px]', 'border-line']);
        assert.deepEqual(map('border:1px dashed #6E54F0'), ['border', 'border-dashed', 'border-primary']);
        assert.deepEqual(map('border:none'), ['border-0']);
        assert.deepEqual(map('border-bottom:2px solid transparent'), ['border-b-2', 'border-b-transparent']);
        assert.deepEqual(map('border-radius:14px'), ['rounded-lg']);
        assert.deepEqual(map('height:46px;border-radius:23px'), ['h-11.5', 'rounded-full']);
        assert.deepEqual(map('width:20px;height:46px;border-radius:23px'), ['w-5', 'h-11.5', 'rounded-[23px]']);
        assert.deepEqual(map('border-radius:13px'), ['rounded-[13px]']);
    });

    test('gradients: native bg-linear for plain palette stops, tokenised arbitrary values otherwise', () => {
        assert.deepEqual(map('background:linear-gradient(135deg,#D9447F55,#D9447F12)'), ['bg-linear-135/srgb', 'from-stage-pregnancy/33', 'to-stage-pregnancy/7']);
        assert.deepEqual(map('background:linear-gradient(180deg,#FFFFFF,#F7F3FF)'), ['bg-linear-180/srgb', 'from-white', 'to-canvas']);
        assert.deepEqual(map('background:radial-gradient(circle,#ECE6FF,transparent 70%)'), ['bg-[radial-gradient(circle,var(--color-lavender),transparent_70%)]']);
        assert.deepEqual(map('background:linear-gradient(to left,#FF6B8B 0 17%,#B9A6FF 17% 45%)'), [
            'bg-[linear-gradient(to_left,var(--color-phase-period)_0_17%,var(--color-lilac)_17%_45%)]',
        ]);
        assert.deepEqual(
            map('background-image:radial-gradient(circle at 78% 18%,rgba(185,166,255,.35),transparent 42%),radial-gradient(circle at 12% 85%,rgba(255,107,139,.22),transparent 40%)'),
            ['bg-hero-glow'],
        );
    });

    test('shadows, transforms, effects', () => {
        assert.deepEqual(map('box-shadow:0 40px 80px -30px rgba(40,20,90,.45)'), ['shadow-phone']);
        assert.deepEqual(map('box-shadow:0 0 0 14px #0C70640D'), ['ring-14', 'ring-stage-teen/5']);
        assert.deepEqual(map('box-shadow:0 24px 48px -28px rgba(40,20,90,.45)'), ['shadow-[0_24px_48px_-28px_rgba(40,20,90,.45)]']);
        assert.deepEqual(map('transform:scale(.85);transform-origin:right top'), ['scale-85', 'origin-top-right']);
        assert.deepEqual(map('transform:translate(-50%,-50%)'), ['-translate-x-1/2', '-translate-y-1/2']);
        assert.deepEqual(map('transform:none', { baseTransforms: ['scale'] }), ['scale-none']);
        assert.deepEqual(map('opacity:.75;backdrop-filter:blur(8px);accent-color:#6E54F0'), ['opacity-75', 'backdrop-blur-sm', 'accent-primary']);
    });

    test('unknown declarations fall back to arbitrary properties and are reported', () => {
        const report = new Report();
        const out = new Mapper(theme, report).map(parseStyle('mix-blend-mode:multiply'));
        assert.deepEqual(out, ['[mix-blend-mode:multiply]']);
        assert.equal(report.unmapped.get('mix-blend-mode:multiply'), 1);
    });
});

describe('responsive rules from ritme.css', () => {
    test('parses the simple [style*=] rules per breakpoint', () => {
        assert.ok(rules.length > 40);
        assert.ok(rules.some((r) => r.variant === 'max-xl' && r.subs.includes('padding:20px 120px')));
        assert.ok(rules.some((r) => r.variant === 'max-sm' && r.subs.includes('position:absolute;left:3')));
    });

    test('section padding: 120px gutters → 20px, vertical ×0.55 at ≤1024', () => {
        assert.deepEqual(classesOf('<section style="padding:96px 120px;"></section>'), ['py-24', 'px-30', 'max-lg:py-[52.8px]', 'max-lg:px-5']);
        assert.deepEqual(classesOf('<section style="padding:64px 120px 32px;"></section>'), ['pt-16', 'pb-8', 'px-30', 'max-lg:pt-[35.2px]', 'max-lg:px-5']);
    });

    test('header padding steps 1180 → 1024', () => {
        assert.deepEqual(classesOf('<header style="padding:20px 120px;"></header>'), ['py-5', 'px-30', 'max-xl:py-3.5', 'max-xl:px-6', 'max-lg:p-5']);
    });

    test('grids collapse 4→2→1, 5→3→2, 8→4→2; footer brand cell spans', () => {
        assert.deepEqual(classesOf('<div style="display:grid;grid-template-columns:repeat(4,minmax(0,1fr));"></div>'), ['grid', 'grid-cols-4', 'max-lg:grid-cols-2', 'max-sm:grid-cols-1']);
        assert.deepEqual(classesOf('<div style="display:grid;grid-template-columns:repeat(5,minmax(0,1fr));"></div>'), ['grid', 'grid-cols-5', 'max-lg:grid-cols-3', 'max-sm:grid-cols-2']);
        assert.deepEqual(classesOf('<div style="display:grid;grid-template-columns:repeat(8,minmax(0,1fr));"></div>'), ['grid', 'grid-cols-8', 'max-lg:grid-cols-4', 'max-sm:grid-cols-2']);
        const footer = convert('<div style="display:grid;grid-template-columns:1.3fr repeat(5,minmax(0,1fr));"><div>a</div><div>b</div></div>');
        assert.match(footer, /max-lg:grid-cols-3 max-sm:grid-cols-1/);
        assert.match(footer, /<div class="max-lg:col-span-full">a<\/div><div>b<\/div>/);
    });

    test('flex rows wrap at ≤1024; split columns get basis 300 → 100%', () => {
        assert.deepEqual(classesOf('<div style="display:flex;gap:40px;"></div>'), ['flex', 'gap-10', 'max-lg:flex-wrap']);
        assert.deepEqual(classesOf('<div style="display:flex;flex-direction:column;"></div>'), ['flex', 'flex-col']);
        assert.deepEqual(classesOf('<nav style="display:flex;"></nav>'), ['flex']);
        assert.deepEqual(classesOf('<div style="flex:1 1 0;"></div>'), ['flex-1', 'max-lg:basis-75', 'max-sm:basis-full']);
    });

    test('display sizes: tokens step down by themselves, other sizes get variants', () => {
        assert.deepEqual(classesOf('<h1 style="font-size:68px;"></h1>'), ['text-d-4xl']);
        assert.deepEqual(classesOf('<h1 style="font-size:64px;"></h1>'), ['text-[64px]', 'max-lg:text-[46px]', 'max-sm:text-[34px]']);
        assert.deepEqual(classesOf('<h2 style="font-size:46px;"></h2>'), ['text-[46px]', 'max-sm:text-[30px]']);
    });

    test('≤700: fixed widths, radius 40, absolute decorations, scale, app-CTA margins', () => {
        assert.deepEqual(classesOf('<div style="width:600px;"></div>'), ['w-150', 'max-sm:w-full', 'max-sm:max-w-full']);
        assert.deepEqual(classesOf('<div style="border-radius:40px;"></div>'), ['rounded-7xl', 'max-sm:rounded-4xl']);
        assert.deepEqual(classesOf('<div style="position:absolute;left:340px;top:90px;"></div>'), ['absolute', 'end-85', 'top-22.5', 'max-sm:hidden']);
        assert.deepEqual(classesOf('<div style="transform:scale(.85);"></div>'), ['scale-85', 'max-sm:scale-none']);
        assert.deepEqual(classesOf('<div style="margin:0 120px 96px;"></div>'), ['mt-0', 'mb-24', 'mx-30', 'max-sm:mb-12', 'max-sm:mx-5']);
        assert.deepEqual(classesOf('<div style="padding:56px 64px;"></div>'), ['py-14', 'px-16', 'max-sm:py-7', 'max-sm:px-5.5']);
    });

    test('header nav and actions hide at ≤1024; footer bottom row stacks at ≤700', () => {
        const header = convert('<header><nav aria-label="منوی اصلی" style="display:flex;gap:26px;"><a href="#" style="font-size:15px;">x</a></nav><div style="display:flex;"></div></header>');
        assert.match(header, /<nav aria-label="منوی اصلی" class="flex gap-6.5 max-xl:gap-3.5 max-lg:hidden">/);
        assert.match(header, /<a href="#" class="text-md max-xl:text-base">/);
        assert.match(header, /<div class="flex max-lg:hidden max-lg:flex-wrap">/);
        const footer = convert('<footer><div style="display:flex;justify-content:space-between;"></div></footer>');
        assert.match(footer, /max-sm:flex-col max-sm:gap-3 max-sm:items-start/);
    });
});

describe('markup', () => {
    test('strips style, merges existing classes, keeps attributes and SVG children', () => {
        const out = convert('<button class="rt-burger" aria-expanded="false" style="display:flex;"><span></span></button><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#FFFFFF"><path d="M1 1"/></svg>');
        assert.equal(out, '<button class="rt-burger flex max-lg:flex-wrap" aria-expanded="false"><span></span></button><svg width="20" height="20" viewBox="0 0 24 24" fill="none" class="stroke-white"><path d="M1 1"/></svg>');
    });

    test('links to design pages become route() calls', () => {
        const urlMap = loadUrlMap();
        assert.equal(routeFor('index.html', urlMap), "{{ route('home') }}");
        assert.equal(routeFor('cycle.html#faq', urlMap), "{{ route('stage.cycle') }}#faq");
        assert.equal(routeFor('article.html', urlMap), "{{ route('blog.show', 'period-pain') }}");
        assert.equal(routeFor('shop-done.html', urlMap), "{{ route('shop.index') }}");
        assert.equal(routeFor('#download', urlMap), null);
        assert.equal(routeFor('https://example.com/a.html', urlMap), null);
    });

    test('text is Blade-safe', () => {
        assert.equal(convert('<p>a {{ b }} @if x@y.z</p>'), '<p>a @{{ b }} @@if x@y.z</p>');
    });

    test('index.html converts with zero style attributes', () => {
        const html = readFileSync(join(ROOT, 'design/html/index.html'), 'utf8');
        const out = new Converter({ theme, rules, urlMap: loadUrlMap(), icons: null }).convert(html, { source: 'index.html' });
        assert.equal((out.match(/\sstyle=/g) ?? []).length, 0);
        assert.doesNotMatch(out, /href="[a-z-]+\.html/);
        assert.equal((out.match(/<h1[\s>]/g) ?? []).length, 1);
    });
});

describe('Tailwind validity', () => {
    const tw = join(ROOT, 'node_modules/@tailwindcss/node/dist/index.mjs');
    test('every class emitted for every design page is a valid utility against app.css', { skip: !existsSync(tw) && '@tailwindcss/node not installed' }, async () => {
        const { __unstable__loadDesignSystem } = await import(tw);
        const ds = await __unstable__loadDesignSystem(readFileSync(join(ROOT, 'resources/css/app.css'), 'utf8'), { base: join(ROOT, 'resources/css') });
        const conv = new Converter({ theme, rules, urlMap: loadUrlMap(), icons: null });
        const classes = new Set();
        for (const page of loadUrlMap().keys()) {
            const out = conv.convert(readFileSync(join(ROOT, 'design/html', page), 'utf8'));
            for (const [, c] of out.matchAll(/class="([^"]*)"/g)) for (const k of c.split(/\s+/)) if (k && !k.startsWith('rt-')) classes.add(k);
        }
        const list = [...classes];
        const css = ds.candidatesToCss(list);
        const invalid = list.filter((_, i) => !css[i]);
        assert.deepEqual(invalid, []);
    });
});

#!/usr/bin/env python3
"""
font-subset — rebuild the Arabic-script woff2 files in resources/fonts (L9-02).

    python3 docs/qa/perf/font-subset.py <dir with the original @fontsource *-arabic-* and *-latin-* woff2 files>

Needs fontTools + brotli (`pip install fonttools brotli`). Writes resources/fonts/{vazirmatn-arabic-{400..800},
lalezar-arabic-400}-normal.woff2. The Latin files are not touched.

What changes per Arabic file (glyph outlines are copied, never redrawn):
  1. + the punctuation Persian copy uses, taken from the same weight's Latin file: ASCII punctuation (no digits),
     « » © · × – — ' ' " " • … ‹ › −, and for Vazirmatn also the Latin capitals A–Z (acronyms such as PMS / SPF in
     Persian copy). CSS unicode-range in resources/css/fonts.css claims them, so the 16 KB Latin file is no longer
     downloaded for a full stop, a colon or an acronym — only for lowercase Latin words and ASCII digits.
  2. − the cmap entries of the Arabic Presentation Forms blocks (U+FB50–FDFF, U+FE70–FEFC). Shaping goes through
     GSUB from U+0600–06FF, so every contextual form stays; only text typed *as* presentation-form code points (and
     the standalone ligature signs there) falls back to the next font in the stack. Lalezar: −14 KB.
  3. glyph names dropped (post table format 3): −2 KB per file.
"""
import os
import sys

from fontTools import subset
from fontTools.merge import Merger, Options as MergeOptions
from fontTools.ttLib import TTFont

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..', '..'))
OUT = os.path.join(ROOT, 'resources', 'fonts')

PUNCT = (list(range(0x20, 0x30)) + list(range(0x3A, 0x41)) + list(range(0x5B, 0x61)) + list(range(0x7B, 0x7F))
         + [0xA0, 0xA9, 0xAB, 0xB7, 0xBB, 0xD7, 0x2013, 0x2014, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2026,
            0x2039, 0x203A, 0x2212])
# Text faces only: Latin capitals, for the acronyms in Persian copy (PMS, PMDD, IRC, SPF…). Lalezar (display) does not
# get them; lowercase words and ASCII digits still load the Latin file.
CAPITALS = list(range(0x41, 0x5B))
ARABIC = (list(range(0x0600, 0x0700)) + list(range(0x0750, 0x0780)) + list(range(0x0870, 0x0900))
          + list(range(0x200C, 0x200F)) + [0x2010, 0x2011, 0x204F, 0x2E41])
FILES = [('vazirmatn', w) for w in (400, 500, 600, 700, 800)] + [('lalezar', 400)]


def options(glyph_names):
    o = subset.Options()
    o.layout_features = ['*']
    o.name_IDs = ['*']
    o.notdef_outline = True
    o.glyph_names = glyph_names
    return o


def main(src):
    tmp = os.path.join(OUT, '.subset-tmp')
    os.makedirs(tmp, exist_ok=True)
    for family, weight in FILES:
        arabic = TTFont(os.path.join(src, f'{family}-arabic-{weight}-normal.woff2'))
        latin = TTFont(os.path.join(src, f'{family}-latin-{weight}-normal.woff2'))
        arabic.flavor = latin.flavor = None
        # glyph names stay until the merge so equal names can be told apart, then go
        sub = subset.Subsetter(options(True))
        extra = PUNCT + (CAPITALS if family == 'vazirmatn' else [])
        sub.populate(unicodes=[u for u in extra if u in latin.getBestCmap()])
        sub.subset(latin)
        a_path, l_path = os.path.join(tmp, 'a.ttf'), os.path.join(tmp, 'l.ttf')
        arabic.save(a_path)
        latin.save(l_path)
        merged = Merger(options=MergeOptions(drop_tables=['vhea', 'vmtx'])).merge([a_path, l_path])
        keep = set(ARABIC) | set(extra)
        sub = subset.Subsetter(options(False))
        sub.populate(unicodes=[u for u in merged.getBestCmap() if u in keep])
        sub.subset(merged)
        merged.flavor = 'woff2'
        dst = os.path.join(OUT, f'{family}-arabic-{weight}-normal.woff2')
        merged.save(dst)
        missing = [hex(u) for u in PUNCT if u not in merged.getBestCmap()]
        print(f'{os.path.basename(dst)}: {os.path.getsize(dst)} bytes' + (f', missing {missing}' if missing else ''))
    for name in os.listdir(tmp):
        os.remove(os.path.join(tmp, name))
    os.rmdir(tmp)


if __name__ == '__main__':
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    main(sys.argv[1])

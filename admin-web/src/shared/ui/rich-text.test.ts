import { getSchema } from '@tiptap/core';
import { describe, expect, it } from 'vitest';
import type { DOMOutputSpec, Mark, Node } from '@tiptap/pm/model';

import { editorExtensions } from './editor-extensions';
import { normalizeEditorHtml, SANITIZER_ALLOWED_ATTRS, SANITIZER_ALLOWED_TAGS } from './rich-text';

/** Tag name + attribute names of every element a DOMOutputSpec produces. */
function walk(spec: DOMOutputSpec, out: Array<{ tag: string; attrs: string[] }>): void {
  if (typeof spec === 'string' || !Array.isArray(spec)) return;
  const [tag, ...rest] = spec as [string, ...unknown[]];
  const attrs: string[] = [];
  for (const part of rest) {
    if (part && typeof part === 'object' && !Array.isArray(part)) {
      for (const [k, v] of Object.entries(part)) if (v !== null && v !== undefined) attrs.push(k);
    } else if (Array.isArray(part)) {
      walk(part as unknown as DOMOutputSpec, out);
    }
  }
  out.push({ tag: tag.replace(/^.*\s/, ''), attrs });
}

describe('editor output vs backend sanitizer', () => {
  const schema = getSchema(editorExtensions);

  it('every node renders to allowed tags with allowed attributes', () => {
    const produced: Array<{ tag: string; attrs: string[] }> = [];
    for (const type of Object.values(schema.nodes)) {
      if (!type.spec.toDOM) continue;
      const attrs = type.name === 'heading' ? { level: 2 } : undefined;
      walk(type.spec.toDOM(type.create(attrs) as Node), produced);
    }
    for (const type of Object.values(schema.marks)) {
      if (!type.spec.toDOM) continue;
      const attrs = type.name === 'link' ? { href: 'https://ritme.app' } : undefined;
      walk(type.spec.toDOM(type.create(attrs) as Mark, true), produced);
    }
    expect(produced.length).toBeGreaterThan(10);
    for (const { tag, attrs } of produced) {
      expect(SANITIZER_ALLOWED_TAGS.has(tag), `<${tag}> is stripped by the sanitizer`).toBe(true);
      for (const a of attrs) {
        expect(SANITIZER_ALLOWED_ATTRS[tag] ?? [], `<${tag} ${a}> is stripped`).toContain(a);
      }
    }
  });

  it('normalizes the empty document to an empty string', () => {
    expect(normalizeEditorHtml('<p></p>')).toBe('');
    expect(normalizeEditorHtml(' <p>x</p> ')).toBe('<p>x</p>');
  });
});

import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const css = readFileSync(fileURLToPath(new URL('./globals.css', import.meta.url)), 'utf8');

function rule(selector: string): string {
  const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const match = css.match(new RegExp(`(?:^|\\n)\\s*${escaped}\\s*\\{([^}]*)\\}`));
  return match?.[1] ?? '';
}

describe('globals.css', () => {
  // QA 2026-09-29-c M1: the scroll box must contain the absolutely positioned .sr-only
  // table headers, or every list screen scrolls sideways at 390 px.
  it('.table-wrap is the containing block of its scrolled content', () => {
    const body = rule('.table-wrap');
    expect(body).toMatch(/overflow-x:\s*auto/);
    expect(body).toMatch(/position:\s*relative/);
  });
});

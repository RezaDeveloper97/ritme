import { describe, expect, it } from 'vitest';

import { infoPageSchema, splitAbout, splitLegal } from './info-page';

const raw = (sections: Array<Record<string, unknown>>) =>
  infoPageSchema.parse({
    updated_at: '2026-09-23',
    sections: sections.map((s, i) => ({ id: i + 1, key: null, heading: `h${i}`, body: `b${i}`, link_label: null, link_url: null, ...s })),
  });

describe('info page', () => {
  it('drops unsafe links and half links', () => {
    const page = raw([
      { link_label: 'x', link_url: 'javascript:alert(1)' },
      { link_label: null, link_url: 'https://a.b' },
      { link_label: 'IG', link_url: 'https://instagram.com/ritme' },
    ]);
    expect(page.sections.map((s) => s.linkUrl)).toEqual([null, null, 'https://instagram.com/ritme']);
  });

  it('splitLegal takes the summary lines out of the numbered sections', () => {
    const { summary, sections } = splitLegal(raw([{ key: 'summary', body: 'one\n two \n\nthree' }, { key: 'a' }]));
    expect(summary).toEqual(['one', 'two', 'three']);
    expect(sections.map((s) => s.key)).toEqual(['a']);
  });

  it('splitAbout: intro, rows, link chips, disclaimer', () => {
    const parts = splitAbout(
      raw([
        { key: 'what' },
        { key: 'team' },
        { key: 'ig', link_label: 'IG', link_url: 'https://instagram.com/x' },
        { key: 'disclaimer' },
      ]),
    );
    expect(parts.intro?.key).toBe('what');
    expect(parts.rows.map((s) => s.key)).toEqual(['team']);
    expect(parts.links.map((s) => s.key)).toEqual(['ig']);
    expect(parts.disclaimer?.key).toBe('disclaimer');
  });
});

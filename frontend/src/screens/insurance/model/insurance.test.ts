import { describe, expect, it } from 'vitest';

import { insurancePageSchema, splitInsurance } from './insurance';

const box = (id: number, key: string | null, body = 'x', link: [string, string] | null = null) => ({
  id, key, heading: `h${id}`, body, link_label: link?.[0] ?? null, link_url: link?.[1] ?? null,
});

describe('insurance page model', () => {
  it('splits keyed boxes, lines and FAQ', () => {
    const page = insurancePageSchema.parse({
      updated_at: '2026-10-11',
      sections: [box(1, 'coverage'), box(2, 'claim_steps', 'a\n\n b '), box(3, 'documents', 'c'), box(4, null), box(5, 'disclaimer')],
    });
    const s = splitInsurance(page);
    expect(s.coverage?.id).toBe(1);
    expect(s.steps).toEqual(['a', 'b']);
    expect(s.documents).toEqual(['c']);
    expect(s.faq.map((f) => f.id)).toEqual([4]);
    expect(s.disclaimer?.id).toBe(5);
  });

  it('drops non-web links', () => {
    const page = insurancePageSchema.parse({
      updated_at: null,
      sections: [box(1, null, 'x', ['go', 'javascript:alert(1)']), box(2, null, 'x', ['ok', 'https://e.com'])],
    });
    expect(page.sections[0]?.linkUrl).toBeNull();
    expect(page.sections[1]?.linkUrl).toBe('https://e.com');
  });
});

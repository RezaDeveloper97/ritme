import { describe, expect, it } from 'vitest';

import type { SchemaField } from '../api/messages';
import { normalizeDraft, seedDraft } from './schema-form';

const fields: SchemaField[] = [
  { key: 'title', kind: 'text', nullable: false },
  { key: 'read_minutes', kind: 'integer', nullable: false, min: 1, max: 60 },
  { key: 'article_url', kind: 'url', nullable: true },
  { key: 'tags', kind: 'text_list', nullable: false },
];

describe('schema payload editor', () => {
  it('seeds every schema key and drops unknown ones', () => {
    expect(seedDraft(fields, { title: 'x', other: 1 })).toEqual({
      title: 'x',
      read_minutes: 1,
      article_url: null,
      tags: [],
    });
  });

  it('normalises blanks, integers and list lines', () => {
    expect(
      normalizeDraft(fields, {
        title: 't',
        read_minutes: '5',
        article_url: ' ',
        tags: [' a ', '', 'b'],
      }),
    ).toEqual({
      title: 't',
      read_minutes: 5,
      article_url: null,
      tags: ['a', 'b'],
    });
  });
});

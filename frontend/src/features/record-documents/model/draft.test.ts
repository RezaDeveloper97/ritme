import { describe, expect, it } from 'vitest';

import { addFiles, fileKindOf, overallProgress, validateDraft } from './draft';

const file = (name: string, type: string, size = 1000) => ({ name, type, size }) as File;
let n = 0;
const key = () => `k${++n}`;

describe('record upload draft', () => {
  it('recognises photos and PDFs only', () => {
    expect(fileKindOf(file('a.jpg', 'image/jpeg'))).toBe('image');
    expect(fileKindOf(file('a.pdf', 'application/pdf'))).toBe('pdf');
    expect(fileKindOf(file('scan.PDF', ''))).toBe('pdf');
    expect(fileKindOf(file('a.heic', 'image/heic'))).toBe('image');
    expect(fileKindOf(file('a.svg', 'image/svg+xml'))).toBeNull();
    expect(fileKindOf(file('x.pdf', 'text/html'))).toBeNull();
    expect(fileKindOf(file('a.docx', 'application/msword'))).toBeNull();
  });

  it('keeps valid files and reports the first problem', () => {
    const r = addFiles([], [file('a.doc', 'text/plain'), file('b.jpg', 'image/jpeg'), file('c.jpg', 'image/jpeg', 11 * 1024 * 1024)], key);
    expect(r.files.map((f) => f.file.name)).toEqual(['b.jpg']);
    expect(r.error).toBe('fileType');
  });

  it('caps a document at ten files', () => {
    const many = Array.from({ length: 12 }, (_, i) => file(`${i}.jpg`, 'image/jpeg'));
    const r = addFiles([], many, key);
    expect(r.files).toHaveLength(10);
    expect(r.error).toBe('tooMany');
  });

  it('needs a kind and a file', () => {
    expect(validateDraft({ kind: null, files: [] })).toBe('kind');
    expect(validateDraft({ kind: 'visit', files: [] })).toBe('files');
    expect(validateDraft({ kind: 'visit', files: [{ key: 'x', kind: 'pdf', file: file('a.pdf', 'application/pdf') }] })).toBeNull();
  });

  it('sums the progress over the files', () => {
    expect(overallProgress(0, 0.5, 2)).toBe(0.25);
    expect(overallProgress(2, 0, 2)).toBe(1);
    expect(overallProgress(0, 0, 0)).toBe(0);
  });
});

import { describe, expect, it } from 'vitest';

import { parseSse } from './stream';

const streamOf = (...chunks: string[]) =>
  new ReadableStream<Uint8Array>({
    start(c) {
      const enc = new TextEncoder();
      chunks.forEach((x) => c.enqueue(enc.encode(x)));
      c.close();
    },
  });

async function all(s: ReadableStream<Uint8Array>) {
  const out = [];
  for await (const e of parseSse(s)) out.push(e);
  return out;
}

describe('parseSse', () => {
  it('parses events split across chunks', async () => {
    const events = await all(streamOf('event: meta\ndata: {"conversation_id":1}\n\nevent: delta\nda', 'ta: {"text":"سل"}\n\nevent: done\ndata: {"emergency":false}\n\n'));
    expect(events).toEqual([
      { event: 'meta', data: { conversation_id: 1 } },
      { event: 'delta', data: { text: 'سل' } },
      { event: 'done', data: { emergency: false } },
    ]);
  });

  it('skips malformed blocks and handles CRLF', async () => {
    const events = await all(streamOf('event: delta\r\ndata: nope\r\n\r\nevent: error\r\ndata: {"error_code":"ai_failed"}\r\n\r\n'));
    expect(events).toEqual([{ event: 'error', data: { error_code: 'ai_failed' } }]);
  });
});

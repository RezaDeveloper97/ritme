import { ApiError } from '@/shared/api';
import { env } from '@/shared/config';
import { getAuthToken } from '@/shared/session';

/** One server-sent event of POST /assistant/conversations/{id}/messages. */
export interface SseEvent {
  event: string;
  data: unknown;
}

/**
 * Parses an SSE text stream into events. Blocks end with a blank line; `event:` names it, `data:` lines are JSON.
 * Exported for tests; `streamMessage` feeds it the response body.
 */
export async function* parseSse(body: ReadableStream<Uint8Array>): AsyncGenerator<SseEvent> {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  try {
    for (;;) {
      const { done, value } = await reader.read();
      buffer += decoder.decode(value, { stream: !done }).replace(/\r\n/g, '\n');
      let cut = buffer.indexOf('\n\n');
      while (cut >= 0) {
        const block = buffer.slice(0, cut);
        buffer = buffer.slice(cut + 2);
        const ev = parseBlock(block);
        if (ev) yield ev;
        cut = buffer.indexOf('\n\n');
      }
      if (done) {
        const ev = parseBlock(buffer);
        if (ev) yield ev;
        return;
      }
    }
  } finally {
    reader.releaseLock();
  }
}

function parseBlock(block: string): SseEvent | null {
  let event = 'message';
  const data: string[] = [];
  for (const line of block.split('\n')) {
    if (line.startsWith('event:')) event = line.slice(6).trim();
    else if (line.startsWith('data:')) data.push(line.slice(5).trimStart());
  }
  if (data.length === 0) return null;
  try {
    return { event, data: JSON.parse(data.join('\n')) as unknown };
  } catch {
    return null;
  }
}

/**
 * POST a user message and yield the SSE events. `fetch` + ReadableStream (not EventSource) because the request is a POST
 * with an `Authorization` header. A non-2xx answer (402 Plus quota, 403 consent, 422, 429) throws an {@link ApiError} with
 * the parsed JSON body — the same shape `apiClient` throws, so `plusDenialOf` / `getApiErrorCode` work on it.
 * `signal` aborts the stream (stop button, unmount).
 */
export async function* streamMessage(
  conversationId: number,
  input: { text: string; photoId?: number | null },
  signal: AbortSignal,
): AsyncGenerator<SseEvent> {
  const token = getAuthToken();
  const headers: Record<string, string> = {
    Accept: 'text/event-stream',
    'Content-Type': 'application/json',
    'Accept-Language': typeof document !== 'undefined' && document.documentElement.lang ? document.documentElement.lang : 'fa',
  };
  if (token) headers.Authorization = `Bearer ${token}`;
  let response: Response;
  try {
    response = await fetch(`${env.apiBaseUrl.replace(/\/+$/, '')}/assistant/conversations/${conversationId}/messages`, {
      method: 'POST',
      headers,
      body: JSON.stringify({ text: input.text, photo_id: input.photoId ?? undefined }),
      signal,
    });
  } catch (e) {
    if (signal.aborted) throw e;
    throw new ApiError('Network error', { code: 'network', sentToken: token });
  }
  if (!response.ok || !response.body) {
    let data: unknown = '';
    try {
      data = await response.json();
    } catch {
      /* not JSON */
    }
    throw new ApiError(`Request failed with status code ${response.status}`, {
      response: { status: response.status, data, headers: {} },
      sentToken: token,
    });
  }
  yield* parseSse(response.body);
}

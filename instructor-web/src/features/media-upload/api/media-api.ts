import { z } from 'zod';

import { api, ApiError, toApiError } from '@/shared/api';
import { INSTRUCTOR_API_BASE_URL } from '@/shared/config';
import { getAuthToken } from '@/shared/session';

/** backend-go internal/media JSON (instructor view). */
export const mediaSchema = z.object({
  id: z.number().int(),
  lesson_id: z.number().int().nullable(),
  kind: z.string(),
  mime: z.string().nullable(),
  status: z.enum(['uploading', 'processing', 'ready', 'failed']).or(z.string()),
  size_bytes: z.number().int(),
  offset_bytes: z.number().int(),
  upload_expires_at: z.string().nullable(),
});
export type Media = z.infer<typeof mediaSchema>;

const createSchema = z.object({
  media: mediaSchema,
  upload: z.object({ chunk_size: z.number().int(), max_chunk_bytes: z.number().int(), content_type: z.string() }),
});
const showSchema = z.object({ media: mediaSchema });
const playbackSchema = z.object({ playback: z.object({ url: z.string(), mime: z.string(), kind: z.string() }) });

export const CHUNK_CONTENT_TYPE = 'application/offset+octet-stream';

/** POST /media {lesson_id, size, mime?} — a new upload for one of the caller's lessons. */
export const createUpload = (input: { lesson_id: number; size: number; mime?: string }) =>
  api.post('/media', input, { schema: createSchema });

/** GET /media/{id} — the resume point (offset_bytes). */
export const fetchUpload = (id: number, signal?: AbortSignal) => api.get(`/media/${id}`, { schema: showSchema, signal });

export const deleteUpload = (id: number) => api.delete(`/media/${id}`);

export const fetchPlayback = (id: number) => api.get(`/media/${id}/playback`, { schema: playbackSchema });

/**
 * PATCH /media/{id} with one chunk (Upload-Offset). Raw body, so it bypasses the JSON client; errors become
 * ApiError like everywhere else (on 409 media_offset_mismatch the caller re-reads GET /media/{id}).
 */
export async function patchChunk(id: number, offset: number, chunk: Blob, signal?: AbortSignal): Promise<Media> {
  const headers: Record<string, string> = {
    Accept: 'application/json',
    'Accept-Language': 'fa',
    'Content-Type': CHUNK_CONTENT_TYPE,
    'Upload-Offset': String(offset),
  };
  const token = getAuthToken();
  if (token) headers.Authorization = `Bearer ${token}`;
  let res: Response;
  try {
    res = await fetch(`${INSTRUCTOR_API_BASE_URL}/media/${id}`, {
      method: 'PATCH',
      headers,
      body: chunk,
      credentials: 'same-origin',
      signal,
    });
  } catch (cause) {
    if (cause instanceof DOMException && cause.name === 'AbortError') throw cause;
    throw new ApiError({ status: 0, code: 'network_error' });
  }
  let body: unknown;
  try {
    body = await res.json();
  } catch {
    body = undefined;
  }
  if (!res.ok) throw toApiError(res.status, body, res.headers.get('retry-after'));
  const parsed = z.object({ data: showSchema }).safeParse(body);
  if (!parsed.success) throw new ApiError({ status: res.status, code: 'invalid_response' });
  return parsed.data.data.media;
}

'use client';

import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import type { AutopilotManifest } from '@/shared/pilot';

/**
 * What `POST /logs/voice/pilot` answers (backend-go/internal/voicelog/pilot.go): what was heard and a UI
 * autopilot script that fills it in. The steps stay `unknown` here on purpose — a model wrote them, so
 * `voiceScript` (model/pilot.ts) and then the engine validate them before anything runs.
 */
export const voicePilotSchema = z.object({
  transcript: z.string(),
  language: z.string(),
  script: z.object({ steps: z.array(z.unknown()) }),
});

export type VoicePilotResult = z.infer<typeof voicePilotSchema>;

/** Speech-to-text plus the script can take a while: above the server's 40 s AI bound. */
export const VOICE_UPLOAD_TIMEOUT_MS = 45_000;

export interface VoiceUpload {
  audio: Blob;
  durationMs: number;
  /** The log sheet's controls (vocabulary only — nothing the user logged, §11). */
  manifest: AutopilotManifest;
}

function fileName(type: string): string {
  if (type.includes('mp4')) return 'voice.m4a';
  if (type.includes('ogg')) return 'voice.ogg';
  if (type.includes('wav')) return 'voice.wav';
  return 'voice.webm';
}

/**
 * POST /logs/voice/pilot (multipart). The recording travels only in this request body — it is never put in a
 * URL, a cache or storage, and the server deletes it right after transcription. Nothing is saved server-side.
 */
export async function postVoicePilot({ audio, durationMs, manifest }: VoiceUpload): Promise<VoicePilotResult> {
  const form = new FormData();
  form.append('duration_ms', String(Math.round(durationMs)));
  form.append('manifest', JSON.stringify(manifest));
  form.append('audio', audio, fileName(audio.type));
  const { data } = await apiClient.post<ApiEnvelope<unknown>>('/logs/voice/pilot', form, { timeoutMs: VOICE_UPLOAD_TIMEOUT_MS });
  return voicePilotSchema.parse(data.data);
}

/** The upload mutation. Gate refusals (402 / 403) are handled by the caller through `useAiGate(...).handleError`. */
export function useVoicePilot() {
  return useMutation<VoicePilotResult, unknown, VoiceUpload>({
    mutationFn: postVoicePilot,
    retry: false,
  });
}

'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { plusKeys } from '@/entities/plus';
import { type ApiEnvelope, apiClient } from '@/shared/api';
import { plusDenialOf } from '@/shared/ui/plus-gate';

/** One reviewed-before-saving suggestion (`POST /logs/voice`, backend-go/internal/voicelog). */
export const voiceSuggestionSchema = z.object({
  category: z.string(),
  param: z.string(),
  item: z.string().nullable(),
  value: z.union([z.string(), z.number(), z.boolean()]),
  confidence: z.number(),
  label: z.string(),
});

export const voiceResultSchema = z.object({
  transcript: z.string(),
  language: z.string(),
  mode: z.string(),
  suggestions: z.array(voiceSuggestionSchema),
});

export type VoiceSuggestion = z.infer<typeof voiceSuggestionSchema>;
export type VoiceResult = z.infer<typeof voiceResultSchema>;

/** Speech-to-text can take a while: more than the default 15 s, and above the server's 40 s AI bound. */
export const VOICE_UPLOAD_TIMEOUT_MS = 45_000;

export interface VoiceUpload {
  audio: Blob;
  durationMs: number;
}

function fileName(type: string): string {
  if (type.includes('mp4')) return 'voice.m4a';
  if (type.includes('ogg')) return 'voice.ogg';
  if (type.includes('wav')) return 'voice.wav';
  return 'voice.webm';
}

/**
 * POST /logs/voice (multipart). The recording travels only in this request body — it is never put in a URL,
 * a cache or storage, and the server deletes it right after transcription. Nothing is saved server-side.
 */
export async function postVoiceLog({ audio, durationMs }: VoiceUpload): Promise<VoiceResult> {
  const form = new FormData();
  form.append('duration_ms', String(Math.round(durationMs)));
  form.append('audio', audio, fileName(audio.type));
  const { data } = await apiClient.post<ApiEnvelope<unknown>>('/logs/voice', form, { timeoutMs: VOICE_UPLOAD_TIMEOUT_MS });
  return voiceResultSchema.parse(data.data);
}

/** The upload mutation; a Plus refusal refreshes the entitlements so the tab's lock catches up. */
export function useVoiceLog() {
  const queryClient = useQueryClient();
  return useMutation<VoiceResult, unknown, VoiceUpload>({
    mutationFn: postVoiceLog,
    retry: false,
    onError: (error) => {
      if (plusDenialOf(error)) void queryClient.invalidateQueries({ queryKey: plusKeys.all });
    },
  });
}

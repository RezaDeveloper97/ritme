'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { contraceptionKeys } from '@/entities/contraception';
import { menopauseKeys } from '@/entities/menopause';
import { plusKeys } from '@/entities/plus';
import { type ApiEnvelope, apiClient } from '@/shared/api';
import { plusDenialOf } from '@/shared/ui/plus-gate';

const valueSchema = z.union([z.string(), z.number(), z.boolean()]);

/**
 * Where a suggestion is saved (CB-VOICE-01): `log` → the day's `PUT /logs/days` (with `voice_params`); the
 * diary targets → `POST /logs/voice/commit`.
 */
export const VOICE_TARGETS = ['log', 'hot_flash', 'pain_diary', 'pill', 'bladder'] as const;
export type VoiceTarget = (typeof VOICE_TARGETS)[number];
export type VoiceDiaryTarget = Exclude<VoiceTarget, 'log'>;

/** Another reading of an ambiguous word («بی‌حوصله» → sad / fatigue), same target as its suggestion. */
export const voiceOptionSchema = z.object({
  category: z.string(),
  param: z.string(),
  item: z.string().nullable(),
  value: valueSchema,
  label: z.string(),
});

/** One reviewed-before-saving suggestion (`POST /logs/voice`, backend-go/internal/voicelog). */
export const voiceSuggestionSchema = z.object({
  target: z.enum(VOICE_TARGETS).catch('log').default('log'),
  category: z.string(),
  param: z.string(),
  item: z.string().nullable(),
  value: valueSchema,
  confidence: z.number(),
  label: z.string(),
  options: z.array(voiceOptionSchema).default([]),
});

export const voiceResultSchema = z.object({
  transcript: z.string(),
  language: z.string(),
  mode: z.string(),
  suggestions: z.array(voiceSuggestionSchema),
});

export type VoiceSuggestion = z.infer<typeof voiceSuggestionSchema>;
export type VoiceOption = z.infer<typeof voiceOptionSchema>;
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

/** One diary item for `POST /logs/voice/commit` (hot flash, pain diary, pill, bladder). */
export interface VoiceCommitItem {
  category: VoiceDiaryTarget;
  param: string;
  value: string | number | boolean;
}

export const voiceCommitSchema = z.object({
  date: z.string(),
  saved: z.array(
    z.object({
      target: z.string(),
      category: z.string(),
      param: z.string(),
      value: valueSchema,
      label: z.string(),
    }),
  ),
});

export type VoiceCommitResult = z.infer<typeof voiceCommitSchema>;

/**
 * POST /logs/voice/commit — writes the confirmed diary items through their own services (CB-VOICE-01). Runs
 * after the day's PUT: a pain-diary score needs the pain location saved first.
 */
export async function postVoiceCommit(body: { date: string; items: VoiceCommitItem[] }): Promise<VoiceCommitResult> {
  const { data } = await apiClient.post<ApiEnvelope<unknown>>('/logs/voice/commit', body);
  return voiceCommitSchema.parse(data.data);
}

/** The diary commit; refreshes the menopause (hot flashes) and contraception (pill) caches it wrote to. */
export function useVoiceCommit() {
  const queryClient = useQueryClient();
  return useMutation<VoiceCommitResult, unknown, { date: string; items: VoiceCommitItem[] }>({
    mutationFn: postVoiceCommit,
    retry: false,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: menopauseKeys.all });
      void queryClient.invalidateQueries({ queryKey: contraceptionKeys.all });
    },
  });
}

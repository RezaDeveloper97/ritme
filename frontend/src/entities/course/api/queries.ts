'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient, getApiErrorCode, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { CourseOutline, LessonBlock, LessonDetail, MyCourses, Playback, UnlockedGrant } from '../model/types';
import { courseKeys } from './keys';
import { lessonDetailSchema, myCoursesSchema, outlineSchema, playbackSchema, progressSavedSchema, unlockedSchema } from './schema';

/*
 * `/api/v1/learning/*` (B-N8-01) and lesson playback (B-N8-02), Go only. Ids are
 * the only thing in a URL; signed media URLs are never logged.
 */

const noRetryOn4xx = (count: number, error: unknown) => {
  const s = getApiErrorStatus(error);
  return !(s && s >= 400 && s < 500) && count < 1;
};

/** GET /learning/courses — the continue card and every course open to the caller. */
export function useMyCourses() {
  return useQuery<MyCourses>({
    queryKey: courseKeys.mine(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/learning/courses');
      return myCoursesSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

/** GET /learning/courses/{id} — chapters and lessons with progress. */
export function useCourseOutline(id: number | null) {
  return useQuery<CourseOutline>({
    queryKey: courseKeys.course(id ?? 0),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/learning/courses/${id}`);
      return outlineSchema.parse(data.data);
    },
    enabled: isAuthenticated() && !!id,
    staleTime: 30_000,
    retry: noRetryOn4xx,
  });
}

/** GET /learning/lessons/{id} — 403 learning_access_expired / learning_chapter_locked, 404 without access. */
export function useLesson(id: number | null) {
  return useQuery<LessonDetail>({
    queryKey: courseKeys.lesson(id ?? 0),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/learning/lessons/${id}`);
      return lessonDetailSchema.parse(data.data);
    },
    enabled: isAuthenticated() && !!id,
    staleTime: 30_000,
    retry: noRetryOn4xx,
  });
}

/** Maps a lesson / playback error to the screen's blocked state (`null` = a plain load error). */
export function lessonBlock(error: unknown): LessonBlock | null {
  const code = getApiErrorCode(error);
  if (code === 'learning_access_expired') return 'expired';
  if (code === 'learning_chapter_locked') return 'locked';
  if (getApiErrorStatus(error) === 404) return 'notFound';
  return null;
}

/** GET /learning/lessons/{id}/playback — a 30-min signed stream URL (Range-capable). Refetch when it lapses. */
export function usePlayback(lessonId: number | null, enabled: boolean) {
  return useQuery<Playback>({
    queryKey: courseKeys.playback(lessonId ?? 0),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/learning/lessons/${lessonId}/playback`);
      return playbackSchema.parse(data.data);
    },
    enabled: isAuthenticated() && !!lessonId && enabled,
    staleTime: 20 * 60_000,
    gcTime: 25 * 60_000,
    refetchOnWindowFocus: false,
    retry: noRetryOn4xx,
  });
}

export interface ProgressInput {
  lessonId: number;
  positionSeconds: number;
  percent?: number;
  completed?: boolean;
}

/** PUT /learning/lessons/{id}/progress (monotonic on the server). `silent` saves don't refresh the lists. */
export function useSaveLessonProgress() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ProgressInput & { silent?: boolean }) => {
      const body: Record<string, unknown> = { position_seconds: Math.max(0, Math.min(86400, Math.floor(input.positionSeconds))) };
      if (input.percent !== undefined) body.percent = Math.max(0, Math.min(100, Math.round(input.percent)));
      if (input.completed) body.completed = true;
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/learning/lessons/${input.lessonId}/progress`, body);
      return progressSavedSchema.parse(data.data);
    },
    onSuccess: (_d, input) => {
      if (input.silent) return;
      void qc.invalidateQueries({ queryKey: courseKeys.mine() });
      void qc.invalidateQueries({ queryKey: [...courseKeys.all, 'course'] });
      void qc.invalidateQueries({ queryKey: courseKeys.lesson(input.lessonId) });
    },
  });
}

/** GET /learning/unlocked/{grant} — the «دوره برایت باز شد» screen (inbox link /learn/unlocked/{id}). */
export function useUnlockedGrant(grant: number | null) {
  return useQuery<UnlockedGrant>({
    queryKey: courseKeys.unlocked(grant ?? 0),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/learning/unlocked/${grant}`);
      return unlockedSchema.parse(data.data);
    },
    enabled: isAuthenticated() && !!grant,
    staleTime: 60_000,
    retry: noRetryOn4xx,
  });
}

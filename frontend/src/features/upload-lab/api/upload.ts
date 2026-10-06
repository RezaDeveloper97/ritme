'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';

import { type Lab, labKeys, labSchema } from '@/entities/lab';
import { type ApiEnvelope, apiClient } from '@/shared/api';

import { toFormData, type UploadDraft } from '../model/validation';

/** A 20 MB body on a slow connection needs more than the default 15 s. */
const UPLOAD_TIMEOUT_MS = 180_000;

/**
 * POST /labs (multipart, 202 with the lab — `queued`, or already `needs_review`
 * / `failed` when the server runs jobs inline). `progress` is the upload share
 * 0–1 while the body is on its way. Health data (§11): nothing is logged.
 */
export function useUploadLab() {
  const queryClient = useQueryClient();
  const [progress, setProgress] = useState(0);
  const mutation = useMutation<Lab, unknown, UploadDraft>({
    mutationFn: async (draft) => {
      setProgress(0);
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/labs', toFormData(draft), {
        timeoutMs: UPLOAD_TIMEOUT_MS,
        onUploadProgress: setProgress,
      });
      return labSchema.parse(data.data);
    },
    onSuccess: (lab) => {
      queryClient.setQueryData(labKeys.detail(lab.id), lab);
      void queryClient.invalidateQueries({ queryKey: labKeys.list() });
    },
  });
  return { ...mutation, progress };
}

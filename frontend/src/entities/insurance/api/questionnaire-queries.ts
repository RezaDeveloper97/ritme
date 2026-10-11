'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { QuestionnairePurpose, QuestionnaireStep, QuestionnaireUpdate } from '../model/questionnaire';
import { insuranceKeys } from './keys';
import { questionnaireListSchema, questionnaireSchema } from './questionnaire-schema';

/*
 * `/api/v1/insurance/questionnaires` (CB-INS-01, Go only). Self-tracking: answers stay in her record, nothing is sent
 * to an insurer. Health data (§11): never log a payload or a response.
 */

/** GET /insurance/questionnaires — her questionnaires, newest first (status page, questionnaire entry). */
export function useQuestionnaires(enabled = true) {
  return useQuery({
    queryKey: insuranceKeys.questionnaires(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/insurance/questionnaires');
      return questionnaireListSchema.parse(data.data);
    },
    enabled: enabled && isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** GET /insurance/questionnaires/{id} — answers, from-record flags, explanation (also the PDF source). */
export function useQuestionnaire(id: number | null) {
  return useQuery({
    queryKey: insuranceKeys.questionnaire(id ?? 0),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/insurance/questionnaires/${id}`);
      return questionnaireSchema.parse(data.data);
    },
    enabled: id !== null && isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** Every questionnaire write returns the whole questionnaire; cache it and refresh the list. */
function useQuestionnaireWrite<V>(call: (v: V) => Promise<unknown>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (v: V) => questionnaireSchema.parse(await call(v)),
    onSuccess: (q) => {
      queryClient.setQueryData(insuranceKeys.questionnaire(q.id), q);
      void queryClient.invalidateQueries({ queryKey: insuranceKeys.questionnaires() });
    },
  });
}

/** POST /insurance/questionnaires — a draft prefilled from her own record. */
export function useStartQuestionnaire() {
  return useQuestionnaireWrite(async (v: { purpose: QuestionnairePurpose }) => {
    const { data } = await apiClient.post<ApiEnvelope<unknown>>('/insurance/questionnaires', v);
    return data.data;
  });
}

/** PUT /insurance/questionnaires/{id} — answers / explanation (a confirmed one goes back to draft). */
export function useUpdateQuestionnaire(id: number) {
  return useQuestionnaireWrite(async (v: QuestionnaireUpdate) => {
    const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/insurance/questionnaires/${id}`, v);
    return data.data;
  });
}

/** POST /insurance/questionnaires/{id}/confirm — 422 while a question is unanswered. */
export function useConfirmQuestionnaire(id: number) {
  return useQuestionnaireWrite<void>(async () => {
    const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/insurance/questionnaires/${id}/confirm`);
    return data.data;
  });
}

/** POST /insurance/questionnaires/{id}/status — she records a step herself (after confirming). */
export function useAdvanceQuestionnaire(id: number) {
  return useQuestionnaireWrite(async (v: { status: QuestionnaireStep; occurred_on: string }) => {
    const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/insurance/questionnaires/${id}/status`, v);
    return data.data;
  });
}

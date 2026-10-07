'use client';

import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  DatingOffer,
  RecordCategories,
  RecordDocument,
  RecordDocumentInput,
  RecordExtras,
  RecordExtrasInput,
  StoredFile,
  TimelineKind,
  TimelinePage,
} from '../model/documents';
import {
  datingOfferSchema,
  type DocConsent,
  docConsentSchema,
  recordCategoriesSchema,
  recordDocumentSchema,
  recordExtrasSchema,
  reviewResultSchema,
  storedFileSchema,
  timelinePageSchema,
} from './documents-schema';
import { healthRecordKeys } from './keys';

/*
 * Record documents (canvas-build CB-REC-01/02 + CB-CORE-05 files), Go only, owner-only. Health data (§11): never log
 * a payload, a response or an error body; only ids and the filter kind travel in URLs.
 */

export const recordDocKeys = {
  all: [...healthRecordKeys.all, 'docs'] as const,
  categories: () => [...recordDocKeys.all, 'categories'] as const,
  extras: () => [...recordDocKeys.all, 'extras'] as const,
  timeline: (kind: TimelineKind) => [...recordDocKeys.all, 'timeline', kind] as const,
  detail: (id: number) => [...recordDocKeys.all, 'detail', id] as const,
  dating: (id: number) => [...recordDocKeys.all, 'dating', id] as const,
  consent: () => ['consents', 'ai_documents'] as const,
};

const DOC_CONSENT_CODE = 'ai_documents';
/** Photos up to 10 MB on a slow connection need more than the default 15 s. */
const UPLOAD_TIMEOUT_MS = 120_000;
/** A signed file link lives 5 minutes; re-read the document a little before. */
const LINK_STALE_MS = 4 * 60_000;

/** GET /health-record/categories — the home grid counts. */
export function useRecordCategories() {
  return useQuery<RecordCategories>({
    queryKey: recordDocKeys.categories(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/health-record/categories');
      return recordCategoriesSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** GET /health-record/extras — emergency-card allergy flag, surgeries, family history. */
export function useRecordExtras() {
  return useQuery<RecordExtras>({
    queryKey: recordDocKeys.extras(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/health-record/extras');
      return recordExtrasSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** PUT /health-record/extras — a present key replaces, an absent key keeps. */
export function useSaveRecordExtras() {
  const queryClient = useQueryClient();
  return useMutation<RecordExtras, unknown, RecordExtrasInput>({
    mutationFn: async (input) => {
      const body: Record<string, unknown> = {};
      if (input.allergiesOnEmergencyCard !== undefined) body.allergies_on_emergency_card = input.allergiesOnEmergencyCard;
      if (input.surgeries !== undefined) body.surgeries = input.surgeries;
      if (input.familyHistory !== undefined) body.family_history = input.familyHistory;
      const { data } = await apiClient.put<ApiEnvelope<unknown>>('/health-record/extras', body);
      return recordExtrasSchema.parse(data.data);
    },
    onSuccess: (extras) => {
      queryClient.setQueryData(recordDocKeys.extras(), extras);
      void queryClient.invalidateQueries({ queryKey: recordDocKeys.categories() });
    },
  });
}

/** GET /health-record/timeline — pages by `next_before` (a page never splits a day). */
export function useRecordTimeline(kind: TimelineKind) {
  return useInfiniteQuery<TimelinePage, unknown, { pages: TimelinePage[] }, ReturnType<typeof recordDocKeys.timeline>, string | null>({
    queryKey: recordDocKeys.timeline(kind),
    queryFn: async ({ pageParam }) => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/health-record/timeline', {
        params: { kind, before: pageParam ?? undefined, limit: 30 },
      });
      return timelinePageSchema.parse(data.data);
    },
    initialPageParam: null,
    getNextPageParam: (last) => last.nextBefore,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

async function fetchDocument(id: number): Promise<RecordDocument> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/health-record/documents/${id}`);
  return recordDocumentSchema.parse(data.data);
}

/**
 * GET /health-record/documents/{id}; polls every 2 s while the AI extraction is pending and `poll` is on (the screen
 * turns it off after a few minutes and offers a retry — audit L3).
 */
export function useRecordDocument(id: number, poll = true) {
  return useQuery<RecordDocument>({
    queryKey: recordDocKeys.detail(id),
    queryFn: () => fetchDocument(id),
    enabled: isAuthenticated() && id > 0,
    staleTime: LINK_STALE_MS,
    refetchInterval: (q) => (poll && q.state.data?.reviewState === 'pending' ? 2000 : false),
    retry: 1,
  });
}

function documentBody(input: RecordDocumentInput): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (input.kind !== undefined) body.kind = input.kind;
  if (input.title !== undefined) body.title = input.title;
  if (input.date !== undefined) body.date = input.date;
  if (input.endedOn !== undefined) body.ended_on = input.endedOn;
  if (input.centre !== undefined) body.centre = input.centre;
  if (input.doctor !== undefined) body.doctor = input.doctor;
  if (input.note !== undefined) body.note = input.note;
  if (input.fileIds !== undefined) body.file_ids = input.fileIds;
  if (input.confirm !== undefined) body.confirm = input.confirm;
  return body;
}

/** Everything that lists documents (grid counts, every timeline filter, the bloom record). */
function invalidateLists(queryClient: ReturnType<typeof useQueryClient>) {
  void queryClient.invalidateQueries({ queryKey: recordDocKeys.categories() });
  void queryClient.invalidateQueries({ queryKey: [...recordDocKeys.all, 'timeline'] });
}

/** POST /files {purpose: record_document} — one photo or PDF; the server sniffs, re-encodes and encrypts it. */
export function useUploadRecordFile() {
  return useMutation<StoredFile, unknown, { file: File; onProgress?: (f: number) => void }>({
    mutationFn: async ({ file, onProgress }) => {
      const form = new FormData();
      form.append('purpose', 'record_document');
      form.append('file', file);
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/files', form, {
        timeoutMs: UPLOAD_TIMEOUT_MS,
        onUploadProgress: onProgress,
      });
      return storedFileSchema.parse(data.data);
    },
  });
}

/**
 * Reads a signed file link (already vetted by `safeFileUrl`) into a Blob. The link is the credential, so no bearer,
 * no cookies, no referrer and no cache; the caller turns it into a short-lived object URL instead of ever opening the
 * signed URL in a tab (security audit CB-REC-04 L2).
 */
export async function fetchSignedFile(url: string): Promise<Blob> {
  const res = await fetch(url, { credentials: 'omit', referrerPolicy: 'no-referrer', cache: 'no-store' });
  if (!res.ok) throw new Error(`file ${res.status}`);
  return res.blob();
}

/** Saves a signed file through a blob + `a[download]`; the object URL is revoked right after the click. */
export async function downloadSignedFile(url: string, filename: string): Promise<void> {
  const blob = await fetchSignedFile(url);
  const objectUrl = URL.createObjectURL(blob);
  try {
    const a = document.createElement('a');
    a.href = objectUrl;
    a.download = filename;
    a.rel = 'noopener';
    document.body.appendChild(a);
    a.click();
    a.remove();
  } finally {
    setTimeout(() => URL.revokeObjectURL(objectUrl), 0);
  }
}

/** DELETE /files/{id} — drops an uploaded file that never made it onto a document. */
export async function deleteRecordFile(id: number): Promise<void> {
  await apiClient.delete(`/files/${id}`);
}

/** POST /health-record/documents (201). */
export function useCreateRecordDocument() {
  const queryClient = useQueryClient();
  return useMutation<RecordDocument, unknown, RecordDocumentInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/health-record/documents', documentBody(input));
      return recordDocumentSchema.parse(data.data);
    },
    onSuccess: (doc) => {
      queryClient.setQueryData(recordDocKeys.detail(doc.id), doc);
      invalidateLists(queryClient);
    },
  });
}

/** PUT /health-record/documents/{id} — partial. */
export function useUpdateRecordDocument(id: number) {
  const queryClient = useQueryClient();
  return useMutation<RecordDocument, unknown, RecordDocumentInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/health-record/documents/${id}`, documentBody(input));
      return recordDocumentSchema.parse(data.data);
    },
    onSuccess: (doc) => {
      queryClient.setQueryData(recordDocKeys.detail(id), doc);
      void queryClient.invalidateQueries({ queryKey: recordDocKeys.dating(id) });
      invalidateLists(queryClient);
    },
  });
}

/** DELETE /health-record/documents/{id} — with its files and links. */
export function useDeleteRecordDocument(id: number) {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, void>({
    mutationFn: async () => {
      await apiClient.delete(`/health-record/documents/${id}`);
    },
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: recordDocKeys.detail(id) });
      queryClient.removeQueries({ queryKey: recordDocKeys.dating(id) });
      invalidateLists(queryClient);
    },
  });
}

/** POST …/extract (202) — 402 plus_required for free users, 403 consent_required without the AI consent. */
export function useExtractRecordDocument(id: number) {
  const queryClient = useQueryClient();
  return useMutation<RecordDocument, unknown, void>({
    mutationFn: async () => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/health-record/documents/${id}/extract`, {});
      return recordDocumentSchema.parse(data.data);
    },
    onSuccess: (doc) => {
      queryClient.setQueryData(recordDocKeys.detail(id), doc);
      invalidateLists(queryClient);
    },
  });
}

/** POST …/review {fields, items} — needs_review → confirmed; answers the document and its dating offer. */
export function useReviewRecordDocument(id: number) {
  const queryClient = useQueryClient();
  return useMutation<
    { document: RecordDocument; dating: DatingOffer },
    unknown,
    { fields: Record<string, string | number | null>; items?: Record<string, string | null>[] }
  >({
    mutationFn: async (input) => {
      const body: Record<string, unknown> = { fields: input.fields };
      if (input.items !== undefined) body.items = input.items;
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/health-record/documents/${id}/review`, body);
      return reviewResultSchema.parse(data.data);
    },
    onSuccess: (r) => {
      queryClient.setQueryData(recordDocKeys.detail(id), r.document);
      queryClient.setQueryData(recordDocKeys.dating(id), r.dating);
      invalidateLists(queryClient);
    },
  });
}

/** GET …/dating — the pregnancy re-dating offer (never applied silently). */
export function useRecordDating(id: number, enabled: boolean) {
  return useQuery<DatingOffer>({
    queryKey: recordDocKeys.dating(id),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/health-record/documents/${id}/dating`);
      return datingOfferSchema.parse(data.data);
    },
    enabled: isAuthenticated() && enabled,
    staleTime: 30_000,
    retry: 1,
  });
}

/** POST …/dating {confirm: true} (apply) or DELETE …/dating (dismiss). */
export function useAnswerRecordDating(id: number) {
  const queryClient = useQueryClient();
  return useMutation<DatingOffer, unknown, 'apply' | 'dismiss'>({
    mutationFn: async (answer) => {
      const path = `/health-record/documents/${id}/dating`;
      const { data } =
        answer === 'apply'
          ? await apiClient.post<ApiEnvelope<unknown>>(path, { confirm: true })
          : await apiClient.delete<ApiEnvelope<unknown>>(path);
      return datingOfferSchema.parse(data.data);
    },
    onSuccess: (offer, answer) => {
      queryClient.setQueryData(recordDocKeys.dating(id), offer);
      void queryClient.invalidateQueries({ queryKey: recordDocKeys.detail(id) });
      // the pregnancy screens read the re-dated profile
      if (answer === 'apply') void queryClient.invalidateQueries({ queryKey: ['pregnancy'] });
    },
    onError: () => void queryClient.invalidateQueries({ queryKey: recordDocKeys.dating(id) }),
  });
}

/** GET /consents/ai_documents — the versioned consent text in force and whether it is granted. */
export function useDocAiConsent(enabled = true) {
  return useQuery<DocConsent>({
    queryKey: recordDocKeys.consent(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/consents/${DOC_CONSENT_CODE}`);
      return docConsentSchema.parse(data.data);
    },
    enabled: isAuthenticated() && enabled,
    staleTime: 60_000,
    retry: 1,
  });
}

/** PUT /consents/ai_documents {granted, version} — 409 consent_version_stale when a newer text is in force. */
export function useSetDocAiConsent() {
  const queryClient = useQueryClient();
  return useMutation<DocConsent, unknown, { granted: boolean; version: number }>({
    mutationFn: async ({ granted, version }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(
        `/consents/${DOC_CONSENT_CODE}`,
        granted ? { granted, version } : { granted },
      );
      return docConsentSchema.parse(data.data);
    },
    onSuccess: (consent) => {
      queryClient.setQueryData(recordDocKeys.consent(), consent);
      void queryClient.invalidateQueries({ queryKey: ['consents'], predicate: (q) => q.queryKey[1] !== DOC_CONSENT_CODE });
    },
    onError: () => void queryClient.invalidateQueries({ queryKey: recordDocKeys.consent() }),
  });
}

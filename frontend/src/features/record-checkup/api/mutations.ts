'use client';

import { type QueryClient, useMutation, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { LocalFilesError, type LocalFilesErrorCode } from '@/shared/lib/local-files';
import {
  type CheckupItem,
  type CheckupRecord,
  checkupAttachments,
  checkupKeys,
  checkupRecordResultSchema,
  fetchCheckup,
} from '@/entities/checkup';

import { type CheckupRecordInput, type CheckupRecordPatch, toCheckupRecordBody } from '../model/body';

/*
 * Create / update / delete a checkup record (v14_MarkDone, v14_SelfExam,
 * v14_History edit). Every mutation invalidates the plan list, the home card,
 * the type's detail and the history through `checkupKeys`.
 *
 * The optional report (photo / PDF) is stored ON THIS DEVICE ONLY, keyed by the
 * record id — the API only ever receives `has_attachment` (README privacy
 * promise, the MarkDone copy «فقط روی گوشی تو ذخیره می‌شود»). Nothing about the
 * file or the record's health content is logged (§11).
 */

export interface AttachmentInput {
  file: Blob;
  /** Display name; `File.name` is used when omitted. */
  name?: string;
}

export interface RecordCheckupOutcome {
  item: CheckupItem | null;
  record: CheckupRecord | null;
  /**
   * The record was saved but its report could not be kept on the device
   * (storage full / refused). The server flag has been reset to false; the UI
   * should say so rather than claim the file is attached.
   */
  attachmentError: LocalFilesErrorCode | null;
}

function invalidate(queryClient: QueryClient, typeId?: number): void {
  void queryClient.invalidateQueries({ queryKey: checkupKeys.listAll() });
  void queryClient.invalidateQueries({ queryKey: checkupKeys.home() });
  void queryClient.invalidateQueries({
    queryKey: typeId !== undefined ? checkupKeys.detail(typeId) : checkupKeys.detailAll(),
  });
  void queryClient.invalidateQueries({ queryKey: checkupKeys.recordsAll() });
  void queryClient.invalidateQueries({ queryKey: checkupKeys.attachmentsAll() });
}

/** Refuse a bad file BEFORE the record is written, so nothing half-saves. */
async function assertStorable(attachment: AttachmentInput | null | undefined): Promise<void> {
  if (!attachment) return;
  const invalid = checkupAttachments.check(attachment.file);
  if (invalid) throw new LocalFilesError(invalid);
  if (!(await checkupAttachments.isAvailable())) throw new LocalFilesError('unavailable');
}

/**
 * Keep the file under the record id; on failure flip the record's
 * `has_attachment` back so the server never claims a file that isn't there.
 */
async function storeAttachment(
  recordId: number | null,
  attachment: AttachmentInput,
): Promise<LocalFilesErrorCode | null> {
  let code: LocalFilesErrorCode | null = null;
  if (recordId === null) {
    code = 'failed';
  } else {
    try {
      await checkupAttachments.put(recordId, attachment.file, attachment.name);
      return null;
    } catch (error) {
      code = error instanceof LocalFilesError ? error.code : 'failed';
    }
  }
  if (recordId !== null) {
    try {
      await apiClient.put(`/checkups/records/${recordId}`, { has_attachment: false });
    } catch {
      // The list shows «پیوست» without a file; opening it reports "not on this device".
    }
  }
  return code;
}

export interface CreateCheckupRecordVars {
  /** The checkup type id. */
  typeId: number;
  input: CheckupRecordInput;
  attachment?: AttachmentInput | null;
}

/** POST /checkups/{id}/records → 201. */
export function useCreateCheckupRecord() {
  const queryClient = useQueryClient();
  return useMutation<RecordCheckupOutcome, unknown, CreateCheckupRecordVars>({
    mutationFn: async ({ typeId, input, attachment }) => {
      await assertStorable(attachment);
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(
        `/checkups/${typeId}/records`,
        toCheckupRecordBody(input, !!attachment),
      );
      const result = checkupRecordResultSchema.parse(data.data);
      if (!attachment) return { ...result, attachmentError: null };

      // The README only promises the recomputed item; until the response
      // carries the record (T-M4-02 note), find it in the type's latest records.
      let recordId = result.record?.id ?? null;
      if (recordId === null) {
        try {
          const detail = await fetchCheckup(typeId);
          // Highest id = the row just inserted, not an older one on the same date.
          const ids = detail.records
            .filter((r) => r.doneOn === input.doneOn && r.hasAttachment)
            .map((r) => r.id);
          recordId = ids.length > 0 ? Math.max(...ids) : null;
        } catch {
          recordId = null;
        }
      }
      const attachmentError = await storeAttachment(recordId, attachment);
      return { ...result, attachmentError };
    },
    onSuccess: (_data, { typeId }) => invalidate(queryClient, typeId),
  });
}

export interface UpdateCheckupRecordVars {
  recordId: number;
  /** The record's checkup type id, to refresh its detail. */
  typeId?: number;
  patch: CheckupRecordPatch;
  /** A new file replaces the stored one; `null` removes it; omitted = unchanged. */
  attachment?: AttachmentInput | null;
}

/** PUT /checkups/records/{recordId} — partial. */
export function useUpdateCheckupRecord() {
  const queryClient = useQueryClient();
  return useMutation<RecordCheckupOutcome, unknown, UpdateCheckupRecordVars>({
    mutationFn: async ({ recordId, patch, attachment }) => {
      await assertStorable(attachment);
      const hasAttachment = attachment === undefined ? undefined : attachment !== null;
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(
        `/checkups/records/${recordId}`,
        toCheckupRecordBody(patch, hasAttachment),
      );
      const result = checkupRecordResultSchema.parse(data.data);
      let attachmentError: LocalFilesErrorCode | null = null;
      if (attachment === null) await checkupAttachments.delete(recordId);
      else if (attachment) attachmentError = await storeAttachment(recordId, attachment);
      return { ...result, attachmentError };
    },
    onSuccess: (_data, { typeId }) => invalidate(queryClient, typeId),
  });
}

export interface DeleteCheckupRecordVars {
  recordId: number;
  typeId?: number;
}

/** DELETE /checkups/records/{recordId} — and the on-device report with it. */
export function useDeleteCheckupRecord() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, DeleteCheckupRecordVars>({
    mutationFn: async ({ recordId }) => {
      await apiClient.delete(`/checkups/records/${recordId}`);
      await checkupAttachments.delete(recordId);
    },
    onSuccess: (_data, { recordId, typeId }) => {
      queryClient.removeQueries({ queryKey: checkupKeys.attachment(recordId) });
      invalidate(queryClient, typeId);
    },
  });
}

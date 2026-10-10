import { isApiError } from '@/shared/api';

export type UploadErrorKey = 'tooLarge' | 'type' | 'expired' | 'quota' | 'busy' | 'storage';

/** The upload-specific message key for a /media failure (null → the generic error sentence). */
export function uploadErrorKey(error: unknown): UploadErrorKey | null {
  if (!isApiError(error)) return null;
  const fields = error.fieldErrors ?? {};
  if ('size' in fields) return 'quota';
  if ('lesson_id' in fields) return 'busy';
  if ('mime' in fields) return 'type';
  if (error.status === 413) return 'tooLarge';
  if (error.status === 415 || error.code === 'media_rejected') return 'type';
  if (error.status === 410 || error.code === 'media_not_uploading') return 'expired';
  if (error.status === 503) return 'storage';
  return null;
}

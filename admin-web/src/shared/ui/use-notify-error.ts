'use client';

import { useCallback } from 'react';

import { useErrorMessage } from '@/shared/i18n';

import { toast } from './toast';

/** `onError: notifyError` for mutations whose failure has no field to attach to. */
export function useNotifyError(): (error: unknown) => void {
  const describe = useErrorMessage();
  return useCallback((error: unknown) => toast.error(describe(error)), [describe]);
}

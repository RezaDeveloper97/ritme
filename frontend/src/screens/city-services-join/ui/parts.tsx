'use client';

import { useTranslations } from 'next-intl';

import { ApiError, getApiErrorMessage } from '@/shared/api';
import { Icon } from '@/shared/ui';

import type { SaveState } from '../model/use-autosave';

/** The user-facing text of a failed request: the API's localized message, else a generic one. */
export function useErrorText() {
  const t = useTranslations('cityJoin.errors');
  return (error: unknown): string => {
    if (error instanceof ApiError && !error.response) return t('network');
    return getApiErrorMessage(error) ?? t('unknown');
  };
}

/** «Draft saved» / «Saving…» line under a step. Silent until something was typed. */
export function SaveHint({ state }: { state: SaveState }) {
  const t = useTranslations('cityJoin');
  if (state === 'idle') return null;
  const text = state === 'saving' ? t('saving') : state === 'saved' ? t('saved') : t('saveError');
  return (
    <p className="cjn-save" data-state={state} role="status" aria-live="polite">
      <Icon name={state === 'saved' ? 'check' : state === 'error' ? 'warning' : 'loader'} size={14} />
      {text}
    </p>
  );
}

export function FieldError({ children }: { children: string | null | undefined }) {
  return children ? (
    <p role="alert" className="cjn-err">
      {children}
    </p>
  ) : null;
}

'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useState, type FormEvent } from 'react';

import { APPLY_LIMITS, applyInstructor, instructorKeys, type Instructor } from '@/entities/instructor';
import { isApiError } from '@/shared/api';
import { useErrorMessage } from '@/shared/i18n';
import { toPersianDigits } from '@/shared/lib';
import { Button, TextField } from '@/shared/ui';

import { toApplyInput, validateApply, type ApplyDraft, type ApplyErrors } from '../model/validate';

interface ApplyFormProps {
  initial?: Instructor | null;
  submitLabel: string;
  onDone?: () => void;
  onCancel?: () => void;
}

/** POST /apply — display name (required), title and short bio. */
export function ApplyForm({ initial, submitLabel, onDone, onCancel }: ApplyFormProps) {
  const t = useTranslations('apply');
  const errorMessage = useErrorMessage();
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<ApplyDraft>({
    display_name: initial?.display_name ?? '',
    title: initial?.title ?? '',
    bio: initial?.bio ?? '',
  });
  const [errors, setErrors] = useState<ApplyErrors>({});
  const [serverFields, setServerFields] = useState<Partial<Record<keyof ApplyDraft, string>>>({});

  const mutation = useMutation({
    mutationFn: () => applyInstructor(toApplyInput(draft)),
    onSuccess: (me) => {
      queryClient.setQueryData(instructorKeys.me(), me);
      onDone?.();
    },
    onError: (err) => {
      if (isApiError(err) && err.status === 422) {
        setServerFields({
          display_name: err.field('display_name'),
          title: err.field('title'),
          bio: err.field('bio'),
        });
      }
    },
  });

  const set = (key: keyof ApplyDraft) => (value: string) => {
    setDraft((d) => ({ ...d, [key]: value }));
    setErrors((e) => ({ ...e, [key]: undefined }));
    setServerFields((e) => ({ ...e, [key]: undefined }));
  };

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const found = validateApply(draft);
    setErrors(found);
    if (Object.keys(found).length === 0) mutation.mutate();
  };

  const fieldError = (key: keyof ApplyDraft) =>
    errors[key] ? t(`errors.${errors[key]}`) : serverFields[key];

  const failed = mutation.isError && !(isApiError(mutation.error) && mutation.error.status === 422);

  return (
    <form className="apply-form" onSubmit={submit} noValidate>
      <TextField
        label={t('nameLabel')}
        hint={t('nameHint')}
        error={fieldError('display_name')}
        value={draft.display_name}
        onChange={(e) => set('display_name')(e.target.value)}
        maxLength={APPLY_LIMITS.displayNameMax}
        autoComplete="name"
        required
      />
      <TextField
        label={t('titleLabel')}
        hint={t('titleHint')}
        error={fieldError('title')}
        value={draft.title}
        onChange={(e) => set('title')(e.target.value)}
        maxLength={APPLY_LIMITS.titleMax}
      />
      <TextField
        multiline
        rows={4}
        label={t('bioLabel')}
        hint={t('bioHint', {
          count: toPersianDigits(Array.from(draft.bio).length),
          max: toPersianDigits(APPLY_LIMITS.bioMax),
        })}
        error={fieldError('bio')}
        value={draft.bio}
        onChange={(e) => set('bio')(e.target.value)}
        maxLength={APPLY_LIMITS.bioMax}
      />
      {failed ? (
        <p className="form-error" role="alert">
          {errorMessage(mutation.error)}
        </p>
      ) : null}
      <div className="apply-form__actions">
        <Button type="submit" block loading={mutation.isPending}>
          {submitLabel}
        </Button>
        {onCancel ? (
          <Button variant="ghost" block onClick={onCancel} disabled={mutation.isPending}>
            {t('cancel')}
          </Button>
        ) : null}
      </div>
    </form>
  );
}

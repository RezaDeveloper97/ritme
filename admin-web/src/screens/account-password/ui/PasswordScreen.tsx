'use client';

import { useTranslations } from 'next-intl';
import { useState, type FormEvent } from 'react';

import { fieldError } from '@/shared/api';
import { Button, Panel, TextInput, toast, useNotifyError } from '@/shared/ui';

import { useChangePassword } from '../api/password';

/** /account/password — change your own password (Blade account.password). */
export function PasswordScreen() {
  const t = useTranslations('password');
  const notifyError = useNotifyError();
  const change = useChangePassword();
  const [current, setCurrent] = useState('');
  const [password, setPassword] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const mismatch = confirmation !== '' && confirmation !== password;

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (mismatch) return;
    change.mutate(
      { current_password: current, password, password_confirmation: confirmation },
      {
        onSuccess: () => {
          toast.success(t('changed'));
          setCurrent('');
          setPassword('');
          setConfirmation('');
        },
        onError: notifyError,
      },
    );
  };

  return (
    <Panel title={t('title')} className="max-w-xl" bodyClassName="">
      <form onSubmit={submit}>
        <div className="flex flex-col gap-4 p-4 sm:p-5">
          <p className="field-hint m-0">{t('intro')}</p>
          <TextInput
            label={t('current')}
            type="password"
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
            required
            autoComplete="current-password"
            error={fieldError(change.error, 'current_password')}
          />
          <TextInput
            label={t('new')}
            hint={t('newHint')}
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            minLength={8}
            maxLength={72}
            autoComplete="new-password"
            error={fieldError(change.error, 'password')}
          />
          <TextInput
            label={t('confirm')}
            type="password"
            value={confirmation}
            onChange={(e) => setConfirmation(e.target.value)}
            required
            autoComplete="new-password"
            error={mismatch ? t('mismatch') : fieldError(change.error, 'password_confirmation')}
          />
        </div>
        <div className="form-actions">
          <Button type="submit" variant="primary" loading={change.isPending} disabled={mismatch}>
            {t('submit')}
          </Button>
        </div>
      </form>
    </Panel>
  );
}

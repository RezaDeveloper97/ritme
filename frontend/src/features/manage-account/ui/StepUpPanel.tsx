'use client';

import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import { requestStepUpCode, verifyStepUpCode } from '@/shared/api';
import { toAsciiDigits } from '@/shared/lib/phone';
import { PrimaryButton, SecondaryButton } from '@/shared/ui';

import { stepUpErrorKey } from '../model/stepUpErrors';
import './otp.css';
import '@/shared/ui/styles/del.css';
import './stepup.css';

const CODE_LENGTH = 4;

interface StepUpPanelProps {
  /** One line on why the code is needed (export / delete). */
  body: string;
  /** Called once with the single-use step-up token. */
  onVerified: (token: string) => void;
  onCancel: () => void;
  /** The protected action is running (after a verified code). */
  busy?: boolean;
  /** The protected action failed; shown under the field. */
  actionError?: string | null;
}

/**
 * The step-up body (T-M12-06): send a code to the user's own number, enter the
 * four digits, get a single-use token. Deliberately has no autopilot name on
 * any control — it is an auth step in front of export / delete (§11.6).
 */
export function StepUpPanel({ body, onVerified, onCancel, busy = false, actionError }: StepUpPanelProps) {
  const t = useTranslations('profile.stepUp');
  const fieldId = useId();
  const [sent, setSent] = useState(false);
  const [code, setCode] = useState('');

  const send = useMutation<void, unknown, void>({
    mutationFn: requestStepUpCode,
    onSuccess: () => {
      setSent(true);
      setCode('');
    },
  });
  const verify = useMutation<string, unknown, string>({
    mutationFn: verifyStepUpCode,
    onSuccess: (token) => {
      setCode('');
      onVerified(token);
    },
  });

  const working = send.isPending || verify.isPending || busy;
  const error = verify.isError
    ? t(stepUpErrorKey(verify.error, 'verify'))
    : send.isError
      ? t(stepUpErrorKey(send.error, 'send'))
      : actionError ?? null;

  const submit = () => {
    if (working || code.length !== CODE_LENGTH) return;
    verify.mutate(code);
  };

  return (
    <>
      <div className="del-title">{t('title')}</div>
      <p className="del-body">{sent ? t('sent') : body}</p>

      {sent ? (
        <form
          className="stepup-form"
          onSubmit={(event) => {
            event.preventDefault();
            submit();
          }}
        >
          <label htmlFor={fieldId} className="stepup-label">
            {t('codeLabel')}
          </label>
          <input
            id={fieldId}
            className="otp-box stepup-code"
            inputMode="numeric"
            autoComplete="one-time-code"
            dir="ltr"
            maxLength={CODE_LENGTH}
            value={code}
            disabled={working}
            onChange={(event) => {
              verify.reset();
              setCode(toAsciiDigits(event.target.value).replace(/\D/g, '').slice(0, CODE_LENGTH));
            }}
          />
        </form>
      ) : null}

      {error ? (
        <p className="del-error" role="alert">
          {error}
        </p>
      ) : null}

      <div className="del-btns">
        {sent ? (
          <>
            <PrimaryButton onClick={submit} disabled={code.length !== CODE_LENGTH} loading={verify.isPending || busy}>
              {verify.isPending || busy ? t('verifying') : t('confirm')}
            </PrimaryButton>
            <SecondaryButton variant="text" onClick={() => send.mutate()} disabled={working}>
              {send.isPending ? t('sending') : t('resend')}
            </SecondaryButton>
          </>
        ) : (
          <PrimaryButton onClick={() => send.mutate()} loading={send.isPending}>
            {send.isPending ? t('sending') : t('send')}
          </PrimaryButton>
        )}
        <button type="button" className="del-cancel" onClick={onCancel} disabled={busy}>
          {t('cancel')}
        </button>
      </div>
    </>
  );
}

'use client';

import { useTranslations } from 'next-intl';
import { useRouter, useSearchParams } from 'next/navigation';
import { useEffect, useState, type FormEvent } from 'react';

import { isApiError } from '@/shared/api';
import { useErrorMessage } from '@/shared/i18n';
import { Button, Switch, TextInput } from '@/shared/ui';

import { fetchMe } from '../api/auth-api';
import { useLogin } from '../api/queries';
import { safeNext } from '../lib/safe-next';

export function LoginForm() {
  const t = useTranslations('auth');
  const describe = useErrorMessage();
  const router = useRouter();
  const search = useSearchParams();
  const next = safeNext(search.get('next'));
  const signedOut = search.get('signed_out') === '1';

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [remember, setRemember] = useState(false);
  const login = useLogin();

  // Already signed in (e.g. opened /login in a second tab): go straight on.
  useEffect(() => {
    const ctrl = new AbortController();
    fetchMe({ signal: ctrl.signal, probe: true })
      .then(() => router.replace(next))
      .catch(() => undefined);
    return () => ctrl.abort();
  }, [next, router]);

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    login.mutate(
      { email: email.trim(), password, remember },
      { onSuccess: () => router.replace(next) },
    );
  };

  const error = login.error;
  const fieldError = (name: string) => (isApiError(error) ? error.field(name) : undefined);
  // Field-level problems show under their field; everything else as one line.
  const formError = error && !(isApiError(error) && error.code === 'validation_failed') ? describe(error) : null;

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-4" noValidate>
      {signedOut && !error ? (
        <p className="m-0 rounded-[10px] bg-green-soft px-3 py-2 text-green-deep" role="status">
          {t('loggedOut')}
        </p>
      ) : null}
      {formError ? (
        <p className="m-0 rounded-[10px] bg-danger-soft px-3 py-2 text-danger-deep" role="alert">
          {formError}
        </p>
      ) : null}
      <TextInput
        label={t('email')}
        type="email"
        name="email"
        autoComplete="username"
        dir="ltr"
        required
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        error={fieldError('email')}
      />
      <TextInput
        label={t('password')}
        type="password"
        name="password"
        autoComplete="current-password"
        dir="ltr"
        required
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        error={fieldError('password')}
      />
      <Switch label={t('remember')} hint={t('rememberHint')} checked={remember} onChange={setRemember} />
      <Button
        type="submit"
        variant="gradient"
        className="mt-1 min-h-11 text-[15px]"
        loading={login.isPending}
        disabled={!email.trim() || !password}
      >
        {login.isPending ? t('submitting') : t('submit')}
      </Button>
    </form>
  );
}

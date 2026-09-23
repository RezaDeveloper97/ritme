'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useEffect, useState, type FormEvent } from 'react';

import { fieldError } from '@/shared/api';
import { blankToNull, formatDate, formatNumber } from '@/shared/lib';
import {
  Badge,
  Button,
  confirm,
  ErrorState,
  PageHeader,
  Panel,
  Select,
  Skeleton,
  TextInput,
  toast,
  useNotifyError,
} from '@/shared/ui';

import { usersApi, type UserDetail } from '../api/users';

/** /users/:id — stats, edit (name, subscription, goal), block/unblock, delete (Blade users.show). */
export function UserDetailScreen({ id }: { id: number }) {
  const t = useTranslations('users');
  const query = usersApi.useDetail(id);

  if (query.error && !query.data) {
    return (
      <div className="flex flex-col gap-4">
        <PageHeader title={t('detailTitle')} backHref="/users" backLabel={t('backToList')} />
        <Panel>
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        </Panel>
      </div>
    );
  }
  if (!query.data) {
    return (
      <div className="flex flex-col gap-4" aria-busy="true">
        <Skeleton className="h-8 w-60" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }
  return <UserDetail detail={query.data} />;
}

function UserDetail({ detail }: { detail: UserDetail }) {
  const t = useTranslations('users');
  const tc = useTranslations('common');
  const locale = useLocale();
  const router = useRouter();
  const notifyError = useNotifyError();
  const { user, profile, stats, options } = detail;

  const block = usersApi.useAction('block');
  const unblock = usersApi.useAction('unblock');
  const remove = usersApi.useRemove();
  const save = usersApi.useSave(user.id);

  const [name, setName] = useState(user.name ?? '');
  const [subscription, setSubscription] = useState(user.subscription_type);
  const [goal, setGoal] = useState(user.user_goal);
  // Follow the server after a save / block refetch.
  useEffect(() => {
    setName(user.name ?? '');
    setSubscription(user.subscription_type);
    setGoal(user.user_goal);
  }, [user.name, user.subscription_type, user.user_goal]);

  const onBlock = async () => {
    if (user.is_blocked) {
      unblock.mutate({ id: user.id }, { onSuccess: () => toast.success(t('unblocked')), onError: notifyError });
      return;
    }
    if (!(await confirm({ message: t('confirmBlock'), confirmLabel: t('block'), tone: 'danger' }))) return;
    block.mutate({ id: user.id }, { onSuccess: () => toast.success(t('blockedToast')), onError: notifyError });
  };

  const onDelete = async () => {
    if (!(await confirm({ message: t('confirmDelete'), confirmLabel: t('delete'), tone: 'danger' }))) return;
    remove.mutate(user.id, {
      onSuccess: () => {
        toast.success(t('deleted'));
        router.replace('/users');
      },
      onError: notifyError,
    });
  };

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    save.mutate(
      { name: blankToNull(name), subscription_type: subscription, user_goal: goal },
      { onSuccess: () => toast.success(tc('saved')), onError: notifyError },
    );
  };

  const n = (v: number) => formatNumber(v, locale);
  const title = user.name || user.mobile || `#${n(user.id)}`;

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title={title}
        backHref="/users"
        backLabel={t('backToList')}
        meta={
          <>
            {user.is_blocked ? (
              <Badge tone="red">{t('blockedSince', { date: formatDate(user.blocked_at, locale) })}</Badge>
            ) : (
              <Badge tone="green">{tc('active')}</Badge>
            )}
            <span>{t('joinedOn', { date: formatDate(user.created_at, locale) })}</span>
          </>
        }
        actions={
          <>
            <Button variant={user.is_blocked ? 'default' : 'danger'} onClick={onBlock} loading={block.isPending || unblock.isPending}>
              {user.is_blocked ? t('unblock') : t('block')}
            </Button>
            <Button variant="danger" onClick={onDelete} loading={remove.isPending}>
              {t('delete')}
            </Button>
          </>
        }
      />

      <Panel bodyClassName="">
        <dl className="ledger m-0">
          {(
            [
              ['healthLogs', stats.health_logs],
              ['reminders', stats.reminders],
              ['notifications', stats.notifications],
            ] as const
          ).map(([key, value]) => (
            <div key={key} className="ledger-cell">
              <dt className="text-[13px] font-semibold text-ink-3">{t(`stats.${key}`)}</dt>
              <dd className="m-0 mt-1">
                <span className="ledger-value">{n(value)}</span>
              </dd>
            </div>
          ))}
        </dl>
      </Panel>

      <Panel title={t('account')} bodyClassName="">
        <form onSubmit={onSubmit}>
          <div className="form-grid p-4 sm:p-5">
            <TextInput label={t('mobile')} value={user.mobile ?? '—'} disabled dir="ltr" />
            <TextInput
              label={t('name')}
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={255}
              error={fieldError(save.error, 'name')}
            />
            <Select
              label={t('subscription')}
              value={subscription}
              onChange={(e) => setSubscription(e.target.value)}
              options={options.subscription_types}
              error={fieldError(save.error, 'subscription_type')}
            />
            <Select
              label={t('goal')}
              value={goal}
              onChange={(e) => setGoal(e.target.value)}
              options={options.user_goals}
              error={fieldError(save.error, 'user_goal')}
            />
          </div>
          <div className="form-actions">
            <Button type="submit" variant="primary" loading={save.isPending}>
              {tc('saveChanges')}
            </Button>
          </div>
        </form>
      </Panel>

      <Panel title={t('healthProfile')} bodyClassName="">
        {profile ? (
          <dl className="facts">
            <div>
              <dt>{t('profile.birthday')}</dt>
              <dd>{formatDate(profile.birthday, locale) || '—'}</dd>
            </div>
            <div>
              <dt>{t('profile.lastPeriod')}</dt>
              <dd>{formatDate(profile.last_period_start, locale) || '—'}</dd>
            </div>
            <div>
              <dt>{t('profile.cycleLength')}</dt>
              <dd>{profile.cycle_duration ? t('profile.days', { count: profile.cycle_duration }) : '—'}</dd>
            </div>
            <div>
              <dt>{t('profile.periodLength')}</dt>
              <dd>{profile.period_duration ? t('profile.days', { count: profile.period_duration }) : '—'}</dd>
            </div>
          </dl>
        ) : (
          <p className="m-0 p-4 text-ink-3 sm:p-5">{t('noProfile')}</p>
        )}
      </Panel>
    </div>
  );
}

'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { RequireSuper } from '@/features/auth';
import type { AdminRole, Permissions } from '@/entities/admin';
import { Button, Panel, PageHeader, Select, Skeleton, ErrorState, confirm, toast, useNotifyError } from '@/shared/ui';

import { useResetRole, useRoles, useSaveRole, type Capability, type RoleRow } from '../api/access';

/** /roles — the permission matrix (Admin_Roles artboard): read for every role, edit for the custom ones. */
export function RolesScreen() {
  return (
    <RequireSuper>
      <Roles />
    </RequireSuper>
  );
}

const MARK = { manage: '●', view: '○' } as const;

function Roles() {
  const t = useTranslations('access');
  const query = useRoles();

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('rolesTitle')} meta={t('rolesSubtitle')} />
      {query.isPending ? (
        <Skeleton className="h-80 w-full" />
      ) : query.error ? (
        <ErrorState error={query.error} onRetry={() => query.refetch()} />
      ) : (
        <>
          <Matrix capabilities={query.data.capabilities} roles={query.data.roles.filter((r) => r.key !== 'super')} />
          <RoleEditor capabilities={query.data.capabilities} roles={query.data.roles.filter((r) => r.editable)} />
        </>
      )}
    </div>
  );
}

function Matrix({ capabilities, roles }: { capabilities: Capability[]; roles: RoleRow[] }) {
  const t = useTranslations('access');
  const tr = useTranslations('roles');
  return (
    <Panel
      title={t('rolesTitle')}
      actions={
        <span className="text-sm text-ink-3">
          {t('legend')} · {t('customizedLegend')}
        </span>
      }
      bodyClassName=""
    >
      <div className="overflow-x-auto">
        <table className="data-table access-matrix">
          <caption className="sr-only">{t('rolesTitle')}</caption>
          <thead>
            <tr>
              <th scope="col">{t('capability')}</th>
              {roles.map((r) => (
                <th key={r.key} scope="col" className="text-center whitespace-nowrap">
                  {tr(r.key)}
                  {r.customized ? (
                    <span className="access-dot" aria-label={t('customized')} title={t('customized')} />
                  ) : null}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {capabilities.map((c) => (
              <tr key={c.key}>
                <th scope="row" className="font-medium">
                  {t(`caps.${c.key}` as 'caps.users')}
                </th>
                {roles.map((r) => {
                  const level = r.permissions[c.key];
                  return (
                    <td
                      key={r.key}
                      className="text-center"
                      data-level={level ?? 'none'}
                      aria-label={`${tr(r.key)} — ${level === 'manage' ? t('full') : level === 'view' ? t('viewOnly') : t('none')}`}
                    >
                      <span className="access-mark" data-level={level ?? 'none'} aria-hidden="true">
                        {level ? MARK[level] : '—'}
                      </span>
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

function RoleEditor({ capabilities, roles }: { capabilities: Capability[]; roles: RoleRow[] }) {
  const t = useTranslations('access');
  const tr = useTranslations('roles');
  const notifyError = useNotifyError();
  const [roleKey, setRoleKey] = useState<AdminRole | ''>(roles[0]?.key ?? '');
  const role = roles.find((r) => r.key === roleKey);
  const [draft, setDraft] = useState<Permissions | null>(null);
  const save = useSaveRole();
  const reset = useResetRole();
  if (!role) return null;
  const perms = draft ?? role.permissions;

  const levelLabel = (l: string) => (l === 'manage' ? t('full') : l === 'view' ? t('viewOnly') : t('none'));

  return (
    <Panel title={t('edit')}>
      <div className="flex flex-col gap-4">
        <Select
          label={t('role')}
          value={roleKey}
          onChange={(e) => {
            setRoleKey(e.target.value as AdminRole);
            setDraft(null);
          }}
          options={roles.map((r) => ({ value: r.key, label: tr(r.key) }))}
        />
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {capabilities.map((c) => (
            <Select
              key={c.key}
              label={t(`caps.${c.key}` as 'caps.users')}
              value={perms[c.key] ?? 'none'}
              onChange={(e) => {
                const next = { ...perms };
                if (e.target.value === 'none') delete next[c.key];
                else next[c.key] = e.target.value as 'view' | 'manage';
                setDraft(next);
              }}
              options={['none', ...c.levels].map((l) => ({
                value: l,
                label: levelLabel(l),
              }))}
            />
          ))}
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            variant="primary"
            loading={save.isPending}
            disabled={!draft}
            onClick={() =>
              save.mutate(
                {
                  role: role.key,
                  permissions: Object.fromEntries(capabilities.map((c) => [c.key, perms[c.key] ?? 'none'])),
                },
                {
                  onSuccess: () => {
                    toast.success(t('saved'));
                    setDraft(null);
                  },
                  onError: notifyError,
                },
              )
            }
          >
            {t('save')}
          </Button>
          {draft ? (
            <Button variant="ghost" onClick={() => setDraft(null)}>
              {t('cancel')}
            </Button>
          ) : null}
          {role.customized ? (
            <Button
              variant="ghost"
              loading={reset.isPending}
              onClick={async () => {
                if (!(await confirm({ message: t('confirmReset') }))) return;
                reset.mutate(role.key, {
                  onSuccess: () => {
                    toast.success(t('resetDone'));
                    setDraft(null);
                  },
                  onError: notifyError,
                });
              }}
            >
              {t('reset')}
            </Button>
          ) : null}
        </div>
      </div>
    </Panel>
  );
}

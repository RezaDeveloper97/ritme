'use client';

import { useTranslations } from 'next-intl';

import { useCan } from '@/features/auth';
import { LoadGate, PageHeader, Panel, Switch, toast, useNotifyError } from '@/shared/ui';

import { MODULE_GROUPS, useModules, useSaveModules, type ModuleKey } from '../api/modules';

/**
 * /app-modules — switch whole app modules on and off. An off module disappears from the sign-up flow and from the
 * app, and its API answers 403 module_disabled. Every flip saves at once and is audited.
 */
export function AppModulesScreen() {
  const t = useTranslations('appModules');
  const query = useModules();
  const save = useSaveModules();
  const canManage = useCan('config', 'manage');
  const notifyError = useNotifyError();

  const flip = (key: ModuleKey, on: boolean) => {
    if (!canManage || save.isPending) return;
    save.mutate(
      { [key]: on },
      { onSuccess: () => toast.success(t(on ? 'enabled' : 'disabled', { name: t(`items.${key}.title`) })), onError: notifyError },
    );
  };

  return (
    <LoadGate queries={[query]} header={<PageHeader title={t('title')} />}>
      {() => (
        <div className="flex flex-col gap-5">
          <PageHeader title={t('title')} />
          <p className="m-0 text-sm text-ink-3">{t('intro')}</p>
          {!canManage ? <p className="m-0 text-sm text-ink-3">{t('readOnly')}</p> : null}
          {MODULE_GROUPS.map((group) => (
            <Panel key={group.key} title={t(`groups.${group.key}`)} bodyClassName="">
              <div className="flex flex-col gap-5 p-4 sm:p-5">
                {group.modules.map((key) => (
                  <Switch
                    key={key}
                    label={t(`items.${key}.title`)}
                    hint={t(`items.${key}.hint`)}
                    checked={query.data?.modules[key] ?? true}
                    disabled={!canManage || save.isPending}
                    onChange={(on) => flip(key, on)}
                  />
                ))}
              </div>
            </Panel>
          ))}
        </div>
      )}
    </LoadGate>
  );
}

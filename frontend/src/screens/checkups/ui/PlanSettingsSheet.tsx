'use client';

import { useTranslations } from 'next-intl';

import { type CheckupItem, checkupIcon, useCheckup } from '@/entities/checkup';
import { useUpdateCheckupSettings } from '@/features/manage-custom-checkup';
import { AppSheet } from '@/shared/sheet';
import { Icon } from '@/shared/ui';

function Switch({
  on,
  label,
  disabled,
  onClick,
}: {
  on: boolean;
  label: string;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={on}
      aria-label={label}
      disabled={disabled}
      className="rmd-switch"
      onClick={onClick}
    >
      <span className="rmd-switch-knob" />
    </button>
  );
}

/** One type: «در برنامه من» + «یادآوری». The reminder flag is only on the detail. */
function SettingsRow({ item }: { item: CheckupItem }) {
  const t = useTranslations('checkups.settings');
  const detail = useCheckup(item.id);
  const update = useUpdateCheckupSettings();
  const enabled = detail.data?.settings.enabled ?? item.status !== 'disabled';
  const remind = detail.data?.settings.remind ?? false;

  return (
    <li className={`ck-tone-${item.tone} flex items-center gap-3 py-2.5`}>
      <span className="grid size-9 shrink-0 place-items-center rounded-xl bg-(--ck-soft) text-(--ck-ink)">
        <Icon name={checkupIcon(item.icon, { category: item.category })} size={18} />
      </span>
      <span className="min-w-0 flex-1 truncate text-start text-[13.5px] font-bold text-(--ink)">
        {item.title}
      </span>
      <span className="flex flex-col items-center gap-1">
        <span className="text-[10.5px] font-semibold text-(--ink-3)">{t('enabled')}</span>
        <Switch
          on={enabled}
          label={`${t('enabled')} — ${item.title}`}
          disabled={update.isPending}
          onClick={() => update.mutate({ id: item.id, enabled: !enabled })}
        />
      </span>
      <span className="flex flex-col items-center gap-1">
        <span className="text-[10.5px] font-semibold text-(--ink-3)">{t('remind')}</span>
        <Switch
          on={remind}
          label={`${t('remind')} — ${item.title}`}
          disabled={update.isPending || !detail.data || !enabled}
          onClick={() => update.mutate({ id: item.id, remind: !remind })}
        />
      </span>
    </li>
  );
}

/** «تنظیمات برنامه»: every type of the plan with its switches (PUT /checkups/{id}/settings). */
export function PlanSettingsSheet({
  open,
  items,
  onClose,
}: {
  open: boolean;
  items: readonly CheckupItem[];
  onClose: () => void;
}) {
  const t = useTranslations('checkups');
  return (
    <AppSheet open={open} onClose={onClose} size="full" title={t('settings.title')}>
      <p className="mb-2 text-start text-[12.5px] text-(--ink-3)">{t('settings.subtitle')}</p>
      {items.length === 0 ? (
        <p className="rmd-state">{t('list.empty')}</p>
      ) : (
        <ul className="flex flex-col divide-y divide-(--line)">
          {items.map((item) => (
            <SettingsRow key={item.id} item={item} />
          ))}
        </ul>
      )}
    </AppSheet>
  );
}

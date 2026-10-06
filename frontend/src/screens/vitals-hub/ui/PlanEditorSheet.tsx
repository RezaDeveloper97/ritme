'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { VITAL_TYPES, useSavePlan, useVitalsPlan, type PlanItem, type VitalsPlan } from '@/entities/vital';
import { Link } from '@/shared/i18n';
import { AppSheet } from '@/shared/sheet';
import { ChipGroup, EmptyState, Icon, InfoNote, PillChip, PrimaryButton, SecondaryButton, Skeleton, SkeletonGroup } from '@/shared/ui';

import { ALL_DAYS, nextFreeItem, planIssues, toggleDay, withType } from '../model/plan';

/** The weekly measuring plan editor (PUT /vitals/plan), a full sheet over the hub. */
export function PlanEditorSheet({ onClose }: { onClose: () => void }) {
  const t = useTranslations('vitals');
  const plan = useVitalsPlan();
  return (
    <AppSheet open size="full" onClose={onClose} title={t('plan.title')}>
      {plan.isPending ? (
        <SkeletonGroup label={t('common.loading')} className="vt-plan-edit">
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      ) : plan.isError ? (
        <EmptyState
          icon="warning"
          title={t('common.loadError')}
          action={<PrimaryButton onClick={() => void plan.refetch()}>{t('common.retry')}</PrimaryButton>}
        />
      ) : (
        <PlanForm plan={plan.data} onClose={onClose} />
      )}
    </AppSheet>
  );
}

function PlanForm({ plan, onClose }: { plan: VitalsPlan; onClose: () => void }) {
  const t = useTranslations('vitals');
  const save = useSavePlan();
  const [items, setItems] = useState<PlanItem[]>(() => plan.items.map((i) => ({ ...i })));
  const [tried, setTried] = useState(false);
  const issues = planIssues(items, plan.slots);
  const next = nextFreeItem(items, plan.slots);
  const update = (i: number, item: PlanItem) => setItems((list) => list.map((x, j) => (j === i ? item : x)));
  const submit = () => {
    setTried(true);
    if (Object.keys(issues).length || save.isPending) return;
    save.mutate(items, { onSuccess: onClose });
  };
  return (
    <div className="vt-plan-edit">
      <p className="vt-card-note">{t('plan.intro')}</p>
      {!plan.remindersEnabled ? (
        <InfoNote>
          {t('plan.remindersOff')}{' '}
          <Link href="/profile/notifications" className="vt-inline-link">
            {t('plan.openSettings')}
          </Link>
        </InfoNote>
      ) : null}
      {items.length === 0 ? <p className="vt-card-note">{t('plan.empty')}</p> : null}
      <ol className="vt-plan-items">
        {items.map((item, i) => {
          const rowName = t('hub.planRow', { type: t(`short.${item.type}`), slot: t(`slots.${item.slot}` as 'slots.morning') });
          const issue = tried ? issues[i] : undefined;
          return (
            <li key={i} className="nb-card vt-plan-item">
              <div className="vt-plan-item-head">
                <b className="vt-plan-item-title">{rowName}</b>
                <button type="button" className="vt-icon-btn" aria-label={t('plan.remove', { row: rowName })} onClick={() => setItems((list) => list.filter((_, j) => j !== i))}>
                  <Icon name="trash" size={18} />
                </button>
              </div>
              <ChipGroup label={t('plan.type')}>
                {VITAL_TYPES.map((type) => (
                  <PillChip key={type} pressed={item.type === type} onPressedChange={() => update(i, withType(items, i, type, plan.slots))}>
                    {t(`short.${type}`)}
                  </PillChip>
                ))}
              </ChipGroup>
              <ChipGroup label={t('plan.slot')}>
                {(plan.slots[item.type] ?? []).map((slot) => (
                  <PillChip key={slot} pressed={item.slot === slot} onPressedChange={() => update(i, { ...item, slot })}>
                    {t(`slots.${slot}` as 'slots.morning')}
                  </PillChip>
                ))}
              </ChipGroup>
              <div className="vt-plan-weekdays" role="group" aria-label={t('plan.days')}>
                {ALL_DAYS.map((d) => (
                  <button
                    key={d}
                    type="button"
                    className="vt-weekday"
                    aria-pressed={item.days.includes(d)}
                    onClick={() => update(i, { ...item, days: toggleDay(item.days, d) })}
                  >
                    {t(`weekdays.${d}` as 'weekdays.0')}
                  </button>
                ))}
              </div>
              <label className="fld-label vt-remind">
                <span className="fld-label-t">{t('plan.remindAt')}</span>
                <span className="vt-remind-row">
                  <input
                    className="field fld-input vt-clock"
                    type="time"
                    dir="ltr"
                    value={item.remindAt ?? ''}
                    onChange={(e) => update(i, { ...item, remindAt: e.target.value || null })}
                  />
                  {item.remindAt ? (
                    <button type="button" className="vt-link-btn" onClick={() => update(i, { ...item, remindAt: null })}>
                      {t('plan.remindOff')}
                    </button>
                  ) : null}
                </span>
              </label>
              {issue ? (
                <p className="vt-error" role="alert">
                  {t(issue === 'needDay' ? 'plan.needDay' : 'plan.duplicate')}
                </p>
              ) : null}
            </li>
          );
        })}
      </ol>
      {next ? (
        <SecondaryButton icon="plus" onClick={() => setItems((list) => [...list, next])}>
          {t('plan.add')}
        </SecondaryButton>
      ) : (
        <p className="vt-card-note">{t('plan.full')}</p>
      )}
      {save.isError ? (
        <p className="vt-error" role="alert">
          {t('plan.error')}
        </p>
      ) : null}
      <PrimaryButton loading={save.isPending} onClick={submit}>
        {t('plan.save')}
      </PrimaryButton>
    </div>
  );
}

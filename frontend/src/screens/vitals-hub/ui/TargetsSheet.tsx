'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useGlucoseTargets, useSaveTargets, useVitalFormat, type GlucoseTargetInput, type GlucoseTargets, type TargetContext } from '@/entities/vital';
import { AppSheet } from '@/shared/sheet';
import { Card, ChipGroup, EmptyState, Icon, PillChip, PrimaryButton, SecondaryButton, Skeleton, SkeletonGroup } from '@/shared/ui';

import { parseTarget, targetIssues, targetRows } from '../model/targets';

type T = ReturnType<typeof useTranslations<'vitals'>>;
type Fmt = ReturnType<typeof useVitalFormat>;

/** «۸۰–۱۳۰», «زیر ۱۸۰», «تا ۱۴۰», «از ۹۰» of a band (mg/dL; null = open). */
function bandText(t: T, f: Fmt, low: number | null, high: number | null, highInclusive: boolean): string | null {
  if (low !== null && high !== null) return t('targets.range', { low: f.num(low), high: f.num(high) });
  if (high !== null) return t(highInclusive ? 'targets.atMost' : 'targets.below', { high: f.num(high) });
  if (low !== null) return t('targets.from', { low: f.num(low) });
  return null;
}

/**
 * «هدف‌های قندی که پزشکت تعیین کرده» — only for a user with diabetes in her
 * profile (the server classifies with the targets only then). Shows the saved
 * targets and opens the editor.
 */
export function TargetsCard() {
  const t = useTranslations('vitals');
  const targets = useGlucoseTargets();
  const [open, setOpen] = useState(false);
  if (!targets.data?.diabetic) return null;
  const n = targets.data.items.length;
  return (
    <Card as="section" className="vt-card" aria-labelledby="vt-targets-title">
      <div className="vt-card-head">
        <h2 id="vt-targets-title" className="vt-card-title">
          {t('targets.title')}
        </h2>
      </div>
      <p className="vt-card-note">{n ? t('targets.summaryCustom', { n: String(n) }) : t('targets.summaryDefault')}</p>
      <button type="button" className="vt-link-btn" onClick={() => setOpen(true)}>
        <Icon name={n ? 'pencil' : 'plus'} size={16} />
        {t(n ? 'targets.edit' : 'targets.open')}
      </button>
      {open ? <TargetsSheet targets={targets.data} onClose={() => setOpen(false)} /> : null}
    </Card>
  );
}

function TargetsSheet({ targets, onClose }: { targets: GlucoseTargets; onClose: () => void }) {
  const t = useTranslations('vitals');
  const q = useGlucoseTargets();
  return (
    <AppSheet open size="full" onClose={onClose} title={t('targets.title')}>
      {q.isPending ? (
        <SkeletonGroup label={t('common.loading')} className="vt-plan-edit">
          <Skeleton shape="card" />
        </SkeletonGroup>
      ) : q.isError ? (
        <EmptyState icon="warning" title={t('common.loadError')} action={<PrimaryButton onClick={() => void q.refetch()}>{t('common.retry')}</PrimaryButton>} />
      ) : (
        <TargetsForm targets={q.data ?? targets} onClose={onClose} />
      )}
    </AppSheet>
  );
}

export function TargetsForm({ targets, onClose }: { targets: GlucoseTargets; onClose: () => void }) {
  const t = useTranslations('vitals');
  const f = useVitalFormat();
  const save = useSaveTargets();
  const [rows, setRows] = useState<GlucoseTargetInput[]>(() => targetRows(targets));
  const [tried, setTried] = useState(false);
  const issues = targetIssues(rows, targets.bounds);
  const update = (context: TargetContext, patch: Partial<GlucoseTargetInput>) =>
    setRows((list) => list.map((r) => (r.context === context ? { ...r, ...patch } : r)));
  const submit = () => {
    setTried(true);
    if (Object.keys(issues).length || save.isPending) return;
    save.mutate(rows, { onSuccess: onClose });
  };
  return (
    <div className="vt-plan-edit">
      <p className="vt-card-note">{t('targets.intro')}</p>
      <ol className="vt-plan-items">
        {rows.map((row) => {
          const name = t(`targets.contexts.${row.context}`);
          const d = targets.defaults[row.context];
          const def = d ? bandText(t, f, d.lowMgDl, d.highMgDl, d.highInclusive) : null;
          const issue = tried ? issues[row.context] : undefined;
          return (
            <li key={row.context} className="nb-card vt-plan-item">
              <div className="vt-plan-item-head">
                <b className="vt-plan-item-title">{name}</b>
                <span className="vt-card-aside">{def ? t('targets.defaultRange', { range: def }) : t('targets.noDefault')}</span>
              </div>
              <div className="vt-remind-row">
                <label className="fld-label">
                  <span className="fld-label-t">{t('targets.low')}</span>
                  <input
                    className="field fld-input"
                    inputMode="numeric"
                    dir="ltr"
                    aria-label={t('targets.lowAria', { context: name })}
                    value={row.lowMgDl ?? ''}
                    onChange={(e) => update(row.context, { lowMgDl: parseTarget(e.target.value) })}
                  />
                </label>
                <label className="fld-label">
                  <span className="fld-label-t">{t('targets.high')}</span>
                  <input
                    className="field fld-input"
                    inputMode="numeric"
                    dir="ltr"
                    aria-label={t('targets.highAria', { context: name })}
                    value={row.highMgDl ?? ''}
                    onChange={(e) => update(row.context, { highMgDl: parseTarget(e.target.value) })}
                  />
                </label>
                <span className="vt-unit-text" dir="ltr">
                  {t('targets.unit')}
                </span>
              </div>
              <ChipGroup label={t('targets.setBy')}>
                {(['doctor', 'user'] as const).map((by) => (
                  <PillChip key={by} pressed={row.setBy === by} onPressedChange={() => update(row.context, { setBy: by })}>
                    {t(by === 'doctor' ? 'targets.setByDoctor' : 'targets.setByUser')}
                  </PillChip>
                ))}
              </ChipGroup>
              {issue ? (
                <p className="vt-error" role="alert">
                  {issue === 'bounds'
                    ? t('targets.errors.bounds', { min: f.num(targets.bounds.min), max: f.num(targets.bounds.max) })
                    : t('targets.errors.lowHigh')}
                </p>
              ) : null}
            </li>
          );
        })}
      </ol>
      {save.isError ? (
        <p className="vt-error" role="alert">
          {t('targets.error')}
        </p>
      ) : null}
      <PrimaryButton loading={save.isPending} onClick={submit}>
        {t('targets.save')}
      </PrimaryButton>
      {targets.items.length ? (
        <SecondaryButton icon="refresh" disabled={save.isPending} onClick={() => save.mutate(null, { onSuccess: onClose })}>
          {t('targets.reset')}
        </SecondaryButton>
      ) : null}
    </div>
  );
}

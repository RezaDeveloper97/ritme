'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';

import { useCan } from '@/features/auth';
import { fieldError } from '@/shared/api';
import { formatDateTime, formatNumber } from '@/shared/lib';
import { Badge, Button, Panel, Skeleton, Switch, TextInput, toast, useNotifyError } from '@/shared/ui';

import { useAlertRules, useSaveAlertRule, type AlertRule } from '../api/alert-rules';
import type { OverviewMetrics } from '../api/overview';
import { alertRows, fromDisplay, ruleUnit, toDisplay, type AlertRow } from '../lib/alerts';

const RULES = ['ocr_correction_rate', 'message_ctr', 'rui_users', 'engine_mae', 'safety_selftest_failed'] as const;

function useValueText() {
  const t = useTranslations('overview');
  const locale = useLocale();
  return (v: number, unit: 'pct' | 'days' | 'count') => {
    const n = formatNumber(v, locale);
    return unit === 'pct' ? `${n}٪` : unit === 'days' ? t('daysValue', { n }) : n;
  };
}

/** Active alerts (rule, subject, value vs threshold, opened time, link to the source) + the last resolved ones. */
export function AlertsPanel({ m }: { m: OverviewMetrics | null }) {
  const t = useTranslations('overview');
  const locale = useLocale();
  const canManage = useCan('dashboard', 'manage');
  const [editing, setEditing] = useState(false);
  const fmt = useValueText();
  const rows = m ? alertRows(m.alerts, m.resolved_alerts) : null;

  const label = (r: AlertRow) =>
    r.ref
      ? t('alertRuleCtr', { rule: formatNumber(r.ref.rule, locale), version: formatNumber(r.ref.version, locale) })
      : t(`alertRule.${r.key}` as 'alertRule.engine_mae', { default: r.key } as never);

  return (
    <Panel
      title={t('alertsTitle')}
      actions={canManage ? <Button size="sm" variant="ghost" aria-expanded={editing} onClick={() => setEditing((v) => !v)}>{t('alertRulesEdit')}</Button> : undefined}
    >
      {!rows ? (
        <Skeleton className="h-24 w-full" />
      ) : (
        <div className="flex flex-col gap-3">
          {rows.open.length === 0 ? (
            <p className="m-0 text-ink-3">{t('alertsNone')}</p>
          ) : (
            <ul className="m-0 flex list-none flex-col gap-3 p-0 text-sm">
              {rows.open.map((r) => (
                <li key={r.id} className="flex flex-col gap-0.5">
                  <span className="flex flex-wrap items-center gap-2 text-ink">
                    <Badge tone={r.critical ? 'red' : 'amber'}>{r.critical ? t('severity.critical') : t('severity.warning')}</Badge>
                    <span className="font-semibold">{label(r)}</span>
                  </span>
                  <span className="text-ink-3">
                    {t('alertValue', { value: fmt(r.value, r.unit), limit: fmt(r.threshold, r.unit) })}
                  </span>
                  <span className="flex flex-wrap gap-x-3 text-xs text-muted">
                    <span>{t('alertOpened', { at: formatDateTime(r.openedAt, locale) })}</span>
                    <Link href={r.href} className="text-brand-deep">{t('alertSource')}</Link>
                  </span>
                </li>
              ))}
            </ul>
          )}
          {rows.resolved.length > 0 ? (
            <details>
              <summary className="cursor-pointer text-[13px] font-semibold text-ink-3">{t('alertsResolved', { n: formatNumber(rows.resolved.length, locale) })}</summary>
              <ul className="m-0 mt-2 flex list-none flex-col gap-1 p-0 text-xs text-ink-3">
                {rows.resolved.map((r) => (
                  <li key={r.id}>
                    {label(r)} · {fmt(r.value, r.unit)} · {r.resolvedAt ? t('alertResolvedAt', { at: formatDateTime(r.resolvedAt, locale) }) : formatDateTime(r.openedAt, locale)}
                  </li>
                ))}
              </ul>
            </details>
          ) : null}
          {editing && canManage ? <AlertRulesEditor onClose={() => setEditing(false)} /> : null}
        </div>
      )}
    </Panel>
  );
}

/** Threshold and enabled per alert rule; the comparator, window and minimum volume are fixed by the backend. */
function AlertRulesEditor({ onClose }: { onClose: () => void }) {
  const t = useTranslations('overview');
  const q = useAlertRules(true);
  if (q.isPending) return <Skeleton className="h-24 w-full" />;
  if (q.error || !q.data) return <p role="alert" className="m-0 text-danger-deep">{t('alertRulesError')}</p>;
  const rules = [...q.data].sort((a, b) => RULES.indexOf(a.key as (typeof RULES)[number]) - RULES.indexOf(b.key as (typeof RULES)[number]));
  return (
    <section aria-label={t('alertRulesTitle')} className="flex flex-col gap-3 rounded border border-line p-3">
      <h3 className="m-0 text-sm font-bold">{t('alertRulesTitle')}</h3>
      <p className="m-0 text-xs text-muted">{t('alertRulesNote')}</p>
      {rules.map((r) => (
        <RuleRow key={r.key} rule={r} />
      ))}
      <div><Button size="sm" variant="ghost" onClick={onClose}>{t('close')}</Button></div>
    </section>
  );
}

function RuleRow({ rule }: { rule: AlertRule }) {
  const t = useTranslations('overview');
  const save = useSaveAlertRule();
  const notifyError = useNotifyError();
  const unit = ruleUnit(rule.key);
  const [threshold, setThreshold] = useState(String(toDisplay(rule.key, rule.threshold)));
  const [enabled, setEnabled] = useState(rule.enabled);
  const parsed = Number(threshold);
  const valid = threshold.trim() !== '' && Number.isFinite(parsed) && parsed >= 0;
  const dirty = enabled !== rule.enabled || (valid && fromDisplay(rule.key, parsed) !== rule.threshold);
  return (
    <form
      className="grid items-end gap-3 sm:grid-cols-[1fr_8rem_auto]"
      onSubmit={(e) => {
        e.preventDefault();
        if (!valid) return;
        save.mutate({ key: rule.key, threshold: fromDisplay(rule.key, parsed), enabled }, { onSuccess: () => toast.success(t('alertRulesSaved')), onError: notifyError });
      }}
    >
      <Switch label={t(`alertRule.${rule.key}` as 'alertRule.engine_mae', { default: rule.key } as never)} hint={t(`alertRuleHint.${rule.key}` as 'alertRuleHint.engine_mae', { default: '' } as never)} checked={enabled} onChange={setEnabled} />
      <TextInput
        label={`${t('alertThreshold')} (${unit === 'pct' ? '٪' : unit === 'days' ? t('unitDays') : t('unitCount')})`}
        type="number"
        step="any"
        min={0}
        dir="ltr"
        value={threshold}
        onChange={(e) => setThreshold(e.target.value)}
        error={fieldError(save.error, 'threshold')}
      />
      <Button type="submit" size="sm" variant="primary" loading={save.isPending} disabled={!dirty || !valid}>{t('save')}</Button>
    </form>
  );
}

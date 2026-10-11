'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { formatNumber } from '@/shared/lib';
import { Badge, Button, ErrorState, PageHeader, Panel, Select, Skeleton, TextInput, useNotifyError, type BadgeTone } from '@/shared/ui';

import { useEngineView, useSimulate } from '../api/rules';

const KIND_TONE: Record<string, BadgeTone> = { safety: 'red', engine: 'data', clinical: 'amber', product: 'neutral' };
const MODES = ['cycle', 'pregnancy', 'postpartum'] as const;
const NUMBER_FACTS = ['cycle_day', 'cycle_length', 'days_to_period', 'pregnancy_week'] as const;
const PIPELINE = ['event', 'state', 'candidates', 'evaluate', 'caps', 'select', 'ab'] as const;

/** /messages/rules/engine — pipeline, live rules per placement and the simulator (Admin_Engine artboard). */
export function EngineScreen() {
  const t = useTranslations('messageEngine');
  const tr = useTranslations('messageRules');
  const locale = useLocale();
  const num = (n: number) => formatNumber(n, locale);
  const notifyError = useNotifyError();
  const view = useEngineView();
  const sim = useSimulate();

  const [mode, setMode] = useState<string>('cycle');
  const [nums, setNums] = useState<Record<string, string>>({});
  const [phase, setPhase] = useState('');
  const [name, setName] = useState('');
  const [premium, setPremium] = useState('false');
  const [at, setAt] = useState('');

  const run = () => {
    const facts: Record<string, unknown> = { mode, is_premium: premium === 'true' };
    for (const k of NUMBER_FACTS) {
      const v = nums[k]?.trim();
      if (v && !Number.isNaN(Number(v))) facts[k] = Number(v);
    }
    if (phase.trim()) facts.phase = phase.trim();
    if (name.trim()) facts.name = name.trim();
    sim.mutate({ facts, at: at ? new Date(at).toISOString() : undefined }, { onError: notifyError });
  };

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('title')} backHref="/messages/rules" backLabel={tr('backToList')} meta={t('subtitle')} />

      <Panel title={t('pipelineTitle')}>
        <ol className="m-0 grid list-none gap-3 p-0 sm:grid-cols-2 lg:grid-cols-4">
          {PIPELINE.map((s, i) => (
            <li key={s} className="flex flex-col gap-1">
              <strong>
                {num(i + 1)}. {t(`pipeline.${s}`)}
              </strong>
              <span className="text-[13px] text-ink-3">{t(`pipelineHints.${s}`)}</span>
            </li>
          ))}
        </ol>
      </Panel>

      <Panel title={t('simTitle')}>
        <div className="flex flex-col gap-3">
          <p className="m-0 text-[13px] text-ink-3">{t('simHint')}</p>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <Select
              label={t('fact.mode')}
              value={mode}
              onChange={(e) => setMode(e.target.value)}
              options={MODES.map((m) => ({ value: m, label: t(`modes.${m}`) }))}
            />
            {NUMBER_FACTS.map((k) => (
              <TextInput
                key={k}
                label={t(`fact.${k}`)}
                inputMode="numeric"
                dir="ltr"
                value={nums[k] ?? ''}
                onChange={(e) => setNums((p) => ({ ...p, [k]: e.target.value }))}
              />
            ))}
            <TextInput label={t('fact.phase')} dir="ltr" value={phase} onChange={(e) => setPhase(e.target.value)} placeholder="luteal" />
            <TextInput label={t('fact.name')} value={name} onChange={(e) => setName(e.target.value)} />
            <Select
              label={t('fact.is_premium')}
              value={premium}
              onChange={(e) => setPremium(e.target.value)}
              options={[
                { value: 'false', label: t('no') },
                { value: 'true', label: t('yes') },
              ]}
            />
            <TextInput label={t('at')} type="datetime-local" dir="ltr" value={at} onChange={(e) => setAt(e.target.value)} />
          </div>
          <div>
            <Button variant="primary" loading={sim.isPending} onClick={run}>
              {t('run')}
            </Button>
          </div>
          {sim.data ? (
            <div className="flex flex-col gap-3" aria-live="polite">
              {sim.data.placements.map((p) => (
                <div key={p.placement} className="rounded-lg border border-line p-3">
                  <div className="flex flex-wrap items-center gap-2">
                    <strong>{tr(`placements.${p.placement}` as 'placements.smart_tip')}</strong>
                    {p.winner ? (
                      <Badge tone="green">{t('winner', { title: p.winner.title })}</Badge>
                    ) : (
                      <Badge>{t('noWinner')}</Badge>
                    )}
                    {p.winner ? <span className="text-[13px] text-ink-3">{p.winner.message_title}</span> : null}
                  </div>
                  {p.trace.length > 0 ? (
                    <ol className="m-0 mt-2 flex list-none flex-col gap-1 p-0 text-[13px]">
                      {p.trace.map((r) => (
                        <li key={r.rule_id} className="flex flex-wrap items-center gap-2">
                          <span className="num text-ink-3">{num(r.priority)}</span>
                          <Link href={`/messages/rules/${r.rule_id}`}>{r.title}</Link>
                          <Badge tone={KIND_TONE[r.kind]}>{tr(`kinds.${r.kind}` as 'kinds.safety')}</Badge>
                          {r.winner ? (
                            <Badge tone="green">{t('shown')}</Badge>
                          ) : r.eligible ? (
                            <span className="text-ink-3">{t('lostToHigher')}</span>
                          ) : (
                            <span className="text-ink-3">{tr(`reasons.${r.reason}` as 'reasons.conditions')}</span>
                          )}
                        </li>
                      ))}
                    </ol>
                  ) : (
                    <p className="m-0 mt-2 text-[13px] text-ink-3">{t('noRules')}</p>
                  )}
                </div>
              ))}
            </div>
          ) : null}
        </div>
      </Panel>

      <Panel title={t('liveTitle')}>
        {view.isPending ? (
          <Skeleton className="h-32" />
        ) : view.error ? (
          <ErrorState error={view.error} onRetry={() => view.refetch()} />
        ) : (
          <div className="flex flex-col gap-4">
            {view.data.placements.map((p) => (
              <section key={p.placement} className="flex flex-col gap-2">
                <h3 className="m-0 flex flex-wrap items-center gap-2 text-[15px]">
                  {tr(`placements.${p.placement}` as 'placements.smart_tip')}
                  <Badge>{t('count', { n: num(p.count) })}</Badge>
                  {p.urgent ? <Badge tone="red">{t('urgent')}</Badge> : null}
                </h3>
                {p.rules.length === 0 ? (
                  <p className="m-0 text-[13px] text-ink-3">{t('noRules')}</p>
                ) : (
                  <ul className="m-0 flex list-none flex-col gap-1 p-0 text-[13px]">
                    {p.rules.map((r) => (
                      <li key={r.rule_id} className="flex flex-wrap items-center gap-2">
                        <span className="num text-ink-3">{num(r.priority)}</span>
                        <Link href={`/messages/rules/${r.rule_id}`}>{r.title}</Link>
                        <Badge tone={KIND_TONE[r.kind]}>{tr(`kinds.${r.kind}` as 'kinds.safety')}</Badge>
                        {r.variants ? <Badge tone="data">A/B</Badge> : null}
                        <span className="text-ink-3">
                          {r.unlimited
                            ? t('noCap')
                            : [
                                r.cap ? t('cap', { n: num(r.cap.count), p: r.cap.period }) : null,
                                r.cooldown_days > 0 ? t('cooldown', { n: num(r.cooldown_days) }) : null,
                              ]
                                .filter(Boolean)
                                .join(' · ') || t('noCap')}
                        </span>
                        {r.conflicts > 0 ? <Badge tone="amber">{t('conflicts', { n: num(r.conflicts) })}</Badge> : null}
                      </li>
                    ))}
                  </ul>
                )}
              </section>
            ))}
          </div>
        )}
      </Panel>
    </div>
  );
}

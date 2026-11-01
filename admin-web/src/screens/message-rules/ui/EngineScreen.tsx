'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';

import { formatNumber } from '@/shared/lib';
import { Badge, ErrorState, PageHeader, Panel, Skeleton, type BadgeTone } from '@/shared/ui';

import { useEngineView } from '../api/rules';
import { SimulatorPanel } from './SimulatorPanel';

const KIND_TONE: Record<string, BadgeTone> = { safety: 'red', engine: 'data', clinical: 'amber', product: 'neutral' };
const PIPELINE = ['event', 'state', 'candidates', 'evaluate', 'caps', 'select', 'ab'] as const;

/** /messages/rules/engine — pipeline, live rules per placement and the simulator (Admin_Engine artboard). */
export function EngineScreen() {
  const t = useTranslations('messageEngine');
  const tr = useTranslations('messageRules');
  const locale = useLocale();
  const num = (n: number) => formatNumber(n, locale);
  const view = useEngineView();

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('title')} backHref="/messages/rules" backLabel={tr('backToList')} meta={t('subtitle')} />

      <Panel title={t('pipelineTitle')}>
        <ol className="m-0 flex list-none flex-wrap items-stretch gap-2 p-0">
          {PIPELINE.map((s, i) => (
            <li key={s} className="flex items-stretch gap-2">
              <div className="tone-card min-w-40 flex-1" data-tone="neutral">
                <strong className="flex items-center gap-2">
                  <span className="step-chip">{num(i + 1)}</span>
                  {t(`pipeline.${s}`)}
                </strong>
                <span className="text-[13px] text-ink-3">{t(`pipelineHints.${s}`)}</span>
              </div>
              {i < PIPELINE.length - 1 ? (
                <span className="step-arrow" aria-hidden="true">
                  ←
                </span>
              ) : null}
            </li>
          ))}
        </ol>
      </Panel>

      <SimulatorPanel />

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
      <Panel title={t('fixedTitle')}>
        <ul className="m-0 grid list-none gap-2 p-0 text-[13px] text-ink-3 sm:grid-cols-2">
          {(['safety', 'ab', 'events', 'versions'] as const).map((k) => (
            <li key={k} className="flex items-start gap-2">
              <span className="text-data-deep" aria-hidden="true">
                ✓
              </span>
              {t(`fixed.${k}`)}
            </li>
          ))}
        </ul>
      </Panel>
    </div>
  );
}

'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { isApiError } from '@/shared/api';
import { formatNumber } from '@/shared/lib';
import { Badge, Button, Panel, TextArea, TextInput, toast, useNotifyError, type BadgeTone } from '@/shared/ui';

import { useRequestHealthAccess, useResolveUser, useSimulateV6, type SimulateInput } from '../api/rules';
import { engineSummary, parseUserRef, toRows } from '../lib/simulator';

const KIND_TONE: Record<string, BadgeTone> = { safety: 'red', engine: 'data', clinical: 'amber', product: 'neutral' };

/**
 * «Why didn't user X see message Y today» (T-M10-11, SPEC §5.4.1): a user (id or RT code) or synthetic facts, a date
 * and a time. A real user needs a live health-access grant; without one the API answers `health_access_required`
 * and the reason + ticket prompt of the access screen is shown here.
 */
export function SimulatorPanel() {
  const t = useTranslations('messageSim');
  const tr = useTranslations('messageRules');
  const locale = useLocale();
  const num = (n: number) => formatNumber(n, locale);
  const notifyError = useNotifyError();
  const sim = useSimulateV6();
  const resolve = useResolveUser();

  const [userRef, setUserRef] = useState('');
  const [date, setDate] = useState('');
  const [time, setTime] = useState('');
  const [facts, setFacts] = useState('');
  const [needGrantFor, setNeedGrantFor] = useState<number | null>(null);
  const [fieldErr, setFieldErr] = useState<string | null>(null);

  const run = async () => {
    setFieldErr(null);
    setNeedGrantFor(null);
    const body: SimulateInput = {};
    const ref = parseUserRef(userRef);
    if (ref.kind === 'invalid') return setFieldErr(t('userInvalid'));
    if (ref.kind === 'id') body.user_id = ref.id;
    if (ref.kind === 'code') {
      try {
        const id = await resolve.mutateAsync(ref.code);
        if (id === null) return setFieldErr(t('userNotFound'));
        body.user_id = id;
      } catch (e) {
        return notifyError(e);
      }
    }
    if (date) body.date = date;
    if (time) body.time = time;
    if (facts.trim()) {
      try {
        body.facts_override = JSON.parse(facts) as Record<string, unknown>;
      } catch {
        return setFieldErr(t('factsInvalid'));
      }
    }
    sim.mutate(body, {
      onError: (e) => {
        if (isApiError(e) && e.code === 'health_access_required' && body.user_id) setNeedGrantFor(body.user_id);
        else notifyError(e);
      },
    });
  };

  const rows = sim.data ? toRows(sim.data) : [];

  return (
    <Panel title={t('title')}>
      <div className="flex flex-col gap-3">
        <p className="m-0 text-[13px] text-ink-3">{t('hint')}</p>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <TextInput label={t('user')} hint={t('userHint')} dir="ltr" value={userRef} onChange={(e) => setUserRef(e.target.value)} placeholder="RT-ABC123" />
          <TextInput label={t('date')} type="date" dir="ltr" value={date} onChange={(e) => setDate(e.target.value)} />
          <TextInput label={t('time')} type="time" dir="ltr" value={time} onChange={(e) => setTime(e.target.value)} />
          <TextInput
            label={t('facts')}
            hint={t('factsHint')}
            dir="ltr"
            value={facts}
            onChange={(e) => setFacts(e.target.value)}
            placeholder='{"mode":"cycle","cycle_day":12}'
          />
        </div>
        {fieldErr ? (
          <span className="field-error" role="alert">
            {fieldErr}
          </span>
        ) : null}
        <div>
          <Button variant="primary" loading={sim.isPending || resolve.isPending} onClick={run}>
            {t('run')}
          </Button>
        </div>

        {needGrantFor !== null ? <GrantPrompt userId={needGrantFor} /> : null}

        {sim.data ? (
          <div className="flex flex-col gap-3" aria-live="polite">
            <div className="flex flex-wrap items-center gap-2 text-[13px]">
              <Badge tone={sim.data.source === 'user' ? 'amber' : 'data'}>{t(`source.${sim.data.source}`)}</Badge>
              <span dir="ltr" className="text-ink-3">
                {sim.data.date} {sim.data.time} · {sim.data.zone}
              </span>
              {sim.data.facts_overridden.length > 0 ? (
                <span className="text-ink-3">{t('overridden', { list: sim.data.facts_overridden.join(', ') })}</span>
              ) : null}
            </div>
            <dl className="m-0 grid grid-cols-2 gap-2 sm:grid-cols-4" aria-label={t('engineTitle')}>
              {engineSummary(sim.data.engine).map((e) => (
                <div key={e.key} className="rounded-lg border border-line p-2">
                  <dt className="field-label m-0">{e.key}</dt>
                  <dd className="m-0 text-[14px] font-medium" dir="ltr">
                    {e.value}
                  </dd>
                </div>
              ))}
            </dl>
            {rows.map((s) => (
              <section key={s.slot} className="rounded-lg border border-line p-3" aria-label={tr(`placements.${s.slot}` as 'placements.smart_tip')}>
                <div className="flex flex-wrap items-center gap-2">
                  <strong>{tr(`placements.${s.slot}` as 'placements.smart_tip')}</strong>
                  <span className="text-[12px] text-ink-3">{s.capacity === null ? t('capacityAll') : t('capacity', { n: num(s.capacity) })}</span>
                  {s.winners.length === 0 ? <Badge>{t('noWinner')}</Badge> : null}
                </div>
                {s.winners.map((w) => (
                  <div key={`${w.ruleId}-${w.variant}`} className="mt-2 flex flex-col gap-1 rounded-md bg-surface-2 p-2">
                    <span className="flex flex-wrap items-center gap-2 text-[13px]">
                      <Badge tone="green">{t('winner')}</Badge>
                      <Link href={`/messages/rules/${w.ruleId}`}>{w.title}</Link>
                      <Badge tone="data">{t('variantLabel', { v: w.variant.toUpperCase() })}</Badge>
                      {w.version !== null ? <span className="text-ink-3">{t('versionLabel', { n: num(w.version) })}</span> : null}
                    </span>
                    <strong>{w.messageTitle}</strong>
                    <span className="text-[13px]">{w.messageBody}</span>
                  </div>
                ))}
                {s.dropped.length > 0 ? (
                  <ul className="m-0 mt-2 flex list-none flex-col gap-1 p-0 text-[13px]">
                    {s.dropped.map((d) => (
                      <li key={d.ruleId} className="flex flex-wrap items-center gap-2" data-reason={d.reason}>
                        <Link href={`/messages/rules/${d.ruleId}`}>{d.title}</Link>
                        <Badge tone={KIND_TONE[d.kind]}>{tr(`kinds.${d.kind}` as 'kinds.safety')}</Badge>
                        <span className="text-ink-3">{t(`reasons.${d.reasonKey}` as 'reasons.unknown')}</span>
                      </li>
                    ))}
                  </ul>
                ) : null}
              </section>
            ))}
            {sim.data.notes.length > 0 ? (
              <ul className="m-0 list-disc ps-5 text-[12px] text-ink-3">
                {sim.data.notes.map((n) => (
                  <li key={n}>{n}</li>
                ))}
              </ul>
            ) : null}
          </div>
        ) : null}
      </div>
    </Panel>
  );
}

/** 403 health_access_required: nothing about the user was read; ask for the same reason + ticket grant as /health-access. */
function GrantPrompt({ userId }: { userId: number }) {
  const t = useTranslations('messageSim');
  const notifyError = useNotifyError();
  const request = useRequestHealthAccess();
  const [reason, setReason] = useState('');
  const [ticket, setTicket] = useState('');
  return (
    <div className="flex flex-col gap-3 rounded-lg border border-line p-3" role="alert" data-testid="grant-prompt">
      <Badge tone="amber">{t('grantTitle')}</Badge>
      <p className="m-0 text-[13px] text-ink-3">{t('grantHint', { id: userId })}</p>
      <div className="grid gap-3 sm:grid-cols-2">
        <TextInput label={t('ticket')} dir="ltr" maxLength={64} value={ticket} onChange={(e) => setTicket(e.target.value)} />
        <TextArea className="sm:col-span-2" label={t('reason')} rows={2} minLength={10} maxLength={500} value={reason} onChange={(e) => setReason(e.target.value)} />
      </div>
      <div>
        <Button
          loading={request.isPending}
          disabled={reason.trim().length < 10 || ticket.trim() === ''}
          onClick={() =>
            request.mutate({ user_id: userId, reason, ticket }, { onSuccess: () => toast.success(t('grantSent')), onError: notifyError })
          }
        >
          {t('grantSend')}
        </Button>
      </div>
    </div>
  );
}

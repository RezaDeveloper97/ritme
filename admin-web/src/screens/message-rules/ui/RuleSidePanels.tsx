'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useCan } from '@/features/auth';
import { formatNumber } from '@/shared/lib';
import { Badge, Button, Panel, TextInput, toast, useNotifyError, confirm } from '@/shared/ui';

import {
  useConflictCheck,
  usePreview,
  useRuleAction,
  type Conflict,
  type Preview,
  type Rule,
  type Stat,
} from '../api/rules';

/** Medical approval (Admin_MessageEditor «تأیید»): submit, approve / reject (clinical capability), publish. */
export function ApprovalPanel({ rule }: { rule: Rule }) {
  const t = useTranslations('messageRules');
  const notifyError = useNotifyError();
  const act = useRuleAction();
  const canApprove = useCan('clinical', 'manage');
  const run = (action: 'submit' | 'approve' | 'reject' | 'toggle', done: string) =>
    act.mutate({ id: rule.id, action }, { onSuccess: () => toast.success(done), onError: notifyError });

  return (
    <Panel title={t('approvalTitle')}>
      <div className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone={rule.status === 'approved' ? 'green' : rule.status === 'pending_approval' ? 'amber' : 'neutral'}>
            {t(`statuses.${rule.status}`)}
          </Badge>
          {rule.is_active ? <Badge tone="green">{t('active')}</Badge> : <Badge>{t('inactive')}</Badge>}
          <span className="text-[13px] text-ink-3">{t('version', { n: rule.version })}</span>
        </div>
        <p className="m-0 text-[13px] text-ink-3">{rule.needs_medical_approval ? t('approvalHint') : t('approvalNotNeeded')}</p>
        <div className="flex flex-wrap gap-2">
          {rule.needs_medical_approval && rule.status === 'draft' ? (
            <Button loading={act.isPending} onClick={() => run('submit', t('submitted'))}>
              {t('submit')}
            </Button>
          ) : null}
          {rule.status === 'pending_approval' && canApprove ? (
            <>
              <Button variant="primary" loading={act.isPending} onClick={() => run('approve', t('approved'))}>
                {t('approve')}
              </Button>
              <Button loading={act.isPending} onClick={() => run('reject', t('rejected'))}>
                {t('reject')}
              </Button>
            </>
          ) : null}
          {rule.status === 'pending_approval' && !canApprove ? (
            <span className="text-[13px] text-ink-3">{t('waitingForAdvisor')}</span>
          ) : null}
          <Button
            loading={act.isPending}
            disabled={!rule.is_active && rule.needs_medical_approval && rule.status !== 'approved'}
            onClick={async () => {
              if (rule.is_active && !(await confirm({ message: t('confirmUnpublish') }))) return;
              run('toggle', t('statusChanged'));
            }}
          >
            {rule.is_active ? t('unpublish') : t('publish')}
          </Button>
        </div>
      </div>
    </Panel>
  );
}

/** Preview for a user id (the real engine: conditions, window, audience, caps) — fact values are never shown. */
export function PreviewPanel({ payload, ruleId }: { payload: () => unknown; ruleId?: number }) {
  const t = useTranslations('messageRules');
  const notifyError = useNotifyError();
  const preview = usePreview();
  const [userId, setUserId] = useState('');
  const [facts, setFacts] = useState('');
  const result: Preview | undefined = preview.data;

  const run = () => {
    let parsed: Record<string, unknown> | undefined;
    if (facts.trim()) {
      try {
        parsed = JSON.parse(facts) as Record<string, unknown>;
      } catch {
        toast.error(t('factsInvalid'));
        return;
      }
    }
    preview.mutate(
      { rule: payload(), rule_id: ruleId, user_id: userId.trim() ? Number(userId) : undefined, facts: parsed },
      { onError: notifyError },
    );
  };

  return (
    <Panel title={t('previewTitle')}>
      <div className="flex flex-col gap-3">
        <TextInput label={t('previewUser')} value={userId} onChange={(e) => setUserId(e.target.value)} hint={t('previewUserHint')} />
        <TextInput label={t('previewFacts')} dir="ltr" value={facts} onChange={(e) => setFacts(e.target.value)} placeholder='{"mode":"cycle"}' />
        <div>
          <Button loading={preview.isPending} onClick={run}>
            {t('previewRun')}
          </Button>
        </div>
        {result ? (
          <div className="flex flex-col gap-2 rounded-lg border border-line p-3" aria-live="polite">
            <Badge tone={result.eligible ? 'green' : 'amber'}>{result.eligible ? t('eligible') : t('notEligible')}</Badge>
            {result.eligible ? (
              <>
                <strong>{result.title}</strong>
                <span>{result.body}</span>
                <span className="text-[13px] text-ink-3">{t('variantLabel', { v: result.variant.toUpperCase() })}</span>
              </>
            ) : (
              <ul className="m-0 list-disc ps-5 text-[13px] text-ink-3">
                {result.reasons.map((r) => (
                  <li key={r}>{t(`reasons.${r}` as 'reasons.conditions')}</li>
                ))}
              </ul>
            )}
          </div>
        ) : null}
      </div>
    </Panel>
  );
}

/** Conflict check against the other published messages of the same placement. */
export function ConflictPanel({ payload, ruleId, initial }: { payload: () => unknown; ruleId?: number; initial?: Conflict[] }) {
  const t = useTranslations('messageRules');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const check = useConflictCheck();
  const list = check.data?.conflicts ?? initial ?? [];

  return (
    <Panel
      title={t('conflictsTitle')}
      actions={
        <Button size="sm" loading={check.isPending} onClick={() => check.mutate({ rule: payload(), rule_id: ruleId }, { onError: notifyError })}>
          {t('conflictsRun')}
        </Button>
      }
    >
      {list.length === 0 ? (
        <p className="m-0 text-[13px] text-ink-3">{t('noConflicts')}</p>
      ) : (
        <ul className="m-0 flex list-none flex-col gap-2 p-0">
          {list.map((c) => (
            <li key={c.rule_id} className="flex flex-col gap-1 text-[13px]">
              <span>
                <strong>{c.title}</strong> · {t('priority')} {formatNumber(c.priority, locale)}
                {c.safety ? ` · ${t('kinds.safety')}` : ''}
              </span>
              <span className="text-ink-3">{t(`winner.${c.winner}`)}</span>
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}

/** A/B numbers of the last 7 days. */
export function VariantStats({ a, b }: { a: Stat; b: Stat }) {
  const t = useTranslations('messageRules');
  const locale = useLocale();
  const n = (v: number) => formatNumber(v, locale);
  const row = (label: string, s: Stat) => (
    <tr>
      <th scope="row">{label}</th>
      <td className="num">{n(s.impressions)}</td>
      <td className="num">{n(s.clicks)}</td>
      <td className="num">{s.impressions > 0 ? `${n(Math.round(s.ctr * 100))}٪` : '—'}</td>
    </tr>
  );
  return (
    <Panel title={t('variantStats')} bodyClassName="">
      <table className="data-table">
        <thead>
          <tr>
            <th scope="col">{t('variant')}</th>
            <th scope="col">{t('impressions7d')}</th>
            <th scope="col">{t('clicks')}</th>
            <th scope="col">{t('ctr')}</th>
          </tr>
        </thead>
        <tbody>
          {row('A', a)}
          {row('B', b)}
        </tbody>
      </table>
    </Panel>
  );
}

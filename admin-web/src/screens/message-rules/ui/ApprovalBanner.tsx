'use client';

import { useTranslations } from 'next-intl';

import { useCan } from '@/features/auth';
import { Button, toast, useNotifyError } from '@/shared/ui';

import { useRuleAction, type Rule } from '../api/rules';
import { contentDiff, type TextContent } from '../lib/diff';

/** «live: approved v{n} · draft v{m} awaiting approval» with a text diff; approve / reject for the clinical capability. */
export function ApprovalBanner({ rule, approved }: { rule: Rule; approved?: { content?: TextContent } | null }) {
  const t = useTranslations('messageRules');
  const notifyError = useNotifyError();
  const act = useRuleAction();
  const canApprove = useCan('clinical', 'manage');
  if (!rule.needs_medical_approval) return null;
  const pending = rule.status === 'pending_approval';
  const live = rule.approved_version;
  const diffs = pending && approved?.content ? contentDiff(approved.content, rule.content) : [];
  const run = (action: 'approve' | 'reject', done: string) =>
    act.mutate({ id: rule.id, action }, { onSuccess: () => toast.success(done), onError: notifyError });

  return (
    <section
      role="status"
      aria-label={t('bannerLabel')}
      className="flex flex-col gap-3 rounded-lg border border-line bg-surface-2 p-3 text-[13px]"
    >
      <p className="m-0 font-semibold">
        {live !== null ? t('bannerLive', { n: live }) : t('bannerNotLive')}
        {pending ? ` · ${t('bannerDraft', { m: rule.version })}` : rule.status === 'draft' && live !== null && rule.version !== live ? ` · ${t('bannerDraftNotSent', { m: rule.version })}` : ''}
      </p>
      {pending ? (
        diffs.length > 0 ? (
          <div className="flex flex-col gap-2" aria-label={t('diffTitle')}>
            {diffs.map((d) => (
              <p key={d.label} className="m-0 leading-6">
                <span className="text-ink-3" dir="ltr">{d.label}: </span>
                {d.parts.map((p, i) =>
                  p.type === 'same' ? (
                    <span key={i}>{p.text} </span>
                  ) : p.type === 'add' ? (
                    <ins key={i} className="rounded bg-green-500/15 px-0.5 no-underline">{p.text} </ins>
                  ) : (
                    <del key={i} className="rounded bg-red-500/15 px-0.5">{p.text} </del>
                  ),
                )}
              </p>
            ))}
          </div>
        ) : (
          <p className="m-0 text-ink-3">{approved?.content ? t('diffNone') : t('diffUnavailable')}</p>
        )
      ) : null}
      {pending && canApprove ? (
        <div className="flex gap-2">
          <Button variant="primary" loading={act.isPending} onClick={() => run('approve', t('approved'))}>
            {t('approve')}
          </Button>
          <Button loading={act.isPending} onClick={() => run('reject', t('rejected'))}>
            {t('reject')}
          </Button>
        </div>
      ) : null}
      {pending && !canApprove ? <p className="m-0 text-ink-3">{t('waitingForAdvisor')}</p> : null}
    </section>
  );
}

'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  BodySkeleton,
  Disclaimer,
  LoadError,
  ConversationRow,
  deptIcon,
  scalarOf,
  stashDraft,
  useAssistantError,
  useHub,
  useStartConversation,
} from '@/entities/assistant';
import { CreditMeter, useAiCredits } from '@/entities/ai-credit';
import { Link, useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, EmptyState, Icon, IconCircle, ScreenHeader, SectionTitle, SkyLayer, UrgentCard } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

/**
 * `/assistant` — «کلینیک دستیار ریتمی» (nbl_/nbd_v18_AssistantHub): question box, what the assistant knows about her,
 * the departments, recent conversations, free-quota note (links to Plus) and the 115 line.
 */
export function AssistantHubPage() {
  const t = useTranslations('assistant');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const errText = useAssistantError();
  const hub = useHub();
  const credits = useAiCredits();
  const start = useStartConversation();
  const [text, setText] = useState('');

  const ask = () => {
    const body = text.trim();
    const first = hub.data?.departments[0];
    if (!body || !first || start.isPending) return;
    start.mutate(first.code, {
      onSuccess: (id) => {
        stashDraft(id, body);
        router.push(`/assistant/chat/${id}`);
      },
    });
  };

  let body;
  if (hub.isPending) body = <BodySkeleton rows={4} />;
  else if (hub.isError) body = <div className="as-body"><LoadError onRetry={() => void hub.refetch()} busy={hub.isFetching} /></div>;
  else {
    const h = hub.data;
    const shared = h.profile.items.filter((i) => i.shared);
    body = (
      <div className="as-body">
        <div>
          <h2 className="as-greet">{t('hub.greeting')}</h2>
          <form className="as-ask" onSubmit={(e) => { e.preventDefault(); ask(); }}>
            <input className="as-ask-input" value={text} maxLength={2000} onChange={(e) => setText(e.target.value)} placeholder={t('hub.placeholder')} aria-label={t('hub.placeholder')} />
            <button type="submit" className="as-send" aria-label={t('chat.send')} disabled={!text.trim() || start.isPending}><Icon name="send" size={20} className="as-flip" /></button>
          </form>
          {start.isError ? <p className="as-error" role="alert">{errText(start.error)}</p> : null}
        </div>

        {credits.data ? (
          credits.data.canUse ? (
            <Link href="/profile/credits" className="as-quota is-credits" aria-label={t('hub.credits')}>
              <CreditMeter credits={credits.data} />
            </Link>
          ) : (
            <Link href="/plus" className="as-quota is-spent">
              <Icon name="crown" size={18} />
              <span>{t('hub.noSubscription')}</span>
              <span className="as-link">{t('hub.plus')}</span>
            </Link>
          )
        ) : null}

        <Card className="as-know">
          <div className="as-know-head">
            <strong>{t('hub.knows')}</strong>
            <Link href="/assistant/profile" className="as-link">{t('hub.edit')}</Link>
          </div>
          {h.profile.consent.granted && shared.length > 0 ? (
            <div className="as-tags">
              {shared.map((i) => (
                <span key={i.key} className="as-tag">{[t(`profile.keys.${i.key}` as 'profile.keys.age'), scalarOf(i)].filter((v) => v !== null && v !== '').join(' ')}</span>
              ))}
            </div>
          ) : (
            <p className="as-text">{t('hub.knowsNone')}</p>
          )}
        </Card>

        <section aria-labelledby="as-depts">
          <SectionTitle id="as-depts" title={t('hub.departments')} />
          <div className="as-depts">
            {h.departments.map((d) => (
              <Link key={d.code} href={`/assistant/${d.code}`} className="as-dept">
                <IconCircle icon={deptIcon(d.icon)} tone="brand" size="md" />
                <strong>{d.name}</strong>
                <span>{d.tagline}</span>
              </Link>
            ))}
          </div>
        </section>

        <section aria-labelledby="as-recent">
          <SectionTitle id="as-recent" title={t('hub.recent')} actionLabel={h.recent.length ? t('hub.all') : undefined} onAction={() => router.push('/assistant/history')} />
          {h.recent.length === 0 ? (
            <Card><EmptyState icon="chat" title={t('hub.emptyTitle')} body={t('hub.emptyBody')} /></Card>
          ) : (
            <div className="as-list">{h.recent.map((c) => <ConversationRow key={c.id} c={c} />)}</div>
          )}
        </section>

        <UrgentCard variant="note" title={t('hub.urgent')} hotlines={[{ label: t('emergency.line'), number: h.emergency.phone, display: formatNumber(h.emergency.phone, locale) }]} hotlinesLabel={t('emergency.line')} />
        <Disclaimer />
      </div>
    );
  }

  return (
    <div className="view as-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('hub.title')} subtitle={t('hub.subtitle')} onBack={() => router.push('/services')} backLabel={t('common.back')} />
        {body}
      </div>
      <BottomNav />
    </div>
  );
}

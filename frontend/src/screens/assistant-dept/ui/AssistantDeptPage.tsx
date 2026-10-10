'use client';

import { useTranslations } from 'next-intl';

import {
  BodySkeleton,
  Disclaimer,
  LoadError,
  deptIcon,
  scalarOf,
  stashDraft,
  useAssistantError,
  useDepartment,
  useStartConversation,
} from '@/entities/assistant';
import { Link, useRouter } from '@/shared/i18n';
import { Card, EmptyState, Icon, IconCircle, PrimaryButton, ScreenHeader, SkyLayer } from '@/shared/ui';

/**
 * `/assistant/[dept]` — department intro (nbl_/nbd_v18_DeptIntro): what it knows about her (each item switchable on the
 * profile screen), what it does NOT do, then «شروع گفت‌وگو». `ask=lab` (the lab result «ask the assistant» link)
 * starts the conversation with a fixed, data-free first message.
 */
export function AssistantDeptPage({ code, ask }: { code: string; ask?: string }) {
  const t = useTranslations('assistant');
  const router = useRouter();
  const errText = useAssistantError();
  const q = useDepartment(code);
  const start = useStartConversation();

  const begin = () =>
    start.mutate(code, {
      onSuccess: (id) => {
        if (ask === 'lab') stashDraft(id, t('dept.askLab'));
        router.push(`/assistant/chat/${id}`);
      },
    });

  let body;
  if (q.isPending) body = <BodySkeleton rows={3} />;
  else if (q.isError || !q.data)
    body = (
      <div className="as-body">
        {q.error && (q.error as { response?: { status?: number } }).response?.status === 404 ? (
          <Card><EmptyState icon="chat" title={t('dept.notFound')} action={<PrimaryButton onClick={() => router.push('/assistant')}>{t('hub.title')}</PrimaryButton>} /></Card>
        ) : (
          <LoadError onRetry={() => void q.refetch()} busy={q.isFetching} />
        )}
      </div>
    );
  else {
    const { department: d, profile } = q.data;
    const shared = profile.consent.granted ? profile.items.filter((i) => i.shared) : [];
    body = (
      <div className="as-body">
        <Card className="as-hero">
          <IconCircle icon={deptIcon(d.icon)} tone="brand" size="lg" />
          <h2 className="as-h2">{d.name}</h2>
          <p className="as-text">{d.intro ?? d.tagline}</p>
        </Card>

        <Card>
          <h3 className="as-h3">{t('dept.knows')}</h3>
          {shared.length > 0 ? (
            <ul className="as-facts">
              {shared.map((i) => (
                <li key={i.key}>
                  <Icon name="check" size={16} />
                  <span>{t(`profile.keys.${i.key}` as 'profile.keys.age')}</span>
                  {scalarOf(i) !== null ? <strong>{scalarOf(i)}</strong> : null}
                </li>
              ))}
              <li>
                <Icon name="history" size={16} />
                <span>{t('dept.previous')}</span>
                <strong>{t('dept.count', { count: d.conversations_count })}</strong>
              </li>
            </ul>
          ) : (
            <p className="as-text">{t('hub.knowsNone')}</p>
          )}
          <p className="as-fine">
            {t('dept.switchHint')} <Link href="/assistant/profile" className="as-link">{t('hub.edit')}</Link>
          </p>
        </Card>

        {d.does_not_do.length > 0 ? (
          <Card className="as-not">
            <h3 className="as-h3"><Icon name="warning" size={16} /> {t('dept.doesNot')}</h3>
            <ul className="as-bullets">
              {d.does_not_do.map((x) => <li key={x}>{x}</li>)}
            </ul>
          </Card>
        ) : null}

        {start.isError ? <p className="as-error" role="alert">{errText(start.error)}</p> : null}
        <PrimaryButton block icon="chat" loading={start.isPending} onClick={begin}>{t('dept.start')}</PrimaryButton>
        <Disclaimer />
      </div>
    );
  }

  return (
    <div className="view as-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={q.data?.department.name ?? t('hub.title')} subtitle={t('dept.subtitle')} onBack={() => router.push('/assistant')} backLabel={t('common.back')} />
        {body}
      </div>
    </div>
  );
}

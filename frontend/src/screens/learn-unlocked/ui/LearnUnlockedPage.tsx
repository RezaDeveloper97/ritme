'use client';

import { useLocale, useTranslations } from 'next-intl';

import { KindThumb, lessonBlock, useCourseMeta, useUnlockedGrant } from '@/entities/course';
import { Link, useRouter, type Locale } from '@/shared/i18n';
import { formatLongDate, formatNumber } from '@/shared/lib/date';
import { Avatar, Card, EmptyState, Icon, PrimaryButton, SecondaryButton, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

/**
 * `/learn/unlocked/[grant]` — Learn_Unlocked, the target of the «… برایت باز
 * کرد» inbox notice: who opened which courses, until when, «شروع دوره».
 */
export function LearnUnlockedPage({ grantId }: { grantId: number }) {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const q = useUnlockedGrant(grantId);
  const meta = useCourseMeta();
  const toHub = () => router.push('/learn');

  let body;
  if (q.isPending) {
    body = (
      <SkeletonGroup label={t('common.loading')} className="lrn-unl-card">
        <Skeleton shape="circle" />
        <Skeleton width="medium" />
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (q.isError) {
    const notFound = lessonBlock(q.error) === 'notFound';
    body = (
      <Card className="lrn-unl-card">
        <EmptyState
          icon={notFound ? 'gradCap' : 'warning'}
          title={t(notFound ? 'unlocked.notFound' : 'common.loadError')}
          action={
            notFound ? (
              <PrimaryButton onClick={toHub}>{t('unlocked.toHub')}</PrimaryButton>
            ) : (
              <PrimaryButton icon="refresh" loading={q.isFetching} onClick={() => void q.refetch()}>
                {t('common.retry')}
              </PrimaryButton>
            )
          }
        />
      </Card>
    );
  } else {
    const g = q.data;
    const first = g.courses[0];
    const until = g.unlimited || !g.expiresAt ? t('access.unlimited') : t('unlocked.until', { date: formatLongDate(new Date(g.expiresAt), locale) });
    body = (
      <div className="lrn-unl-card">
        <div className="lrn-unl-from">
          <Avatar name={g.instructor.name} size="lg" className="lrn-unl-avatar" />
          <span className="lrn-row-main">
            <span className="lrn-row-meta">{t('unlocked.fromInstructor')}</span>
            <span className="lrn-unl-name">{g.instructor.title ? `${g.instructor.name} · ${g.instructor.title}` : g.instructor.name}</span>
          </span>
        </div>
        <h1 className="lrn-unl-title">{t('unlocked.title', { count: g.courses.length, n: formatNumber(g.courses.length, locale) })}</h1>
        <ul className="lrn-unl-list">
          {g.courses.map((c) => (
            <li key={c.id}>
              <Link href={`/learn/${c.id}`} className="lrn-unl-item">
                <KindThumb kind={c.primaryKind} />
                <span className="lrn-row-main">
                  <span className="lrn-row-title">{c.title}</span>
                  <span className="lrn-row-meta">{meta(c)}</span>
                </span>
              </Link>
            </li>
          ))}
        </ul>
        <p className={g.expired ? 'lrn-unl-access is-expired' : 'lrn-unl-access'}>
          <Icon name="clock" size={18} />
          {g.expired ? t('access.expired') : g.group?.name ? t('unlocked.accessVia', { until, group: g.group.name }) : until}
        </p>
        {first && !g.expired ? (
          <PrimaryButton block onClick={() => router.push(`/learn/${first.id}`)}>
            {t('unlocked.start')}
          </PrimaryButton>
        ) : null}
        <SecondaryButton variant="text" block onClick={toHub}>
          {t('unlocked.later')}
        </SecondaryButton>
      </div>
    );
  }

  return (
    <div className="view lrn-view lrn-unl">
      <SkyLayer />
      <div className="scroll">
        <span className="lrn-unl-burst" aria-hidden>
          <Icon name="sparkle" size={56} />
        </span>
        {body}
      </div>
    </div>
  );
}

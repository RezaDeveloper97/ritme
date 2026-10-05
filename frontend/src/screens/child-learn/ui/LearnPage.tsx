'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { useChild } from '@/entities/child';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useDirection, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { AppSheet, openSheet } from '@/shared/sheet';
import {
  Card,
  EmptyState,
  Icon,
  type IconName,
  IconCircle,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  type Tone,
} from '@/shared/ui';

import { useChildLearn } from '../api/queries';
import { learnAge } from '../model/age';
import type { LearnTip } from '../model/types';

type T = ReturnType<typeof useTranslations<'children'>>;

const TOPIC_LOOK: Record<string, { icon: IconName; tone: Tone }> = {
  sleep: { icon: 'moon', tone: 'brand' },
  feeding: { icon: 'bottle', tone: 'data' },
  play: { icon: 'smile', tone: 'success' },
  health: { icon: 'stetho', tone: 'warm' },
  mother: { icon: 'mother', tone: 'bloom' },
};
const look = (topic: string) => TOPIC_LOOK[topic] ?? { icon: 'bookOpen' as IconName, tone: 'neutral' as Tone };

function Shell({ header, children }: { header: ReactNode; children: ReactNode }) {
  return (
    <div className="view chd-screen clr-screen">
      <SkyLayer />
      <div className="scroll">
        {header}
        <div className="chd-body">{children}</div>
      </div>
    </div>
  );
}

/**
 * `/children/[id]/learn` (nbl_v16_Learn): tips for the child's age, topic
 * chips (همه / خواب / تغذیه / …), this week's pick and a list; a tip opens in
 * a sheet (and its full article, when one is published, in the article sheet).
 * General education — the disclaimer stays visible.
 */
export function ChildLearnPage({ id }: { id: number }) {
  const t = useTranslations('children');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const valid = Number.isFinite(id) && id > 0;
  const child = useChild(valid ? id : null);
  const [topic, setTopic] = useState<string | null>(null);
  const [open, setOpen] = useState<LearnTip | null>(null);
  const learn = useChildLearn(valid ? id : 0, topic);

  const name = child.data?.name ?? '';
  const age = learn.data ? learnAge(learn.data.ageMonths) : null;
  const header = (
    <ScreenHeader
      title={t('section.learn')}
      subtitle={
        age && name
          ? t('learn.subtitle', {
              age: t(age.unit === 'years' ? 'learn.ageYears' : 'learn.ageMonths', { n: age.n, count: formatNumber(age.n, locale) }),
              name,
            })
          : undefined
      }
      onBack={() => router.push(valid ? `/children/${id}` : '/children')}
      backLabel={t('common.back')}
    />
  );

  if (!valid || getApiErrorStatus(child.error) === 404 || getApiErrorStatus(learn.error) === 404) {
    return (
      <Shell header={header}>
        <EmptyState
          icon="sprout"
          title={t('home.notFoundTitle')}
          body={t('home.notFoundBody')}
          action={<PrimaryButton onClick={() => router.push('/children')}>{t('home.toList')}</PrimaryButton>}
        />
      </Shell>
    );
  }

  if (!mounted || child.isPending || learn.isPending) {
    return (
      <Shell header={header}>
        <SkeletonGroup label={t('common.loading')}>
          <Skeleton shape="block" />
          <Skeleton shape="card" />
          <Skeleton shape="block" />
          <Skeleton shape="block" />
        </SkeletonGroup>
      </Shell>
    );
  }

  if (child.isError || learn.isError) {
    return (
      <Shell header={header}>
        <Card className="chd-state" role="alert">
          <IconCircle icon="warning" tone="danger" size="lg" />
          <p className="chd-state-text">{t('common.loadError')}</p>
          <SecondaryButton
            icon="refresh"
            block={false}
            loading={learn.isFetching}
            onClick={() => {
              void child.refetch();
              void learn.refetch();
            }}
          >
            {t('common.retry')}
          </SecondaryButton>
        </Card>
      </Shell>
    );
  }

  const data = learn.data;
  const minutes = (n: number | null) => (n ? t('learn.minutes', { n: formatNumber(n, locale) }) : null);
  const featured = data.featured;
  const tips = data.tips.filter((tip) => tip.code !== featured?.code);
  return (
    <Shell header={header}>
      <div className="cms-chips" role="group" aria-label={t('learn.topicsLabel')}>
        {data.topics.map((tp) => {
          const code = tp.code === 'all' ? null : tp.code;
          const on = code === topic;
          return (
            <button
              key={tp.code}
              type="button"
              className={clsx('cms-chip', on && 'is-on')}
              aria-pressed={on}
              onClick={() => setTopic(code)}
            >
              {tp.label}
            </button>
          );
        })}
      </div>

      {featured ? (
        <button type="button" className="clr-featured" onClick={() => setOpen(featured)}>
          <span className="clr-chips">
            <span className="clr-chip">
              <Icon name="sparkle" size={13} />
              {t('learn.featured')}
            </span>
            {minutes(featured.minutes) ? <span className="clr-chip">{minutes(featured.minutes)}</span> : null}
          </span>
          <span className="clr-featured-title">{featured.title}</span>
          {featured.body ? <span className="clr-featured-body">{featured.body}</span> : null}
        </button>
      ) : null}

      {tips.length > 0 ? (
        <ul className="clr-list">
          {tips.map((tip) => (
            <li key={tip.code}>
              <TipRow tip={tip} meta={meta(t, minutes(tip.minutes), tip.topicLabel)} onOpen={() => setOpen(tip)} />
            </li>
          ))}
        </ul>
      ) : null}

      {!featured && tips.length === 0 ? (
        <Card className="clr-empty">
          <IconCircle icon="bookOpen" tone="brand" size="lg" />
          <p className="chd-state-text">{t('learn.empty')}</p>
          {topic ? (
            <SecondaryButton block={false} onClick={() => setTopic(null)}>
              {t('learn.showAll')}
            </SecondaryButton>
          ) : null}
        </Card>
      ) : null}

      {data.disclaimer ? <p className="cgr-disclaimer">{data.disclaimer}</p> : null}

      <AppSheet
        open={open !== null}
        onClose={() => setOpen(null)}
        size="half"
        title={open?.title}
        footer={
          open?.articleSlug ? (
            <PrimaryButton icon="bookOpen" onClick={() => openSheet('article', open.articleSlug ?? undefined)}>
              {t('learn.readMore')}
            </PrimaryButton>
          ) : (
            <SecondaryButton onClick={() => setOpen(null)}>{t('learn.close')}</SecondaryButton>
          )
        }
      >
        {open ? (
          <div className="clr-sheet">
            <span className="clr-chips">
              {open.topicLabel ? <span className="clr-chip">{open.topicLabel}</span> : null}
              {minutes(open.minutes) ? <span className="clr-chip">{minutes(open.minutes)}</span> : null}
            </span>
            {open.body ? <p className="clr-sheet-body">{open.body}</p> : null}
            {data.disclaimer ? <p className="cgr-disclaimer">{data.disclaimer}</p> : null}
          </div>
        ) : null}
      </AppSheet>
    </Shell>
  );
}

function meta(t: T, minutes: string | null, topic: string | null): string {
  if (minutes && topic) return t('learn.meta', { minutes, topic });
  return minutes ?? topic ?? '';
}

function TipRow({ tip, meta: text, onOpen }: { tip: LearnTip; meta: string; onOpen: () => void }) {
  const rtl = useDirection() === 'rtl';
  const { icon, tone } = look(tip.topic);
  return (
    <button type="button" className="clr-row" onClick={onOpen}>
      <IconCircle icon={icon} tone={tone} size="md" className="clr-row-icon" />
      <span className="clr-row-text">
        <b className="clr-row-title">{tip.title}</b>
        {text ? <span className="clr-row-meta">{text}</span> : null}
      </span>
      <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="clr-row-chev" />
    </button>
  );
}

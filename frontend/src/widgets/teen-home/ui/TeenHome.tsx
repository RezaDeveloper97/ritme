'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect } from 'react';

import { type TeenKit, type TeenParentLink, type TeenToday, useTeenToday, useToggleTeenKit } from '@/entities/teen';
import { useUserProfile } from '@/entities/user';
import { getApiSaveErrorMessage } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  Card,
  Checkbox,
  EmptyState,
  Icon,
  IconCircle,
  InfoNote,
  PrimaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
} from '@/shared/ui';

import { signsCard } from '../model/signs';

/**
 * Teen home (CB-TEEN-02, nbl_Teen_Home) — replaces bloom's simplified teen
 * cycle home (B-N2-03). Content only; the home screen adds the backdrop and
 * the bottom nav. Signs card + estimate, the school-kit checklist
 * (`PUT /teen/kit/{code}`), «طبیعی است؟» FAQ, the when-to-talk note and the
 * mother-sharing entry. Minors: no shop, banners, ads or Plus upsell here, and
 * no fertility copy (CB-TEEN-01 `allows`). Without onboarding answers she is
 * sent to `/teen/onboarding`.
 */
export function TeenHome() {
  const t = useTranslations('teen.home');
  const router = useRouter();
  const query = useTeenToday();
  const needsOnboarding = query.data?.needsOnboarding ?? false;

  useEffect(() => {
    if (needsOnboarding) router.replace('/teen/onboarding');
  }, [needsOnboarding, router]);

  let body;
  if (query.isPending || needsOnboarding) {
    body = (
      <SkeletonGroup label={t('loading')} className="tnh-skel">
        <Skeleton shape="block" className="tnh-skel-card" />
        <Skeleton shape="block" className="tnh-skel-card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="sprout"
        title={t('loadError')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <TeenBody data={query.data} />;
  }

  return (
    <div className="tnh-page">
      <TeenHeader />
      {body}
    </div>
  );
}

function TeenHeader() {
  const t = useTranslations('teen.home');
  const profile = useUserProfile();
  const first = profile.data?.name?.trim().split(/\s+/)[0] ?? '';
  return (
    <header className="tnh-hdr">
      <div className="tnh-hdr-text">
        <span className="tnh-eyebrow">{t('eyebrow')}</span>
        <h1 className="tnh-hello">{first ? t('hello', { name: first }) : t('helloNoName')}</h1>
      </div>
      <IconCircle icon="sprout" tone="data" size="lg" />
    </header>
  );
}

function TeenBody({ data }: { data: TeenToday }) {
  const t = useTranslations('teen.home');
  return (
    <>
      <SignsCardView data={data} />
      <KitCard kit={data.kit} />
      {data.faq.length ? (
        <section className="tnh-sec" aria-labelledby="tnh-faq">
          <SectionTitle id="tnh-faq" title={t('faq.title')} />
          <Card className="tnh-faq">
            {data.faq.map((item) =>
              item.title ? (
                <details key={item.code} className="tnh-faq-item">
                  <summary className="tnh-faq-q">
                    <Icon name="chevronDown" size={16} className="tnh-faq-chev" />
                    <span>{item.title}</span>
                  </summary>
                  {item.body ? <p className="tnh-faq-a">{item.body}</p> : null}
                </details>
              ) : null,
            )}
          </Card>
        </section>
      ) : null}
      {data.talkNote?.body ? (
        <InfoNote className="tnh-talk">
          <span className="sr-only">{data.talkNote.title ?? t('talk')}: </span>
          {data.talkNote.body}
        </InfoNote>
      ) : null}
      <ParentEntry links={data.parentLinks} />
    </>
  );
}

function SignsCardView({ data }: { data: TeenToday }) {
  const t = useTranslations('teen.home.signs');
  const card = signsCard(data.signs, data.readiness);
  if (!card) return null;
  const hasPeriods = data.profile !== null && data.profile.menarche !== 'not_yet';
  return (
    <Card as="section" className={clsx('tnh-signs', card.caution && 'is-caution')} aria-labelledby="tnh-signs-t">
      <h2 id="tnh-signs-t" className="tnh-signs-title">
        {card.title ?? t('title')}
      </h2>
      {card.body ? <p className="tnh-signs-body">{card.body}</p> : null}
      {card.extra.map((sign) => (
        <p key={sign.code} className="tnh-signs-extra">
          {sign.title ? <b>{sign.title}</b> : null}
          {sign.title && sign.body ? ' · ' : null}
          {sign.body}
        </p>
      ))}
      {card.percent !== null ? (
        <div className="tnh-bar" role="img" aria-label={t('estimate')}>
          {/* Data-driven width: the estimate's position (CLAUDE.md §10.1 exception). */}
          <span className="tnh-bar-fill" style={{ inlineSize: `${card.percent}%` }} />
        </div>
      ) : null}
      {card.estimate ? <p className="tnh-estimate">{card.estimate}</p> : null}
      {hasPeriods ? (
        <Link href="/calendar" className="tnh-link">
          {t('calendar')}
        </Link>
      ) : null}
    </Card>
  );
}

function KitCard({ kit }: { kit: TeenKit }) {
  const t = useTranslations('teen.home.kit');
  const locale = useLocale() as Locale;
  const toggle = useToggleTeenKit();
  if (!kit.items.length) return null;
  return (
    <section className="tnh-sec" aria-labelledby="tnh-kit">
      <div className="tnh-sec-head">
        <SectionTitle id="tnh-kit" title={t('title')} />
        <span className="tnh-count">
          {t('count', { checked: formatNumber(kit.checkedCount, locale), total: formatNumber(kit.total, locale) })}
        </span>
      </div>
      <Card className="tnh-kit">
        {kit.items.map((item) => (
          <Checkbox
            key={item.code}
            checked={item.checked}
            label={item.title ?? item.code}
            onCheckedChange={(checked) => toggle.mutate({ code: item.code, checked })}
          />
        ))}
        {kit.ready ? (
          <p className="tnh-kit-ready" role="status">
            <Icon name="checkCircle" size={16} />
            {t('ready')}
          </p>
        ) : null}
      </Card>
      {toggle.isError ? (
        <p className="tnh-error" role="alert">
          {getApiSaveErrorMessage(toggle.error, t('saveError'))}
        </p>
      ) : null}
    </section>
  );
}

/** «همراهی مادر» → the mother-sharing screen (CB-TEEN-03, nbl_Teen_Parent). */
const PARENT_HREF = '/teen/parent';

function ParentEntry({ links }: { links: TeenParentLink[] }) {
  const t = useTranslations('teen.home.parent');
  const active = links.some((l) => l.status === 'active');
  const invited = !active && links.some((l) => l.status === 'invited');
  return (
    <Link href={PARENT_HREF} className="tnh-parent">
      <Icon name="heart" size={18} className="tnh-parent-icon" />
      <span className="tnh-parent-text">
        <span className="tnh-parent-label">{t('link')}</span>
        {active || invited ? <span className="tnh-parent-status">{t(active ? 'active' : 'invited')}</span> : null}
      </span>
    </Link>
  );
}

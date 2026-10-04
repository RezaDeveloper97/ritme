'use client';

import { useLocale, useTranslations } from 'next-intl';

import { type Child, ChildAvatar, ChildStatusChips, useChildren } from '@/entities/child';
import { useLifeStage } from '@/entities/user';
import { Link, type Locale, useDirection, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  IconCircle,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

type T = ReturnType<typeof useTranslations<'children'>>;

function ChildRow({ child, t }: { child: Child; t: T }) {
  const rtl = useDirection() === 'rtl';
  return (
    <li>
      <Link href={`/children/${child.id}`} className="nb-card chd-row" aria-label={t('list.open', { name: child.name })}>
        <ChildAvatar id={child.id} name={child.name} initial={child.initial} sex={child.sex} size={60} />
        <span className="chd-row-text">
          <span className="chd-row-head">
            <b className="chd-row-name">{child.name}</b>
            {child.sexLabel ? <span className="chd-chip nb-tone-brand">{child.sexLabel}</span> : null}
          </span>
          <span className="chd-row-age">{child.age.label}</span>
          {child.role === 'shared' && child.ownerName ? (
            <span className="chd-row-owner">{t('list.sharedBy', { name: child.ownerName })}</span>
          ) : null}
          <ChildStatusChips child={child} />
        </span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="chd-row-chev" />
      </Link>
    </li>
  );
}

/**
 * `/children` — «فرزندان من» (nbl_v15_Children): own children first, then the
 * ones a spouse shares (read-only, «از طرف …»), each with the next vaccine
 * visit and the growth verdict. Add lives in the header and at the bottom (owners
 * under the limit); a companion account never sees it. Also the «کودک» tab
 * target for 0 or ≥ 2 children (nav.md).
 */
export function ChildrenPage() {
  const t = useTranslations('children');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const life = useLifeStage();
  const query = useChildren();

  const companion = Boolean(life.data?.companion);
  const backTo = companion ? '/companion' : life.data?.mode === 'postpartum' ? '/postpartum' : '/profile';
  const data = query.data;
  const canAdd = !companion && (data?.canAdd ?? false);
  const add = () => router.push('/children/new');

  let body: React.ReactNode;
  if (!mounted || query.isPending) {
    body = (
      <SkeletonGroup label={t('common.loading')}>
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError || !data) {
    body = (
      <Card className="chd-state" role="alert">
        <IconCircle icon="warning" tone="danger" size="lg" />
        <p className="chd-state-text">{t('common.loadError')}</p>
        <SecondaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
          {t('common.retry')}
        </SecondaryButton>
      </Card>
    );
  } else if (data.children.length === 0) {
    body = companion ? (
      <EmptyState icon="users" title={t('list.emptySharedTitle')} body={t('list.emptySharedBody')} />
    ) : (
      <EmptyState
        icon="sprout"
        title={t('list.emptyTitle')}
        body={t('list.emptyBody')}
        action={
          canAdd ? (
            <PrimaryButton icon="plus" onClick={add}>
              {t('list.add')}
            </PrimaryButton>
          ) : undefined
        }
      />
    );
  } else {
    body = (
      <>
        <ul className="chd-list">
          {data.children.map((c) => (
            <ChildRow key={c.id} child={c} t={t} />
          ))}
        </ul>
        {!companion && data.sharingNote ? (
          <Card className="chd-note">
            <IconCircle icon="info" tone="brand" />
            <p>{data.sharingNote}</p>
          </Card>
        ) : null}
        {canAdd ? (
          <SecondaryButton icon="plus" onClick={add}>
            {t('list.add')}
          </SecondaryButton>
        ) : !companion && data.ownedCount >= data.maxChildren ? (
          <p className="chd-hint chd-limit">{t('list.limit', { max: formatNumber(data.maxChildren, locale) })}</p>
        ) : null}
      </>
    );
  }

  const count = data?.children.length ?? 0;
  return (
    <div className="view chd-screen">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('list.title')}
          subtitle={data && count > 0 ? t('list.count', { count: formatNumber(count, locale) }) : undefined}
          onBack={() => router.push(backTo)}
          backLabel={t('common.back')}
          action={canAdd ? <HeaderButton icon="plus" label={t('list.add')} onClick={add} /> : undefined}
        />
        <div className="chd-body">{body}</div>
      </div>
      <BottomNav childIds={data?.children.map((c) => c.id)} />
    </div>
  );
}

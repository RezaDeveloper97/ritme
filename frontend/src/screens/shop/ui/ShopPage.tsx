'use client';

import { useTranslations } from 'next-intl';

import { useShopCategories } from '@/entities/shop';
import { Link, useRouter } from '@/shared/i18n';
import { Card, EmptyState, Icon, PrimaryButton, ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

/**
 * `/shop` — «فروشگاه» categories (entry from the services hub tile, nbl_/nbd_v17_Main). Separate from health data:
 * the disclosure line says so and nothing here depends on the user's logs.
 */
export function ShopPage() {
  const t = useTranslations('shop');
  const router = useRouter();
  const q = useShopCategories();

  let body;
  if (q.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="shp-grid">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (q.isError) {
    body = (
      <Card>
        <EmptyState
          icon="warning"
          title={t('loadError')}
          action={
            <PrimaryButton onClick={() => void q.refetch()} loading={q.isFetching}>
              {t('retry')}
            </PrimaryButton>
          }
        />
      </Card>
    );
  } else if (q.data.categories.length === 0) {
    body = (
      <Card>
        <EmptyState icon="bag" title={t('empty.title')} body={t('empty.body')} />
      </Card>
    );
  } else {
    body = (
      <ul className="shp-grid">
        {q.data.categories.map((c, i) => (
          <li key={c.slug}>
            <Link href={`/shop/${c.slug}`} className={i % 2 ? 'shp-cat is-alt' : 'shp-cat'}>
              <span className="shp-cat-art" aria-hidden>
                <Icon name={c.icon} size={44} strokeWidth={1.4} />
              </span>
              <span className="shp-cat-title">{c.title}</span>
            </Link>
          </li>
        ))}
      </ul>
    );
  }

  return (
    <div className="view shp-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('title')} subtitle={t('tag')} onBack={() => router.push('/services')} backLabel={t('back')} />
        <div className="shp-body">
          {body}
          <p className="shp-note">{t('note')}</p>
        </div>
        <div className="page-tail" />
      </div>
    </div>
  );
}

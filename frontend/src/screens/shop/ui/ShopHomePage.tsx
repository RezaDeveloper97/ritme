'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useShopCatalogHome } from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';
import { Link, useRouter } from '@/shared/i18n';
import { Card, EmptyState, Icon, PrimaryButton, ScreenHeader, SegmentedTabs, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

import { ProductTile } from './ProductTile';
import { ShopPage as ShopLinkOutPage } from './ShopPage';
import { ChecklistCard, SupplyReminderCard } from './ShopHomeCards';

/**
 * `/shop` home (CB-SHOP-06, nbd_Shop_Home + nbd_Shop_Beauty): root tabs (baby / beauty), sub-category chips, best
 * sellers per sub-category, beauty IRC card, trust row and the separation note. Extends bloom's B-N10-03 `/shop`:
 * when the internal catalog has nothing yet (404) it renders bloom's link-out categories unchanged. A teen-mode
 * account (403) sees a plain unavailable state. Pay on delivery only — no online-payment wording.
 *
 * Product tiles open the product page (CB-SHOP-07). Not here yet: the layette-checklist card (CB-SHOP-09), the
 * period-supply reminder card (CB-SHOP-04 API) and the header cart / city (CB-SHOP-08) — each lands with its API.
 * CB-SHOP-09 adds the layette-checklist card (baby) and the discreet supply-reminder toggle (beauty).
 * Not here yet: the header cart / city (CB-SHOP-08) and search + product pages (CB-SHOP-07) — each lands with its API.
 */
export function ShopHomePage() {
  const t = useTranslations('shop');
  const router = useRouter();
  const [root, setRoot] = useState('');
  const q = useShopCatalogHome(root);
  const status = q.isError ? getApiErrorStatus(q.error) : undefined;

  // No internal catalog yet → bloom's link-out categories (unchanged).
  if (status === 404) return <ShopLinkOutPage />;

  let body;
  if (q.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="shp-grid">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (status === 403) {
    body = (
      <Card>
        <EmptyState icon="shield" title={t('home.teenTitle')} body={t('home.teenBody')} />
      </Card>
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
  } else {
    const home = q.data;
    const isBeauty = home.root.slug === 'beauty';
    body = (
      <>
        {home.roots.length > 1 ? (
          <SegmentedTabs
            label={t('home.rootsLabel')}
            tabs={home.roots.map((r) => ({ value: r.slug, label: r.title }))}
            value={home.root.slug}
            onChange={setRoot}
          />
        ) : null}
        {home.sub_categories.length ? (
          <ul className="shp-chips" aria-label={t('home.categories')}>
            {home.sub_categories.map((c) => (
              <li key={c.id}>
                <Link href={`/shop/${c.slug}`} className="shp-chip">
                  {c.title}
                </Link>
              </li>
            ))}
          </ul>
        ) : null}
        {isBeauty ? (
          <Card className="shp-irc">
            <Icon name="shield" size={22} />
            <div>
              <p className="shp-irc-title">{t('home.ircTitle')}</p>
              <p className="shp-irc-body">{t('home.ircBody')}</p>
            </div>
          </Card>
        ) : null}
        {isBeauty ? <SupplyReminderCard /> : <ChecklistCard />}
        {home.sections.length === 0 ? (
          <Card>
            <EmptyState icon="bag" title={t('empty.title')} body={t('empty.body')} />
          </Card>
        ) : (
          home.sections.map((s) => (
            <section key={s.category.id} className="shp-sec" aria-labelledby={`shp-sec-${s.category.id}`}>
              <div className="shp-sec-head">
                <h2 id={`shp-sec-${s.category.id}`} className="shp-sec-title">
                  {t('home.bestOf', { name: s.category.title })}
                </h2>
                <Link href={`/shop/${s.category.slug}`} className="shp-sec-all">
                  {t('home.all')}
                </Link>
              </div>
              <ul className="shp-tiles">
                {s.products.map((p) => (
                  <ProductTile key={p.id} p={p} />
                ))}
              </ul>
            </section>
          ))
        )}
        <ul className="shp-trust" aria-label={t('home.trustLabel')}>
          {(['genuine', 'returns', 'fast', 'discreet'] as const).map((k) => (
            <li key={k} className="shp-trust-item">
              <span className="shp-trust-title">{t(`home.trust.${k}.title`)}</span>
              <span className="shp-trust-body">{t(`home.trust.${k}.body`)}</span>
            </li>
          ))}
        </ul>
        <p className="shp-note">{t('home.cod')}</p>
      </>
    );
  }

  return (
    <div className="view shp-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('title')} subtitle={t('tag')} onBack={() => router.push('/services')} backLabel={t('back')} />
        <div className="shp-body">
          {body}
          <p className="shp-note">{t('home.separation')}</p>
        </div>
        <div className="page-tail" />
      </div>
    </div>
  );
}

'use client';

import { useTranslations } from 'next-intl';

import { type ShopCatalogHome, useShopCatalogHome } from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';
import { Link } from '@/shared/i18n';
import { Card, EmptyState, Icon, Skeleton, SkeletonGroup } from '@/shared/ui';

import { APP_SHOP, heroKey, storeCategoryHref } from '../model/store';
import { ShopWebShell, WebTile } from './ShopWebShell';
import '@/shared/ui/styles/shp.css';
import './wsh.css';

/** One root (baby or beauty): its category chips and best-seller sections. The first root arrives server-rendered. */
function RootBlock({ slug, initial }: { slug: string; initial?: ShopCatalogHome }) {
  const t = useTranslations('shop');
  const w = useTranslations('webShop');
  const q = useShopCatalogHome(slug, { anon: true, initialData: initial });
  if (q.isPending) {
    return (
      <SkeletonGroup label={t('loading')} className="wsh-skel">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  }
  if (q.isError) return null;
  const home = q.data;
  return (
    <section className="wsh-root" aria-label={home.root.title}>
      {home.sub_categories.length > 0 ? (
        <>
          <h2 className="wsh-h2">{w('home.categoriesOf', { name: home.root.title })}</h2>
          <ul className="wsh-cats">
            {home.sub_categories.map((c) => (
              <li key={c.id}>
                <Link href={storeCategoryHref(c.slug)} className="wsh-cat">
                  <span className="wsh-cat-art" aria-hidden>
                    <Icon name={c.icon} size={30} strokeWidth={1.5} />
                  </span>
                  <span className="wsh-cat-name">{c.title}</span>
                </Link>
              </li>
            ))}
          </ul>
        </>
      ) : null}
      {home.sections.slice(0, 2).map((s) => (
        <div key={s.category.id} className="wsh-sec">
          <div className="shp-sec-head">
            <h2 className="wsh-h2">{t('home.bestOf', { name: s.category.title })}</h2>
            <Link href={storeCategoryHref(s.category.slug)} className="shp-sec-all">
              {t('home.all')}
            </Link>
          </div>
          <ul className="wsh-tiles">
            {s.products.map((p) => (
              <WebTile key={p.id} p={p} />
            ))}
          </ul>
        </div>
      ))}
    </section>
  );
}

/**
 * CB-SHOP-10 — `W_Shop_Home`: hero with the two roots (baby and beauty), category strip and best sellers per root, trust
 * strip. Public; the server hands over the first root so the HTML already carries products. COD only.
 */
export function WebShopHome({ initial }: { initial?: ShopCatalogHome | null }) {
  const t = useTranslations('shop');
  const w = useTranslations('webShop');
  const q = useShopCatalogHome('', { anon: true, initialData: initial ?? undefined });
  const status = q.isError ? getApiErrorStatus(q.error) : undefined;

  let body;
  if (q.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="wsh-skel">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (q.isError) {
    body = (
      <Card>
        <EmptyState
          icon={status === 403 ? 'shield' : status === 404 ? 'bag' : 'warning'}
          title={status === 403 ? t('home.teenTitle') : status === 404 ? t('empty.title') : t('loadError')}
          body={status === 403 ? t('home.teenBody') : status === 404 ? t('empty.body') : undefined}
        />
      </Card>
    );
  } else {
    const home = q.data;
    const roots = home.roots.length > 0 ? home.roots : [home.root];
    body = (
      <>
        <ul className="wsh-heroes">
          {roots.map((r) => (
            <li key={r.id} className={`wsh-hero-card is-${heroKey(r.slug)}`}>
              <span className="wsh-hero-art" aria-hidden>
                <Icon name={r.icon} size={72} strokeWidth={1.3} />
              </span>
              <div className="wsh-hero-copy">
                <h2 className="wsh-hero-title">{r.title}</h2>
                <p className="wsh-hero-body">{w(`home.heroBody.${heroKey(r.slug)}`)}</p>
                <Link href={storeCategoryHref(r.slug)} className="wsh-go">
                  {w('home.enter')}
                  <Icon name="arrowR" size={16} />
                </Link>
              </div>
            </li>
          ))}
        </ul>
        {roots.map((r) => (
          <RootBlock key={r.slug} slug={r.slug} initial={r.slug === home.root.slug ? home : undefined} />
        ))}
        <ul className="shp-trust wsh-trust" aria-label={t('home.trustLabel')}>
          {(['genuine', 'returns', 'fast', 'discreet'] as const).map((k) => (
            <li key={k} className="shp-trust-item">
              <span className="shp-trust-title">{t(`home.trust.${k}.title`)}</span>
              <span className="shp-trust-body">{t(`home.trust.${k}.body`)}</span>
            </li>
          ))}
        </ul>
      </>
    );
  }

  return (
    <ShopWebShell appHref={APP_SHOP}>
      <div className="wsh">
        <header className="wsh-head">
          <p className="wsh-eyebrow">{w('home.eyebrow')}</p>
          <h1 className="wsh-h1">{w('home.title')}</h1>
          <p className="wsh-lead">{w('home.lead')}</p>
        </header>
        {body}
        <p className="shp-note wsh-note">{t('home.cod')}</p>
      </div>
    </ShopWebShell>
  );
}

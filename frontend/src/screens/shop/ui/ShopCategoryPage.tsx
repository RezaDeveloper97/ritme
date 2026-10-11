'use client';

import { useFormatter, useTranslations } from 'next-intl';
import { useState } from 'react';

import { countShopClick, useShopProducts, type ShopProduct } from '@/entities/shop';
import { useRouter } from '@/shared/i18n';
import { Card, EmptyState, Icon, PrimaryButton, ScreenHeader, SecondaryButton, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

/**
 * `/shop/[category]` — the products of one category, then the link-out. Tapping a product with a link shows the
 * interstitial line «به سایت فروشنده می‌روی»; «ادامه» opens the seller's site (`noopener sponsored`) and counts the
 * click. The URL carries no user identifier and no history is kept. Products without a link render without action.
 */
export function ShopCategoryPage({ category }: { category: string }) {
  const t = useTranslations('shop');
  const fmt = useFormatter();
  const router = useRouter();
  const [page, setPage] = useState(1);
  const [open, setOpen] = useState<number | null>(null);
  const q = useShopProducts(category, page);

  const card = (p: ShopProduct) => {
    const inner = (
      <>
        <span className="shp-thumb" aria-hidden>
          {p.image_url ? (
            // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
            <img src={p.image_url} alt="" loading="lazy" />
          ) : (
            <Icon name="bag" size={28} strokeWidth={1.4} />
          )}
        </span>
        <span className="shp-prod-main">
          <span className="shp-prod-title">{p.title}</span>
          <span className="shp-prod-meta">
            {p.badge ? <span className="shp-badge">{p.badge}</span> : null}
            {p.price_text ? <span className="shp-price">{p.price_text}</span> : null}
          </span>
          {p.partner_label ? <span className="shp-partner">{t('partner', { name: p.partner_label })}</span> : null}
        </span>
      </>
    );
    return (
      <li key={p.id} className="shp-prod">
        {p.link_url ? (
          <button type="button" className="shp-prod-row" aria-expanded={open === p.id} onClick={() => setOpen(open === p.id ? null : p.id)}>
            {inner}
            <Icon name="chevronLeft" size={18} className="shp-prod-go" />
          </button>
        ) : (
          <div className="shp-prod-row is-static">{inner}</div>
        )}
        {p.link_url && open === p.id ? (
          <div className="shp-inter" role="group" aria-label={t('interstitial.label', { title: p.title })}>
            <p className="shp-inter-line">{t('interstitial.line')}</p>
            <div className="shp-inter-actions">
              <a
                className="shp-go"
                href={p.link_url}
                target="_blank"
                rel="noopener noreferrer sponsored"
                onClick={() => {
                  countShopClick(p.id);
                  setOpen(null);
                }}
              >
                {t('interstitial.go')}
                <Icon name="arrowR" size={16} />
              </a>
              <SecondaryButton variant="text" onClick={() => setOpen(null)}>
                {t('interstitial.cancel')}
              </SecondaryButton>
            </div>
          </div>
        ) : null}
      </li>
    );
  };

  let body;
  if (q.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="shp-list">
        <Skeleton shape="block" />
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
  } else if (q.data.products.length === 0) {
    body = (
      <Card>
        <EmptyState icon="bag" title={t('products.emptyTitle')} body={t('products.emptyBody')} />
      </Card>
    );
  } else {
    body = (
      <>
        <ul className="shp-list">{q.data.products.map(card)}</ul>
        {q.data.meta.last_page > 1 ? (
          <nav className="shp-pager" aria-label={t('products.pages')}>
            <SecondaryButton variant="text" disabled={page <= 1} onClick={() => setPage((n) => n - 1)}>
              {t('products.prev')}
            </SecondaryButton>
            <span>{t('products.page', { page: fmt.number(page), last: fmt.number(q.data.meta.last_page) })}</span>
            <SecondaryButton variant="text" disabled={page >= q.data.meta.last_page} onClick={() => setPage((n) => n + 1)}>
              {t('products.next')}
            </SecondaryButton>
          </nav>
        ) : null}
      </>
    );
  }

  return (
    <div className="view shp-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={q.data?.category?.title ?? t('title')} onBack={() => router.push('/shop')} backLabel={t('back')} />
        <div className="shp-body">
          {body}
          <p className="shp-note">{t('note')}</p>
        </div>
        <div className="page-tail" />
      </div>
    </div>
  );
}

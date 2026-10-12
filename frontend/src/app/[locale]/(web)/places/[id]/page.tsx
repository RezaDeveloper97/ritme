import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { getTranslations, setRequestLocale } from 'next-intl/server';

import { fetchPublicListing } from '@/entities/city-service';
import { WebPlacePage } from '@/screens/web-city-services';

import { RouteMessages } from '../../../../RouteMessages';
import { siteOrigin as originOf } from '../../site-origin';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

const parseId = (id: string): number | null => {
  const n = Number(id);
  return Number.isInteger(n) && n > 0 ? n : null;
};

/**
 * Title / description / OpenGraph / canonical from the public API, server-side. Only the venue's own public fields
 * are read — no session, no cookies, no user data.
 */
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, id } = await params;
  const n = parseId(id);
  const t = await getTranslations({ locale, namespace: 'webDirectory' });
  const l = n ? await fetchPublicListing(n) : null;
  if (!l) return { title: t('place.metaFallback'), robots: { index: false, follow: true } };

  const title = t('place.metaTitle', { name: l.name, category: l.category.name, city: l.city.name });
  const description = (l.about ?? l.description)?.slice(0, 160) || t('place.metaDescription', { name: l.name, category: l.category.name, city: l.city.name });
  const origin = await originOf();
  const path = `/${locale}/places/${l.id}`;
  const url = origin ? `${origin}${path}` : path;
  const image = l.photos[0] ?? l.photo_url;
  return {
    title,
    description,
    alternates: { canonical: url },
    openGraph: { title, description, url, type: 'website', siteName: 'Ritme', ...(image ? { images: [{ url: image, alt: l.name }] } : {}) },
    twitter: { card: image ? 'summary_large_image' : 'summary', title, description },
  };
}

/** `/places/[id]` — one venue for the web: server-fetched so the first HTML carries the content (CB-DIR-11). */
export default async function PlaceRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const n = parseId(id);
  if (!n) notFound();
  const initial = await fetchPublicListing(n);
  return (
    <RouteMessages route="webDirectory">
      <WebPlacePage id={n} initial={initial} />
    </RouteMessages>
  );
}

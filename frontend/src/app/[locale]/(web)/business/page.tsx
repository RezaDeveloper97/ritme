import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';

import { fetchBusinessTerms } from '@/entities/city-service';
import { BusinessLandingPage } from '@/screens/web-city-services-business';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

const FAQ = ['time', 'rank', 'phone', 'privacy'] as const;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'webBusiness' });
  return {
    title: t('meta.title'),
    description: t('meta.description'),
    alternates: { canonical: `/${locale}/business` },
    openGraph: { title: t('meta.title'), description: t('meta.description'), type: 'website', url: `/${locale}/business` },
  };
}

/** `/business` — the public for-businesses landing (CB-DIR-12). `business` is a public segment (no session needed). */
export default async function BusinessRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'webBusiness' });
  const terms = await fetchBusinessTerms(locale); // public + admin-edited; [] → the neutral placeholder
  // FAQ structured data: the same questions the page shows. `<` is escaped so the text can never close the script tag.
  const faqLd = JSON.stringify({
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: FAQ.map((k) => ({
      '@type': 'Question',
      name: t(`faq.${k}.q`),
      acceptedAnswer: { '@type': 'Answer', text: t(`faq.${k}.a`) },
    })),
  }).replace(/</g, '\\u003c');
  return (
    <RouteMessages route="webBusiness">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: faqLd }} />
      <BusinessLandingPage terms={terms} />
    </RouteMessages>
  );
}

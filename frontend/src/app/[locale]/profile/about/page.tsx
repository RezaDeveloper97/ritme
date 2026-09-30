import { setRequestLocale } from 'next-intl/server';

import { AboutPage } from '@/screens/about';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/about` — درباره ریتمی (B-N1-12, nbl_Me_About). */
export default async function ProfileAboutRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="profileAbout">
      <AboutPage />
    </RouteMessages>
  );
}

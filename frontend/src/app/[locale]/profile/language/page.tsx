import { setRequestLocale } from 'next-intl/server';

import { LanguagePage } from '@/screens/appearance';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/language` — زبان و تقویم (B-N1-10, no artboard). */
export default async function ProfileLanguageRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="profileLanguage">
      <LanguagePage />
    </RouteMessages>
  );
}

import { setRequestLocale } from 'next-intl/server';

import { LabIntroPage } from '@/screens/lab-intro';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/labs` — nbl_Lab_Intro (B-N6-07): lab history + upload entry. */
export default async function LabIntroPageRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="labs">
      <LabIntroPage />
    </RouteMessages>
  );
}

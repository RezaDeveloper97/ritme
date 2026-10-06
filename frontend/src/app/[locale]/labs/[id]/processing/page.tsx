import { setRequestLocale } from 'next-intl/server';

import { LabProcessingPage } from '@/screens/lab-processing';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/labs/[id]/processing` — nbl_Lab_Processing (B-N6-07): status poll. */
export default async function LabProcessingPageRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="labDetail">
      <LabProcessingPage id={Number(id)} />
    </RouteMessages>
  );
}

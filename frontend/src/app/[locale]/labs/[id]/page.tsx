import { setRequestLocale } from 'next-intl/server';

import { LabResultPage } from '@/screens/lab-result';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/labs/[id]` — nbl_Lab_Result (B-N6-07); other statuses redirect to processing / verify. */
export default async function LabResultPageRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="labDetail">
      <LabResultPage id={Number(id)} />
    </RouteMessages>
  );
}

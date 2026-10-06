import { setRequestLocale } from 'next-intl/server';

import { LabMarkerPage } from '@/screens/lab-marker';

import { RouteMessages } from '../../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string; mid: string }>;
}

/** `/labs/[id]/markers/[mid]` — nbl_Lab_Marker (B-N6-07). */
export default async function LabMarkerPageRoute({ params }: Props) {
  const { locale, id, mid } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="labDetail">
      <LabMarkerPage id={Number(id)} markerId={Number(mid)} />
    </RouteMessages>
  );
}

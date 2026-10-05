import { setRequestLocale } from 'next-intl/server';

import { ChildVaccinesPage } from '@/screens/child-vaccines';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/children/[id]/vaccines` — nbl_v16_Vaccines (B-N5-06): schedule / card / notes, mark given, book a visit. */
export default async function ChildVaccinesRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="childSection">
      <ChildVaccinesPage id={Number(id)} />
    </RouteMessages>
  );
}

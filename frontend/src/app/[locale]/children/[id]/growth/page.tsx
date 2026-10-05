import { setRequestLocale } from 'next-intl/server';

import { ChildGrowthPage } from '@/screens/child-growth';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/children/[id]/growth` — nbl_v16_Growth (B-N5-06): WHO chart, history, add / edit measurements. */
export default async function ChildGrowthRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="childSection">
      <ChildGrowthPage id={Number(id)} />
    </RouteMessages>
  );
}

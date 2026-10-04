import { setRequestLocale } from 'next-intl/server';

import { ChildSectionSoon } from '@/screens/child-home';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/children/[id]/growth` — interim page until B-N5-06 builds v16_Growth (B-N5-05 links here). */
export default async function ChildGrowthRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="childSection">
      <ChildSectionSoon id={Number(id)} section="growth" />
    </RouteMessages>
  );
}

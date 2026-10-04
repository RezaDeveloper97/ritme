import { setRequestLocale } from 'next-intl/server';

import { ChildSectionSoon } from '@/screens/child-home';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/children/[id]/milestones` — interim page until B-N5-06 builds v16_Milestones (B-N5-05 links here). */
export default async function ChildMilestonesRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="childSection">
      <ChildSectionSoon id={Number(id)} section="milestones" />
    </RouteMessages>
  );
}

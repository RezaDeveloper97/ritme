import { setRequestLocale } from 'next-intl/server';

import { ChildMilestonesPage } from '@/screens/child-milestones';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/children/[id]/milestones` — nbl_v16_Milestones (B-N5-06): month chips, checklist, play ideas. */
export default async function ChildMilestonesRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="childSection">
      <ChildMilestonesPage id={Number(id)} />
    </RouteMessages>
  );
}

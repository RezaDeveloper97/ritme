import { setRequestLocale } from 'next-intl/server';

import { ChildLearnPage } from '@/screens/child-learn';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/children/[id]/learn` — nbl_v16_Learn (B-N5-06): tips by age and topic, tip sheet. */
export default async function ChildLearnRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="childSection">
      <ChildLearnPage id={Number(id)} />
    </RouteMessages>
  );
}

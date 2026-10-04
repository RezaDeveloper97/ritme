import { setRequestLocale } from 'next-intl/server';

import { ChildHomePage } from '@/screens/child-home';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/children/[id]` — the child home (nbl_v16_ChildHome, B-N5-05), mode tab «کودک». */
export default async function ChildHomeRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="childHome">
      <ChildHomePage id={Number(id)} />
    </RouteMessages>
  );
}

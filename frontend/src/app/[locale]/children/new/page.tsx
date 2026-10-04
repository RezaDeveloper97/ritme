import { setRequestLocale } from 'next-intl/server';

import { ChildFormPage } from '@/screens/child-add';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/children/new` — add a child (nbl_v15_AddChild, B-N5-05). */
export default async function ChildNewRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="childForm">
      <ChildFormPage />
    </RouteMessages>
  );
}

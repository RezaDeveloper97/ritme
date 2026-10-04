import { setRequestLocale } from 'next-intl/server';

import { ChildFormPage } from '@/screens/child-add';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/children/[id]/edit` — edit or delete a child (B-N5-05, same form as /children/new). */
export default async function ChildEditRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="childForm">
      <ChildFormPage childId={Number(id)} />
    </RouteMessages>
  );
}

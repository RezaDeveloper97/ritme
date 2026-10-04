import { setRequestLocale } from 'next-intl/server';

import { ChildrenPage } from '@/screens/children';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/children` — «فرزندان من» (nbl_v15_Children, B-N5-05). */
export default async function ChildrenRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="children">
      <ChildrenPage />
    </RouteMessages>
  );
}

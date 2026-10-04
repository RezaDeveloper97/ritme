import { setRequestLocale } from 'next-intl/server';

import { TeenParentPage } from '@/screens/teen-parent';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/teen/parent` — mother sharing (CB-TEEN-03, nbl_Teen_Parent). A flow with a back header: no bottom nav. */
export default async function TeenParentRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="teenParent">
      <TeenParentPage />
    </RouteMessages>
  );
}

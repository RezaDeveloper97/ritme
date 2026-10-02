import { setRequestLocale } from 'next-intl/server';

import { CompanionsPage } from '@/screens/companion-list';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/companions` — همدم‌ها (B-N4-04, nbl_Hamdam_List). */
export default async function CompanionsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="companions">
      <CompanionsPage />
    </RouteMessages>
  );
}

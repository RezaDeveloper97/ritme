import { setRequestLocale } from 'next-intl/server';

import { PostpartumSetupPage } from '@/screens/postpartum';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/postpartum/setup` — activation form (birth date, delivery type, baby count). A form: no bottom nav. */
export default async function PostpartumSetupRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="postpartumSetup">
      <PostpartumSetupPage />
    </RouteMessages>
  );
}

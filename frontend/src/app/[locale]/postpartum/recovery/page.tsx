import { setRequestLocale } from 'next-intl/server';

import { PostpartumRecoveryPage } from '@/screens/postpartum-recovery';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/postpartum/recovery` — today's recovery form (nbl_v15_Recovery). Back header, no bottom nav. */
export default async function PostpartumRecoveryRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="postpartumRecovery">
      <PostpartumRecoveryPage />
    </RouteMessages>
  );
}

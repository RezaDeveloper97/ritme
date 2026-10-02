import { setRequestLocale } from 'next-intl/server';

import { PostpartumPage } from '@/screens/postpartum';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/postpartum` — postpartum «امروز» (nbl_v15_Main, B-N5-04); `/home` hands postpartum users here. */
export default async function PostpartumRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="postpartum">
      <PostpartumPage />
    </RouteMessages>
  );
}

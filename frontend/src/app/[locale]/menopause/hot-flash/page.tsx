import { setRequestLocale } from 'next-intl/server';

import { MenopauseHotFlashPage } from '@/screens/menopause-hot-flash';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/menopause/hot-flash` — گرگرفتگی (CB-MENO-07, nbl_Meno_HotFlash). A flow: no bottom nav. */
export default async function MenopauseHotFlashRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="menopauseHotFlash">
      <MenopauseHotFlashPage />
    </RouteMessages>
  );
}

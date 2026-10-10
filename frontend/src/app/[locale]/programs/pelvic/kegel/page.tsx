import { setRequestLocale } from 'next-intl/server';

import { PelvicKegelPage } from '@/screens/pelvic-kegel';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/programs/pelvic/kegel` — «تمرین کف لگن» (CB-PELV-02, nbl_Pelvic_Kegel). Full-screen flow: no bottom nav. */
export default async function PelvicKegelRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="pelvicKegel">
      <PelvicKegelPage />
    </RouteMessages>
  );
}

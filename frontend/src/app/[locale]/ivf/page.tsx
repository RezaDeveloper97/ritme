import { setRequestLocale } from 'next-intl/server';

import { IvfHomePage } from '@/screens/ivf';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/ivf` — IVF «امروز» (nbl_IVF_Home, CB-IVF-02); `/home` hands TTC users with «IVF/IUI» on here. */
export default async function IvfRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="ivf">
      <IvfHomePage />
    </RouteMessages>
  );
}

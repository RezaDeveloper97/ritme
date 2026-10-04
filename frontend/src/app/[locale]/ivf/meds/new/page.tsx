import { setRequestLocale } from 'next-intl/server';

import { IvfMedFormPage } from '@/screens/ivf-meds';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/ivf/meds/new` — «افزودن دارو از روی نسخه» (CB-IVF-03; form, no nav). */
export default async function NewIvfMedRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="ivfMedForm">
      <IvfMedFormPage />
    </RouteMessages>
  );
}

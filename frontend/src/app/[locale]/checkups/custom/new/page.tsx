import { setRequestLocale } from 'next-intl/server';

import { CustomCheckupFormPage } from '@/screens/checkup-custom-form';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function NewCustomCheckupRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="checkups">
      <CustomCheckupFormPage />
    </RouteMessages>
  );
}

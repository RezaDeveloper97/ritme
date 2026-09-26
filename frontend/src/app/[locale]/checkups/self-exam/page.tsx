import { setRequestLocale } from 'next-intl/server';

import { SelfExamPage } from '@/screens/checkup-self-exam';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function SelfExamRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="checkups">
      <SelfExamPage />
    </RouteMessages>
  );
}

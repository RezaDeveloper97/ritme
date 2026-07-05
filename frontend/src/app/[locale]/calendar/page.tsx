import { setRequestLocale } from 'next-intl/server';

import { CalendarPage } from '@/screens/calendar';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function CalendarRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <CalendarPage />;
}

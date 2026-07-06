import { setRequestLocale } from 'next-intl/server';

import { RemindersPage } from '@/screens/profile-reminders';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function RemindersRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <RemindersPage />;
}

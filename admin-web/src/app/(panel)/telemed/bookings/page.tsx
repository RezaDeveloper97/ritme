import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { BookingsScreen } from '@/screens/telemed';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('telemed.bookings');
  return { title: t('title') };
}

export default function Page() {
  return <BookingsScreen />;
}

import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { AdminsScreen } from '@/screens/admins';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('admins');
  return { title: t('title') };
}

export default function AdminsPage() {
  return <AdminsScreen />;
}

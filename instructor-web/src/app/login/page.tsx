import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { LoginScreen } from '@/screens/login';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('login');
  return { title: t('metaTitle') };
}

export default function LoginPage() {
  return <LoginScreen />;
}

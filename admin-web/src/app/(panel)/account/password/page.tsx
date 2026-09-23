import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { PasswordScreen } from '@/screens/account-password';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('password');
  return { title: t('title') };
}

export default function AccountPasswordPage() {
  return <PasswordScreen />;
}

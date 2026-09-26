import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { AlertRulesScreen } from '@/screens/pregnancy-alert-rules';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('pregnancyAlertRules');
  return { title: t('title') };
}

export default function PregnancyAlertRulesPage() {
  return <AlertRulesScreen />;
}

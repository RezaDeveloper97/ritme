import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { AlertRuleFormScreen } from '@/screens/pregnancy-alert-rules';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('pregnancyAlertRules');
  return { title: t('edit') };
}

export default async function PregnancyAlertRuleKeyPage({ params }: { params: Promise<{ key: string }> }) {
  const key = (await params).key;
  if (!/^[a-z][a-z0-9_]*$/.test(key)) notFound();
  return <AlertRuleFormScreen ruleKey={key} />;
}

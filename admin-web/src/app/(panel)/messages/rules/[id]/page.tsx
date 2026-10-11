import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { RuleEditorScreen } from '@/screens/message-rules';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('messageRules');
  return { title: t('editTitle') };
}

export default async function MessageRuleIdPage({ params }: { params: Promise<{ id: string }> }) {
  const raw = (await params).id;
  if (raw === 'new') return <RuleEditorScreen id={null} />;
  const id = parseRouteId(raw);
  if (id === null) notFound();
  return <RuleEditorScreen id={id} />;
}

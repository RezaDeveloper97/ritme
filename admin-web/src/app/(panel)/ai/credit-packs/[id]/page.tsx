import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { PackFormScreen } from '@/screens/ai-credits';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('aiCredits.packs');
  return { title: t('edit') };
}

export default async function AiCreditPackEditPage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <PackFormScreen id={id} />;
}

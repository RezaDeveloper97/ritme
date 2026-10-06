import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { PostpartumContentScreen } from '@/screens/postpartum-content';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('postpartumContent');
  return { title: t('title') };
}

export default function PostpartumContentPage() {
  return <PostpartumContentScreen />;
}

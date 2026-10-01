import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { DiscountFormScreen } from '@/screens/plus-billing';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('plus.discounts');
  return { title: t('new') };
}

export default function PlusDiscountNewPage() {
  return <DiscountFormScreen id={null} />;
}

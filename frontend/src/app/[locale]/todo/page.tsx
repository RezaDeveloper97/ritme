import { setRequestLocale } from 'next-intl/server';

import { TodoPage } from '@/screens/todo';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/todo` — «کارهای من» (nbl_Todo_Home / Todo_Empty, B-N6-08). A hub: bottom nav («من»). */
export default async function TodoRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="todo">
      <TodoPage />
    </RouteMessages>
  );
}

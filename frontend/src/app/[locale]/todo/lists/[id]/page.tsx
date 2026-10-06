import { setRequestLocale } from 'next-intl/server';

import { TodoListPage } from '@/screens/todo-list';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/todo/lists/[id]` — one list (nbl_Todo_List, B-N6-08); a non-numeric id shows «not found». */
export default async function TodoListRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="todoList">
      <TodoListPage id={/^\d{1,12}$/.test(id) ? Number(id) : 0} />
    </RouteMessages>
  );
}

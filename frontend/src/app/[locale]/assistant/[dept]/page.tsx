import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { AssistantDeptPage } from '@/screens/assistant-dept';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; dept: string }>;
  searchParams: Promise<{ ask?: string }>;
}

/** `/assistant/[dept]` — department intro (nbl_v18_DeptIntro); `?ask=lab` = the lab «ask the assistant» link (B-N7-07). */
export default async function AssistantDeptPageRoute({ params, searchParams }: Props) {
  const { locale, dept } = await params;
  setRequestLocale(locale);
  const { ask } = await searchParams;
  return (
    <RouteMessages route="assistantDept">
      <Suspense>
        <AssistantDeptPage code={/^[a-z_]{1,32}$/.test(dept) ? dept : ""} ask={ask === "lab" ? "lab" : undefined} />
      </Suspense>
    </RouteMessages>
  );
}

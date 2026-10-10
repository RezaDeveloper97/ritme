import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { LearnLessonPage } from '@/screens/learn-lesson';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; course: string; lesson: string }>;
}

const id = (v: string) => (/^\d{1,12}$/.test(v) ? Number(v) : 0);

/** `/learn/[course]/[lesson]` — video / audio / PDF lesson (nbl_Learn_Video / Audio / PDF, B-N8-03); `?audio=1` = audio only. */
export default async function LearnLessonRoute({ params }: Props) {
  const { locale, course, lesson } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="learnLesson">
      <Suspense>
        <LearnLessonPage courseId={id(course)} lessonId={id(lesson)} />
      </Suspense>
    </RouteMessages>
  );
}

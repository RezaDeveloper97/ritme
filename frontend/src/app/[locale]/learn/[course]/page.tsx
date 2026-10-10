import { setRequestLocale } from 'next-intl/server';

import { LearnCoursePage } from '@/screens/learn-course';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; course: string }>;
}

/** `/learn/[course]` — one course (nbl_Learn_Course, B-N8-03); a non-numeric id shows «not found». */
export default async function LearnCourseRoute({ params }: Props) {
  const { locale, course } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="learnCourse">
      <LearnCoursePage courseId={/^\d{1,12}$/.test(course) ? Number(course) : 0} />
    </RouteMessages>
  );
}

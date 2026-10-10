import Link from 'next/link';

import type { CourseListItem } from '../model/schema';
import { KindTile } from './KindTile';
import { StatusBadge } from './StatusBadge';

/** One course row (Ins_Dashboard «دوره‌های من», Ins_Content): kind tile, title, meta, status pill. */
export function CourseRow({ course, meta }: { course: CourseListItem; meta: string }) {
  const kind = course.lesson_kinds[0] ?? 'video';
  const draft = course.status === 'draft';
  return (
    <li>
      <Link href={`/content/${course.id}`} className="row">
        <KindTile kind={kind} draft={draft} />
        <span className="row__body">
          <span className="row__title">{course.title}</span>
          <span className="row__meta">{meta}</span>
        </span>
        <StatusBadge status={draft ? 'draft' : course.kind === 'standalone' ? 'standalone' : 'published'} />
      </Link>
    </li>
  );
}

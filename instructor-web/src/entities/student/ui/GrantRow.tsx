import type { ReactNode } from 'react';

import { cn } from '@/shared/lib';

import { initialOf, phoneLabel } from '../model/display';
import type { Grant } from '../model/schema';

/** One person row (Ins_Group members, Ins_Students): avatar, name-or-phone, meta, trailing slot. */
export function GrantRow({ grant, meta, trailing }: { grant: Grant; meta: string; trailing?: ReactNode }) {
  const named = Boolean(grant.student_name?.trim());
  return (
    <li className="row person">
      <span className={cn('avatar avatar--sm', !named && 'avatar--ghost')} aria-hidden="true">
        {initialOf(grant)}
      </span>
      <span className="row__body">
        <span className="row__title">{named ? grant.student_name : <bdi dir="ltr">{phoneLabel(grant.phone)}</bdi>}</span>
        <span className="row__meta">{meta}</span>
      </span>
      {trailing}
    </li>
  );
}

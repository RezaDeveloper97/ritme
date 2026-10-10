import { cn } from '@/shared/lib';
import { Icon } from '@/shared/ui';

import type { LessonKind } from '../model/schema';

const ICON = { video: 'video', audio: 'audio', pdf: 'pdf' } as const;

/** Rounded kind tile of a course / lesson row (video bloom, audio data, pdf warm; drafts brand). */
export function KindTile({ kind, draft = false, size = 'md' }: { kind: LessonKind; draft?: boolean; size?: 'sm' | 'md' }) {
  return (
    <span className={cn('kind-tile', `kind-tile--${draft ? 'draft' : kind}`, size === 'sm' && 'kind-tile--sm')} aria-hidden="true">
      <Icon name={ICON[kind]} size={size === 'sm' ? 18 : 22} />
    </span>
  );
}

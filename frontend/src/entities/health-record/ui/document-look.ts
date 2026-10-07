import type { IconName, Tone } from '@/shared/ui';

import type { CategoryKey, TimelineItemKind } from '../model/documents';

/** Icon disc of a document kind / home category (boards nbl_Rec_Home, nbl_Rec_Timeline). */
export const KIND_LOOK: Record<TimelineItemKind, { icon: IconName; tone: Tone }> = {
  lab: { icon: 'flask', tone: 'period' },
  imaging: { icon: 'grid', tone: 'brand' },
  visit: { icon: 'stetho', tone: 'brand' },
  prescription: { icon: 'fileDoc', tone: 'data' },
  hospital: { icon: 'bed', tone: 'bloom' },
  other: { icon: 'note', tone: 'neutral' },
};

export const CATEGORY_LOOK: Record<CategoryKey | 'all', { icon: IconName; tone: Tone }> = {
  labs: KIND_LOOK.lab,
  imaging: KIND_LOOK.imaging,
  visit: KIND_LOOK.visit,
  prescription: KIND_LOOK.prescription,
  hospital: KIND_LOOK.hospital,
  other: KIND_LOOK.other,
  surgeries: { icon: 'syringe', tone: 'warm' },
  family_history: { icon: 'users', tone: 'neutral' },
  all: { icon: 'box', tone: 'brand' },
};

import type { IconName, Tone } from '@/shared/ui';

import type { VoiceDiaryTarget } from '../api/voice';

/** Icon + accent of the diaries `POST /logs/voice/commit` writes to (they aren't log-sheet categories). */
const DIARY_LOOK: Record<VoiceDiaryTarget, { icon: IconName; tone: Tone }> = {
  hot_flash: { icon: 'flame', tone: 'period' },
  pain_diary: { icon: 'symptom', tone: 'bloom' },
  pill: { icon: 'tablet', tone: 'bloom' },
  bladder: { icon: 'urine', tone: 'brand' },
};

export function diaryLook(target: VoiceDiaryTarget): { icon: IconName; tone: Tone } {
  return DIARY_LOOK[target];
}

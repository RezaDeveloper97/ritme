import type { IconName, Tone } from '@/shared/ui';

import type { ContraceptionMethodCode } from './types';

/**
 * Icon + tone per method (nbl_Contra_Setup chips). Red stays menstruation-only
 * (CLAUDE.md §10.2), so the board's rose hormonal-IUD shield is warm here.
 */
export const METHOD_LOOK: Record<ContraceptionMethodCode, { icon: IconName; tone: Tone }> = {
  combined_pill: { icon: 'pill', tone: 'brand' },
  progestin_pill: { icon: 'capsule', tone: 'data' },
  copper_iud: { icon: 'shield', tone: 'bloom' },
  hormonal_iud: { icon: 'shield', tone: 'warm' },
  injection: { icon: 'calendar', tone: 'brand' },
  implant: { icon: 'hand', tone: 'data' },
  condom: { icon: 'heart', tone: 'bloom' },
  other: { icon: 'info', tone: 'neutral' },
};

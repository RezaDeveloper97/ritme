import type { IconName, Tone } from '@/shared/ui';

import type { SearchHit } from './types';

/** Icon disc of a result row (Nav_Search): what the hit is, in the palette's accents. */
export function hitLook(hit: SearchHit): { icon: IconName; tone: Tone } {
  switch (hit.type) {
    case 'log_insight':
      if (hit.meta.category === 'bleeding') return { icon: 'drop', tone: 'period' };
      if (hit.meta.category === 'pain') return { icon: 'flame', tone: 'bloom' };
      if (hit.meta.category === 'measurements') return { icon: 'scale', tone: 'data' };
      return { icon: 'symptom', tone: 'brand' };
    case 'log_analysis':
      return { icon: 'chart', tone: 'brand' };
    case 'reminder':
      return hit.meta.kind === 'appointment' ? { icon: 'calendar', tone: 'warm' } : { icon: 'pill', tone: 'data' };
    case 'program':
      return { icon: 'heart', tone: 'bloom' };
    case 'article':
      return { icon: 'bookOpen', tone: 'warm' };
    case 'checkup':
      return { icon: 'todo', tone: 'data' };
    case 'service':
      return { icon: 'stetho', tone: 'data' };
  }
}

import type { CycleClinicalFlags } from '@/entities/cycle';

export type FlagHintKind = 'absent' | 'frequent' | 'infrequent' | 'prolonged';

/**
 * Which calm «worth mentioning to your doctor» hint to show for the engine's
 * clinical flags (v6 §2.27), or null for none. Never a diagnosis: the copy only
 * says the pattern differs from the usual range. Missing flags (Laravel
 * payloads), `normal` values and unknown labels give no hint. Priority: absent,
 * frequency, duration.
 */
export function flagHintKind(flags: CycleClinicalFlags | null | undefined): FlagHintKind | null {
  if (!flags) return null;
  if (flags.absent) return 'absent';
  if (flags.frequency === 'frequent' || flags.frequency === 'infrequent') return flags.frequency;
  if (flags.duration === 'prolonged') return 'prolonged';
  return null;
}

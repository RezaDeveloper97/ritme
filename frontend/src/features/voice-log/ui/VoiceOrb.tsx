'use client';

import { Art3d, Icon } from '@/shared/ui';
import './vorb.css';

export type OrbState = 'idle' | 'listening' | 'thinking';

/** Rendered box in px; the file is 480px, sharp on 3x screens. */
const ORB_SIZE = 148;

interface VoiceOrbProps {
  state: OrbState;
  /** Accessible name: «شروع ضبط» / «پایان ضبط». */
  label: string;
  disabled?: boolean;
  onPress: () => void;
}

/**
 * The one control of voice logging: a round, lit orb (`public/art3d/voice-orb.webp`). Tap to talk, tap again to finish. It deliberately has no
 * `pilot()` name — it submits to an AI model (frontend/CLAUDE.md §11.6, risky controls).
 */
export function VoiceOrb({ state, label, disabled, onPress }: VoiceOrbProps) {
  return (
    <span className="vorb-wrap" data-state={state}>
      <span className="vorb-ring" aria-hidden />
      <button
        type="button"
        className="vorb"
        aria-label={label}
        aria-pressed={state === 'listening'}
        aria-busy={state === 'thinking'}
        disabled={disabled || state === 'thinking'}
        onClick={onPress}
      >
        <Art3d name="voice-orb" size={ORB_SIZE} className="vorb-art" />
        {/* The glyph sits on the orb's saturated fill in both themes: the theme-stable label colour. */}
        <span className="vorb-glyph">
          {state === 'thinking' ? (
            <Icon name="loader" size={40} stroke="var(--on-accent)" className="vorb-spin" />
          ) : state === 'listening' ? (
            <span className="vorb-stop" aria-hidden />
          ) : (
            <Icon name="mic" size={50} strokeWidth={1.8} stroke="var(--on-accent)" />
          )}
        </span>
      </button>
    </span>
  );
}

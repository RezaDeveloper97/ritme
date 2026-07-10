/**
 * The intro carousel's slides, in order. This is pure config — copy lives in
 * the `welcome` i18n namespace and is looked up by `id`; the visual identity
 * (which illustration, which accent) is decided here so the UI stays a plain
 * renderer. Accents are token names, resolved to CSS custom properties at
 * render time so both light and dark themes stay correct.
 */
export type IntroSlideId = 'intro' | 'track' | 'smart';

export type IntroIllustration = 'rhythm' | 'track' | 'insight';

/** Names of CSS custom properties (see app/globals.css `:root`). */
export type AccentToken = '--brand' | '--pink' | '--violet' | '--green';

export interface IntroSlide {
  id: IntroSlideId;
  illustration: IntroIllustration;
  /** Gradient stops for this slide's ambient glow + illustration accent. */
  accentFrom: AccentToken;
  accentTo: AccentToken;
  /** Whether this slide renders the two feature bullets. */
  bullets: boolean;
  /** Whether this slide renders the AI/medical disclaimer + the CTA hint. */
  disclaimer: boolean;
}

export const INTRO_SLIDES: readonly IntroSlide[] = [
  {
    id: 'intro',
    illustration: 'rhythm',
    accentFrom: '--pink',
    accentTo: '--brand',
    bullets: false,
    disclaimer: false,
  },
  {
    id: 'track',
    illustration: 'track',
    accentFrom: '--brand',
    accentTo: '--violet',
    bullets: true,
    disclaimer: false,
  },
  {
    id: 'smart',
    illustration: 'insight',
    accentFrom: '--violet',
    accentTo: '--brand',
    bullets: false,
    disclaimer: true,
  },
] as const;

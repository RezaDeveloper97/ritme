/**
 * The intro carousel's slides, in order (Night & Bloom `Intro_1` … `Intro_5`).
 * Pure config — copy lives in the `welcome` i18n namespace under
 * `slides.<id>`; the UI picks the illustration by id.
 */
export type IntroSlideId = 'cycle' | 'journey' | 'health' | 'privacy' | 'free';

export const INTRO_SLIDES: readonly IntroSlideId[] = [
  'cycle',
  'journey',
  'health',
  'privacy',
  'free',
] as const;

/** Slides whose copy has a paragraph under the title (4 and 5 show a list instead). */
export type BodySlideId = Exclude<IntroSlideId, 'privacy' | 'free'>;

export function hasSlideBody(id: IntroSlideId): id is BodySlideId {
  return id !== 'privacy' && id !== 'free';
}

/**
 * «رد کردن» jumps to the last slide rather than leaving: slide 5 is the
 * always-free promise, which the design wants every newcomer to see.
 */
export const SKIP_TARGET = INTRO_SLIDES.length - 1;

/** The illustrative "today" of slide 1's cycle ring (day 23 of 29) and slide 2's pregnancy week. */
export const INTRO_CYCLE_TODAY = 22;
export const INTRO_NEXT_PERIOD_DAYS = 6;
export const INTRO_PREGNANCY_WEEK = 32;

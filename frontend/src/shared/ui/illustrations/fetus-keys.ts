/**
 * The fetus-size illustration keys — the contract between the admin week
 * editor's illustration picker (T-M7-06/07, `pregnancy_week_details.illustration_key`)
 * and {@link FetusSize}. The admin API should validate against exactly this
 * list; a key the bundle doesn't know renders the neutral `fetus` drawing.
 *
 * Each key is drawn from a small set of shapes × palette tints, so adding a
 * fruit is one row here, not a new drawing.
 */
export const FETUS_ILLUSTRATION_KEYS = [
  'fetus',
  'poppy_seed',
  'sesame_seed',
  'lentil',
  'blueberry',
  'raspberry',
  'cherry',
  'strawberry',
  'fig',
  'lime',
  'plum',
  'lemon',
  'peach',
  'apple',
  'orange',
  'avocado',
  'pear',
  'bell_pepper',
  'mango',
  'banana',
  'carrot',
  'papaya',
  'grapefruit',
  'corn',
  'cauliflower',
  'lettuce',
  'cabbage',
  'eggplant',
  'squash',
  'cucumber',
  'coconut',
  'pineapple',
  'cantaloupe',
  'honeydew',
  'watermelon',
  'pumpkin',
] as const;

export type FetusIllustrationKey = (typeof FETUS_ILLUSTRATION_KEYS)[number];

export type FetusShape =
  | 'fetus'
  | 'seed'
  | 'round'
  | 'cluster'
  | 'drop'
  | 'long'
  | 'leafy'
  | 'striped'
  | 'spiky';

/** A `.pg2-fruit-*` class in globals.css — every tint flips with the theme. */
export type FruitTint = 'berry' | 'yellow' | 'orange' | 'green' | 'purple' | 'brown' | 'cream';

/** Drawn size inside the disc: a seed stays small, a melon fills it. */
export type FruitSize = 's' | 'm' | 'l';

export interface FetusDrawing {
  shape: FetusShape;
  tint: FruitTint;
  size: FruitSize;
}

const D = (shape: FetusShape, tint: FruitTint, size: FruitSize): FetusDrawing => ({ shape, tint, size });

export const FETUS_DRAWINGS: Record<FetusIllustrationKey, FetusDrawing> = {
  fetus: D('fetus', 'berry', 'm'),
  poppy_seed: D('seed', 'brown', 's'),
  sesame_seed: D('seed', 'cream', 's'),
  lentil: D('seed', 'orange', 's'),
  blueberry: D('round', 'purple', 's'),
  raspberry: D('cluster', 'berry', 'm'),
  cherry: D('round', 'berry', 's'),
  strawberry: D('drop', 'berry', 'm'),
  fig: D('drop', 'purple', 'm'),
  lime: D('round', 'green', 'm'),
  plum: D('round', 'purple', 'm'),
  lemon: D('round', 'yellow', 'm'),
  peach: D('round', 'orange', 'm'),
  apple: D('round', 'berry', 'm'),
  orange: D('round', 'orange', 'm'),
  avocado: D('drop', 'green', 'm'),
  pear: D('drop', 'green', 'm'),
  bell_pepper: D('round', 'green', 'm'),
  mango: D('drop', 'orange', 'm'),
  banana: D('long', 'yellow', 'm'),
  carrot: D('long', 'orange', 'm'),
  papaya: D('drop', 'orange', 'l'),
  grapefruit: D('round', 'orange', 'l'),
  corn: D('long', 'yellow', 'l'),
  cauliflower: D('leafy', 'cream', 'l'),
  lettuce: D('leafy', 'green', 'l'),
  cabbage: D('leafy', 'green', 'l'),
  eggplant: D('long', 'purple', 'l'),
  squash: D('drop', 'orange', 'l'),
  cucumber: D('long', 'green', 'l'),
  coconut: D('round', 'brown', 'l'),
  pineapple: D('spiky', 'yellow', 'l'),
  cantaloupe: D('striped', 'orange', 'l'),
  honeydew: D('striped', 'green', 'l'),
  watermelon: D('striped', 'green', 'l'),
  pumpkin: D('striped', 'orange', 'l'),
};

export function isFetusIllustrationKey(value: unknown): value is FetusIllustrationKey {
  return typeof value === 'string' && (FETUS_ILLUSTRATION_KEYS as readonly string[]).includes(value);
}

/** The drawing for any key; unknown / missing → the neutral fetus. */
export function fetusDrawing(key: string | null | undefined): FetusDrawing {
  return isFetusIllustrationKey(key) ? FETUS_DRAWINGS[key] : FETUS_DRAWINGS.fetus;
}

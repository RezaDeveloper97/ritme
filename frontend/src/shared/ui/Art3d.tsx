import { clsx } from 'clsx';

import type { IconName } from './Icon';

/**
 * The 3D feature illustrations (glossy clay style, brand purple → rose with a turquoise accent). They mark the app's
 * features and services — a care tile, a life mode, a shop category — where a picture helps recognition. Controls,
 * states and inline glyphs stay on {@link Icon}: those must follow the text colour and read at 16–20px.
 *
 * Files live in `public/art3d/<name>.webp` (320×320, transparent), generated with Canva.
 */
export const ART3D_NAMES = [
  'cycle', 'ttc', 'pregnancy', 'postpartum', 'menopause', 'teen',
  'assistant', 'doctors', 'record', 'labs', 'vitals', 'insurance',
  'pain', 'mood', 'heavy-bleeding', 'pelvic-floor', 'contraception',
  'layette', 'beauty', 'shop', 'learning', 'checkups', 'medication', 'mother-child', 'credits',
] as const;
export type Art3dName = (typeof ART3D_NAMES)[number];

/** A file under `public/art3d/`: a feature illustration, or the 3D twin of a line glyph in `icons/` (200px). */
export type Art3dPath = Art3dName | `icons/${IconName}` | 'icons/child' | 'icons/gift' | 'fab-plus';

/**
 * The 3D twin of every line glyph: {@link Icon} draws the picture instead of the line whenever one is listed here
 * (owner decision 2026-11-12: every icon in the user app is 3D, small ones and controls included). Controls
 * (arrows, check, close, plus, play…) are turquoise so they still read on a brand-coloured button.
 */
export const ICON_ART: Partial<Record<IconName, Art3dPath>> = {
  alarm: 'icons/alarm',
  apple: 'icons/apple',
  bed: 'icons/bed',
  bell: 'icons/bell',
  bellRing: 'icons/bellRing',
  book: 'icons/book',
  bookOpen: 'icons/bookOpen',
  bookmark: 'icons/bookmark',
  bottle: 'icons/bottle',
  box: 'icons/box',
  brain: 'icons/brain',
  briefcase: 'icons/briefcase',
  cake: 'icons/cake',
  calendar: 'icons/calendar',
  camera: 'icons/camera',
  capsule: 'icons/capsule',
  card: 'icons/card',
  chart: 'icons/chart',
  chat: 'icons/chat',
  checkCircle: 'icons/checkCircle',
  clock: 'icons/clock',
  cog: 'icons/cog',
  contrast: 'icons/contrast',
  copy: 'icons/copy',
  crown: 'icons/crown',
  doctor: 'icons/doctor',
  download: 'icons/download',
  drop: 'icons/drop',
  dropLine: 'icons/dropLine',
  export: 'icons/export',
  eye: 'icons/eye',
  faceGood: 'icons/faceGood',
  faceGreat: 'icons/faceGreat',
  faceHard: 'icons/faceHard',
  faceLow: 'icons/faceLow',
  faceOkay: 'icons/faceOkay',
  female: 'icons/female',
  fileDoc: 'icons/fileDoc',
  flame: 'icons/flame',
  flaskLh: 'icons/flaskLh',
  gauge: 'icons/gauge',
  glass: 'icons/glass',
  globe: 'icons/globe',
  glucose: 'icons/glucose',
  grid: 'icons/grid',
  gut: 'icons/gut',
  hand: 'icons/hand',
  heart: 'icons/heart',
  heartLine: 'icons/heartLine',
  help: 'icons/help',
  history: 'icons/history',
  home: 'icons/home',
  image: 'icons/image',
  implant: 'icons/implant',
  info: 'icons/info',
  lock: 'icons/lock',
  logout: 'icons/logout',
  male: 'icons/male',
  mapPin: 'icons/mapPin',
  mic: 'icons/mic',
  modeRing: 'icons/modeRing',
  moon: 'icons/moon',
  moonReminder: 'icons/moonReminder',
  mother: 'icons/mother',
  music: 'icons/music',
  note: 'icons/note',
  pencil: 'icons/pencil',
  phone: 'icons/phone',
  pill: 'icons/pill',
  pin: 'icons/pin',
  qr: 'icons/qr',
  ribbon: 'icons/ribbon',
  ruler: 'icons/ruler',
  run: 'icons/run',
  scale: 'icons/scale',
  scaleSquare: 'icons/scaleSquare',
  search: 'icons/search',
  send: 'icons/send',
  share: 'icons/share',
  sleep: 'icons/sleep',
  smartphone: 'icons/smartphone',
  smile: 'icons/smile',
  sparkle: 'icons/sparkle',
  star: 'icons/star',
  sun: 'icons/sun',
  symptom: 'icons/symptom',
  syringe: 'icons/syringe',
  tablet: 'icons/tablet',
  thermo: 'icons/thermo',
  todo: 'icons/todo',
  tooth: 'icons/tooth',
  trash: 'icons/trash',
  urine: 'icons/urine',
  user: 'icons/user',
  users: 'icons/users',
  video: 'icons/video',
  walk: 'icons/walk',
  warning: 'icons/warning',
  watch: 'icons/watch',
  zap: 'icons/zap',
  chevronRight: 'icons/chevronRight',
  chevronLeft: 'icons/chevronLeft',
  chevronDown: 'icons/chevronDown',
  chevronUp: 'icons/chevronUp',
  arrowL: 'icons/arrowL',
  arrowR: 'icons/arrowR',
  check: 'icons/check',
  x: 'icons/x',
  plus: 'icons/plus',
  minus: 'icons/minus',
  loader: 'icons/loader',
  refresh: 'icons/refresh',
  filterLines: 'icons/filterLines',
  grip: 'icons/grip',
  moreH: 'icons/moreH',
  play: 'icons/play',
  pause: 'icons/pause',
  rewind: 'icons/rewind',
  forward: 'icons/forward',
  expand: 'icons/expand',
  stetho: 'doctors',
  shield: 'insurance',
  flask: 'labs',
  gradCap: 'learning',
  sprout: 'teen',
  bag: 'shop',
  target: 'ttc',
  bellPlain: 'icons/bell',
  pen: 'icons/pencil',
};

interface Art3dProps {
  name: Art3dPath;
  /** Rendered box in px (feature files are 320px, `icons/` files 200px — sharp up to 100px on 2x screens). */
  size?: number;
  /** When the picture alone carries meaning; otherwise it is decorative. */
  label?: string;
  className?: string;
}

export function Art3d({ name, size = 44, label, className }: Art3dProps) {
  return (
    // eslint-disable-next-line @next/next/no-img-element -- tiny static webp; next/image adds nothing here
    <img
      src={`/art3d/${name}.webp`}
      alt={label ?? ''}
      aria-hidden={label ? undefined : true}
      width={size}
      height={size}
      loading="lazy"
      decoding="async"
      draggable={false}
      className={clsx('art3d', className)}
    />
  );
}

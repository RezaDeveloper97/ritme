import type { CompanionAction, CompanionGoal, CompanionLinkSettings } from '@/entities/companion';
import type { IconName, Tone } from '@/shared/ui';

/*
 * «اقدام‌های همراهی» (companion v2, compCards): a static catalogue of gentle ideas, grouped by the five
 * support kinds the owner can prefer (`action_prefs`). Copy lives under `companionHome.actions.*`; every
 * card is a suggestion («اگر دوست داشتی…»), never a task, and none needs a purchase.
 */

export type ActionCardKey =
  | 'listen'
  | 'askSpace'
  | 'hug'
  | 'warmDrink'
  | 'snack'
  | 'chores'
  | 'errands'
  | 'quiet'
  | 'smallGesture'
  | 'note'
  | 'walk'
  | 'calmEvening';

export interface ActionCardDef {
  /** Message key under `companionHome.actions.cards`. */
  key: ActionCardKey;
  icon: IconName;
}

export interface ActionCategoryDef {
  /** The owner-side preference this category answers to. */
  id: CompanionAction;
  tone: Tone;
  cards: readonly ActionCardDef[];
}

/** Default order when the owner set no preference: calm first, money-free help before gifts. */
export const ACTION_CATALOGUE: readonly ActionCategoryDef[] = [
  {
    id: 'calm',
    tone: 'bloom',
    cards: [
      { key: 'listen', icon: 'heart' },
      { key: 'askSpace', icon: 'smile' },
      { key: 'hug', icon: 'users' },
    ],
  },
  {
    id: 'food',
    tone: 'warm',
    cards: [
      { key: 'warmDrink', icon: 'glass' },
      { key: 'snack', icon: 'star' },
    ],
  },
  {
    id: 'help',
    tone: 'data',
    cards: [
      { key: 'chores', icon: 'hand' },
      { key: 'errands', icon: 'bag' },
      { key: 'quiet', icon: 'moon' },
    ],
  },
  {
    id: 'gift',
    tone: 'brand',
    cards: [
      { key: 'smallGesture', icon: 'sparkle' },
      { key: 'note', icon: 'note' },
    ],
  },
  {
    id: 'time',
    tone: 'success',
    cards: [
      { key: 'walk', icon: 'walk' },
      { key: 'calmEvening', icon: 'sun' },
    ],
  },
];

/** Which category a stated goal leans towards (none for «cycle» / «info» / «unknown»). */
const GOAL_CATEGORY: Partial<Record<CompanionGoal, CompanionAction>> = {
  support: 'calm',
  low_energy: 'help',
  quality_time: 'time',
};

/**
 * Categories in display order: the owner's preferred kinds first (in the order she chose them), then the
 * one her goal leans towards, then the rest in the catalogue's default order. Unknown ids are ignored.
 */
export function orderedCategories(
  settings: Pick<CompanionLinkSettings, 'actionPrefs' | 'goal'> | null,
): ActionCategoryDef[] {
  const byId = new Map(ACTION_CATALOGUE.map((c) => [c.id, c] as const));
  const order: CompanionAction[] = [];
  const push = (id: CompanionAction | undefined) => {
    if (id && byId.has(id) && !order.includes(id)) order.push(id);
  };
  for (const id of settings?.actionPrefs ?? []) push(id);
  push(settings ? GOAL_CATEGORY[settings.goal] : undefined);
  for (const c of ACTION_CATALOGUE) push(c.id);
  return order.map((id) => byId.get(id)!);
}

export interface PickedAction extends ActionCardDef {
  category: CompanionAction;
  tone: Tone;
}

/** The home row: the first card of each of the top `count` categories. */
export function homeActions(
  settings: Pick<CompanionLinkSettings, 'actionPrefs' | 'goal'> | null,
  count = 3,
): PickedAction[] {
  return orderedCategories(settings)
    .slice(0, count)
    .map((c) => ({ ...c.cards[0]!, category: c.id, tone: c.tone }));
}

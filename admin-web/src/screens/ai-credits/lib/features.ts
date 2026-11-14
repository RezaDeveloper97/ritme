/** Pure helpers of the «قابلیت‌ها» (feature slots) screen. */

export type FeatureFilter = 'all' | 'problems' | 'no_model' | 'inactive' | 'undeclared';
export const FEATURE_FILTERS: readonly FeatureFilter[] = ['all', 'problems', 'no_model', 'inactive', 'undeclared'];

interface FeatureLike {
  key: string;
  title: Record<string, string>;
  declared: boolean;
  is_active: boolean;
  usable: boolean;
  routes: number;
  model_id: number | null;
}

interface ModelLike {
  id: number;
  capabilities: readonly string[];
  is_active: boolean;
}

/** Search by key or any title translation (case-insensitive), then the status filter. */
export function filterFeatures<F extends FeatureLike>(items: readonly F[], q: string, filter: FeatureFilter): F[] {
  const needle = q.trim().toLowerCase();
  return items.filter((f) => {
    if (needle) {
      const hay = [f.key, ...Object.values(f.title)].join(' ').toLowerCase();
      if (!hay.includes(needle)) return false;
    }
    switch (filter) {
      case 'problems':
        return !f.declared || (f.is_active && !f.usable);
      case 'no_model':
        return f.routes === 0;
      case 'inactive':
        return !f.is_active;
      case 'undeclared':
        return !f.declared;
      default:
        return true;
    }
  });
}

/** The capabilities of `needs` that `model` lacks. */
export function missingCapabilities(model: ModelLike, needs: readonly string[]): string[] {
  return needs.filter((n) => !model.capabilities.includes(n));
}

/**
 * The models a slot's select may offer: active models with every needed capability (the API refuses the rest).
 * The currently assigned model stays in the list even when it no longer qualifies, so the select can show it.
 */
export function modelChoices<M extends ModelLike>(models: readonly M[], needs: readonly string[], current: number | null): M[] {
  return models.filter((m) => m.id === current || (m.is_active && missingCapabilities(m, needs).length === 0));
}

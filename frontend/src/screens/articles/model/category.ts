/**
 * The slice of a next-intl translator this module needs: look a key up and ask
 * whether it exists. Kept structural so the helper stays a pure, testable function.
 */
export interface CategoryTranslator {
  (key: string): string;
  has: (key: string) => boolean;
}

/**
 * Display label for an article category.
 *
 * The API sends the stored value as-is: usually a slug (`nutrition`,
 * `cycle_science`), but the admin form accepts free text. A slug with a key under
 * `articles.categories.*` reads as its translation; anything else (a category an
 * admin typed in words, or a new slug nobody has translated yet) shows as stored
 * instead of raising a missing-message error.
 */
export function articleCategoryLabel(category: string, t: CategoryTranslator): string {
  // A dot would make next-intl walk into a nested key; such a value is never a slug.
  if (category === '' || category.includes('.')) return category;
  const key = `categories.${category}`;
  return t.has(key) ? t(key) : category;
}

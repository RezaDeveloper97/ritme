/** Query keys of the shop catalog (CLAUDE.md §8). */
export const shopKeys = {
  all: ['shop'] as const,
  categories: () => [...shopKeys.all, 'categories'] as const,
  products: (category: string, page: number) => [...shopKeys.all, 'products', category, page] as const,
};

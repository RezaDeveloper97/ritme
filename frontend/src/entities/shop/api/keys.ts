/** Query keys of the shop catalog (CLAUDE.md §8). */
export const shopKeys = {
  all: ['shop'] as const,
  categories: () => [...shopKeys.all, 'categories'] as const,
  products: (category: string, page: number) => [...shopKeys.all, 'products', category, page] as const,
  catalogHome: (root: string) => [...shopKeys.all, 'catalog-home', root] as const, // CB-SHOP-06
  catalogList: (params: object) => [...shopKeys.all, 'catalog-list', params] as const, // CB-SHOP-07
  catalogProduct: (id: number) => [...shopKeys.all, 'catalog-product', id] as const,
  catalogReviews: (id: number) => [...shopKeys.all, 'catalog-reviews', id] as const,
};

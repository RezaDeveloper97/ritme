// Public API of the `shop` entity (bloom B-N10-03) — categories and link-out products. Import only from here.

export { shopKeys } from './api/keys';
export { shopCategoriesSchema, shopProductsSchema, type ShopCategory, type ShopProduct } from './api/schema';
export { countShopClick, useShopCategories, useShopProducts } from './api/queries';

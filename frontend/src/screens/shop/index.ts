// CB-SHOP-06: `/shop` now renders the catalog home (baby + beauty), which falls back to bloom's link-out categories
// page (`ShopLinkOutPage`, B-N10-03) while the internal catalog is empty.
export { ShopHomePage as ShopPage } from './ui/ShopHomePage';
export { ShopPage as ShopLinkOutPage } from './ui/ShopPage';
// CB-SHOP-07: `/shop/[category]` shows the filtered catalog list, or bloom's link-out list for a non-catalog slug.
export { ShopCategoryGate as ShopCategoryPage } from './ui/ShopCategoryGate';
export { ShopProductPage } from './ui/ShopProductPage';

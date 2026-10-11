// CB-SHOP-06: `/shop` now renders the catalog home (baby + beauty), which falls back to bloom's link-out categories
// page (`ShopLinkOutPage`, B-N10-03) while the internal catalog is empty.
export { ShopHomePage as ShopPage } from './ui/ShopHomePage';
export { ShopPage as ShopLinkOutPage } from './ui/ShopPage';
export { ShopCategoryPage } from './ui/ShopCategoryPage';

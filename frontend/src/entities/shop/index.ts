// Public API of the `shop` entity (bloom B-N10-03 link-out + CB-SHOP-06 catalog home). Import only from here.

export { shopKeys } from './api/keys';
export {
  shopCatalogHomeSchema,
  shopCategoriesSchema,
  shopProductsSchema,
  shopCatalogListSchema,
  shopCatalogProductSchema,
  SHOP_SORTS,
  type ShopCatalogCard,
  type ShopCatalogFacet,
  type ShopCatalogList,
  type ShopCatalogProduct,
  type ShopCatalogReview,
  type ShopCatalogVariant,
  type ShopSort,
  type ShopCatalogCategory,
  type ShopCatalogHome,
  type ShopCategory,
  type ShopProduct,
} from './api/schema';
export {
  countShopClick,
  useShopCatalogHome,
  useShopCatalogProduct,
  useShopCatalogProducts,
  useShopCatalogReviews,
  useShopCategories,
  useShopProducts,
  type PublicOpts,
  type ShopListParams,
} from './api/queries';
export { fetchPublicShopHome, fetchPublicShopList, fetchPublicShopProduct } from './api/public';
export { formatToman } from './model/price';
export {
  DEFAULT_LIST_STATE,
  NO_LIST_FILTERS,
  parseShopListRecord,
  parseShopListState,
  shopListParams,
  shopListSearch,
  type ShopListFilters,
  type ShopListState,
} from './model/list-url';

// CB-SHOP-09: layette checklist + discreet period-supply reminder.
export {
  CHECKLIST_GROUPS,
  CHECKLIST_STATES,
  type ChecklistGroupKey,
  type ChecklistState,
  type ShopChecklist,
  type ShopChecklistGroup,
  type ShopChecklistItem,
  type SupplyReminder,
} from './api/checklist-schema';
export {
  useAddCustomItem,
  useDeleteCustomItem,
  useSaveSupplyReminder,
  useSetChecklistItemState,
  useShopChecklist,
  useSupplyReminder,
  useUpdateCustomItem,
} from './api/checklist';

// CB-SHOP-08: cart, addresses, COD checkout, orders, returns.
export {
  RETURN_REASONS,
  REVIEW_STATUSES,
  type ReturnReason,
  type ShopAddress,
  type ShopAddressInput,
  type ShopCart,
  type ShopCartLine,
  type ShopCheckout,
  type ShopOrder,
  type ShopOrderItem,
  type ShopOrderSummary,
  type ShopOwnReview,
} from './api/order-schema';
export {
  shopErrorCode,
  shopCodeKey,
  shopErrorKey,
  useAddToCart,
  useApplyDiscount,
  useCancelOrder,
  useCreateReview,
  useDeleteAddress,
  usePlaceOrder,
  useRemoveCartItem,
  useRemoveDiscount,
  useRequestReturn,
  useSaveAddress,
  useSetCartQuantity,
  useShopCart,
  useShopCheckout,
  useShopOrder,
  useShopOrders,
} from './api/order';
export { useShopPrice } from './model/use-price';
export { useSlotText } from './model/use-slot';

// CB-SHOP-11: helpers shared by the app and the desktop checkout.
export { checkoutBlockerKey, maskPhone, useIdempotencyKey } from './model/checkout';

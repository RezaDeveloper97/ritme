---
id: L6-06
title: Admin: products, categories, brands, orders, reviews, low-stock
milestone: L6
type: admin
status: done
depends_on: [L6-05,L2-03,L4-05]
parallel_group: L6-D
touches: [app/Filament/Resources/Shop,app/Filament/Widgets/Shop,resources/views/admin/invoice.blade.php,tests/Feature/Admin/ShopAdminTest.php]
skills: []
verify: composer verify
---

# L6-06 — Admin: products, categories, brands, orders, reviews, low-stock

## Scope
- Product resource (tabs: general, pricing (Money input in Toman), inventory/variants repeater, specs/size chart
  editors, gallery MediaPicker, categories, SEO via `SeoFields`), categories tree (drag-sort), brands.
- Orders: list with filters, status transitions (state machine with allowed transitions only), printable invoice
  (Blade, Persian), notes, CSV export; reviews moderation.
- Widgets: today's orders, revenue (7/30 days), low-stock list. `shop-manager` role.

## Acceptance
- Order lifecycle from admin works; invalid transitions rejected; tests green.

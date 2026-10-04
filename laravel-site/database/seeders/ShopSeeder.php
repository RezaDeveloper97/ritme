<?php

declare(strict_types=1);

namespace Database\Seeders;

use App\Domain\Shop\Catalog\Actions\SyncCrossSells;
use App\Domain\Shop\Catalog\Actions\SyncProductCategories;
use App\Domain\Shop\Catalog\Enums\ReviewStatus;
use App\Domain\Shop\Catalog\Models\Brand;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Support\Money\Money;
use Illuminate\Database\Seeder;

/**
 * DEMO shop content from the design (shop.html, shop-list.html, shop-product.html): the two departments with their
 * category tiles, four brands (the design's «sellers» — single-seller shop, so they become brands), and the eleven
 * sample products incl. «بادی آستین‌بلند نخی · ۳ عدد» (`long-sleeve-cotton-bodysuit-3`) with colour × size variants,
 * specs and the size chart. Not called from DatabaseSeeder — demo data never reaches production by accident:
 *
 *     php artisan db:seed --class=ShopSeeder
 *
 * Every product is `is_demo` and says so in its description; prices are the design's sample prices (stored as integer
 * rials). The design's ratings («۴٫۷ (۲۱۴)») and sales («۱٬۸۰۰+ خرید») are NOT seeded: no product has an invented
 * rating or sales count. The three sample reviews are `is_demo` (shown labelled, never counted). Idempotent: rows are
 * matched by slug and existing products are left untouched. Copy follows the content red lines.
 */
final class ShopSeeder extends Seeder
{
    public const DEMO_PRODUCT_SLUG = 'long-sleeve-cotton-bodysuit-3';

    public const DEMO_CATEGORY_SLUG = 'baby-clothes';

    public const DEMO_NOTE = '<p><small>این محصول و قیمتش نمونه نمایشی فروشگاه است.</small></p>';

    /** Colour swatches of shop-product.html (swatch data, not CSS). */
    private const COLORS = [
        ['شیری', '#F6EFE6'],
        ['صورتی', '#F4D6DE'],
        ['نعنایی', '#CDEBE4'],
        ['یاسی', '#D9D4F2'],
    ];

    private const BABY_SIZES = ['۰-۳ ماه', '۳-۶ ماه', '۶-۹ ماه', '۹-۱۲ ماه', '۱۲-۱۸ ماه'];

    public function run(): void
    {
        $categories = $this->categories();
        $brands = $this->brands();
        $sync = app(SyncProductCategories::class);

        $products = [];
        foreach ($this->products() as $i => $data) {
            $existing = Product::query()->where('slug', $data['slug'])->first();
            if ($existing !== null) {
                $products[$data['slug']] = $existing;

                continue;
            }

            $product = new Product([
                'title' => $data['title'],
                'slug' => $data['slug'],
                'sku' => 'DEMO-'.str_pad((string) ($i + 1), 3, '0', STR_PAD_LEFT),
                'brand_id' => $brands[$data['brand']]->id,
                'short_description' => $data['short'],
                'description' => '<p>'.$data['description'].'</p>'.self::DEMO_NOTE,
                'specs' => $data['specs'] ?? null,
                'size_chart' => $data['chart'] ?? null,
                'badge' => $data['badge'] ?? null,
                'price' => Money::fromToman($data['price']),
                'compare_at_price' => isset($data['compare']) ? Money::fromToman($data['compare']) : null,
                'stock_qty' => $data['stock'] ?? 12,
                'weight_grams' => $data['weight'] ?? null,
                'illustration' => $data['illustration'],
                'life_stages' => $data['stages'],
                'is_published' => true,
                'is_featured' => $data['featured'] ?? false,
                'is_demo' => true,
                'sort_order' => $i,
            ]);
            $product->save();

            $sync->handle($product, array_map(static fn (string $slug): int => $categories[$slug]->id, $data['categories']));

            foreach ($data['variants'] ?? [] as $position => $variant) {
                ProductVariant::query()->create(['product_id' => $product->id, 'sort_order' => $position, ...$variant]);
            }

            foreach ($data['reviews'] ?? [] as $review) {
                ProductReview::query()->create(['product_id' => $product->id, 'status' => ReviewStatus::Approved, 'is_demo' => true, ...$review]);
            }

            $products[$data['slug']] = $product;
        }

        $crossSells = app(SyncCrossSells::class);
        foreach ($this->products() as $data) {
            $product = $products[$data['slug']];
            if (($data['bought_with'] ?? []) !== [] && $product->wasRecentlyCreated) {
                $crossSells->handle($product, array_map(static fn (string $slug): int => $products[$slug]->id, $data['bought_with']));
            }
        }
    }

    /**
     * The two departments and their tiles (shop.html), with the design's subnav names where they differ.
     *
     * @return array<string, Category>
     */
    private function categories(): array
    {
        $tree = [
            'baby' => ['سیسمونی و نوزاد', 'لباس، خواب، تغذیه و بیرون رفتن؛ همراه با لیست سیسمونی قابل تنظیم.', null, [
                'baby-clothes' => ['لباس نوزاد', 'سایزها بر اساس سن و وزن نوزاد است؛ جدول سایز هر محصول را قبل از خرید ببین.', null],
                'sleepwear' => ['سرهمی و خواب', null, null],
                'feeding' => ['تغذیه', null, null],
                'strollers' => ['کالسکه و بیرون', null, 'category-stroller'],
                'nursery' => ['تخت و اتاق', null, 'category-crib'],
                'blankets' => ['پتو و ملحفه', null, null],
                'baby-bath' => ['حمام نوزاد', null, 'category-bath'],
                'socks-hats' => ['جوراب و کلاه', null, null],
            ]],
            'beauty' => ['آرایشی و بهداشتی', 'بهداشت بانوان، مراقبت پوست و مو، با شناسه IRC و ترکیبات کامل هر محصول.', null, [
                'feminine-hygiene' => ['بهداشت بانوان', null, null],
                'skincare' => ['مراقبت پوست', null, null],
                'sunscreen' => ['ضدآفتاب', null, null],
                'makeup' => ['آرایش', null, null],
                'hair' => ['مو', null, null],
                'body-bath' => ['بدن و حمام', null, null],
                'menstrual-cups' => ['کاپ قاعدگی', null, null],
                'breastfeeding' => ['مادر و شیردهی', null, null],
            ]],
        ];

        $categories = [];
        $rootOrder = 0;
        foreach ($tree as $slug => [$name, $intro, $illustration, $children]) {
            $root = Category::query()->firstOrCreate(['slug' => $slug], [
                'name' => $name, 'intro' => $intro, 'illustration' => $illustration, 'sort_order' => ++$rootOrder,
            ]);
            $categories[$slug] = $root;

            $order = 0;
            foreach ($children as $childSlug => [$childName, $childIntro, $childIllustration]) {
                $categories[$childSlug] = Category::query()->firstOrCreate(['slug' => $childSlug], [
                    'parent_id' => $root->id, 'name' => $childName, 'intro' => $childIntro,
                    'illustration' => $childIllustration, 'sort_order' => ++$order,
                ]);
            }
        }

        return $categories;
    }

    /**
     * @return array<string, Brand>
     */
    private function brands(): array
    {
        $rows = [
            'panberiz' => 'پوشاک پنبه‌ریز',
            'mahno' => 'خانه سیسمونی ماه‌نو',
            'sepideh' => 'عطر و آرایش سپیده',
            'banoo' => 'بهداشتی بانو',
        ];

        $brands = [];
        $order = 0;
        foreach ($rows as $slug => $name) {
            $brands[$slug] = Brand::query()->firstOrCreate(['slug' => $slug], ['name' => $name, 'sort_order' => ++$order]);
        }

        return $brands;
    }

    /**
     * @param  list<string>  $sizes
     * @param  list<array{0: string, 1: string}>  $colors
     * @param  array<string, int>  $soldOut  "size|colour" => 0 stock
     * @return list<array<string, mixed>>
     */
    private function variants(string $skuPrefix, array $sizes, array $colors = [], array $soldOut = []): array
    {
        $rows = [];
        $n = 0;
        foreach ($colors === [] ? [[null, null]] : $colors as [$color, $hex]) {
            foreach ($sizes as $size) {
                $rows[] = [
                    'sku' => $skuPrefix.'-'.str_pad((string) ++$n, 2, '0', STR_PAD_LEFT),
                    'size' => $size,
                    'color' => $color,
                    'color_hex' => $hex,
                    'stock_qty' => $soldOut[$size.'|'.$color] ?? 6,
                ];
            }
        }

        return $rows;
    }

    /**
     * The design's sample products (prices in toman as shown; stored ×10 as rials).
     *
     * @return list<array<string, mixed>>
     */
    private function products(): array
    {
        $babyChart = [
            'columns' => ['سایز', 'سن', 'قد (سانت)', 'وزن (کیلو)'],
            'rows' => [
                ['نوزاد', 'تا ۱ ماه', 'تا ۵۶', 'تا ۴'],
                ['۰-۳ ماه', '۰ تا ۳ ماه', '۵۶ تا ۶۱', '۴ تا ۵'],
                ['۳-۶ ماه', '۳ تا ۶ ماه', '۶۱ تا ۶۷', '۵ تا ۷'],
                ['۶-۹ ماه', '۶ تا ۹ ماه', '۶۷ تا ۷۲', '۷ تا ۹'],
                ['۹-۱۲ ماه', '۹ تا ۱۲ ماه', '۷۲ تا ۷۶', '۹ تا ۱۰'],
            ],
        ];

        return [
            [
                'slug' => self::DEMO_PRODUCT_SLUG,
                'title' => 'بادی آستین‌بلند نخی · ۳ عدد',
                'brand' => 'panberiz',
                'categories' => ['baby-clothes'],
                'price' => 485_000,
                'badge' => 'پنبه ۱۰۰٪',
                'illustration' => 'product-bodysuit',
                'stages' => ['pregnancy', 'postpartum'],
                'featured' => true,
                'weight' => 240,
                'short' => 'بسته سه‌تایی بادی آستین‌بلند از پارچه پنبه‌ای نرم، با دکمه فشاری برای عوض کردن ساده پوشک.',
                'description' => 'پارچه پنبه‌ای نرم و قابل شست‌وشو در ماشین. دکمه‌های فشاری بدون نیکل در پایین بادی، عوض کردن پوشک را ساده می‌کند. سایز را با جدول سایز و قد و وزن نوزاد انتخاب کن.',
                'specs' => [
                    ['label' => 'جنس', 'value' => 'پنبه ۱۰۰٪'],
                    ['label' => 'تعداد', 'value' => '۳ عدد در بسته'],
                    ['label' => 'دکمه', 'value' => 'فشاری پلاستیکی، بدون نیکل'],
                    ['label' => 'شست‌وشو', 'value' => 'ماشین با ۳۰ درجه'],
                    ['label' => 'ساخت', 'value' => '[کشور سازنده]'],
                ],
                'chart' => $babyChart,
                'variants' => $this->variants('DEMO-BODY', self::BABY_SIZES, self::COLORS, ['۱۲-۱۸ ماه|شیری' => 0]),
                'reviews' => [
                    ['author_name' => 'مادر پرنیا', 'rating' => 5, 'variant_label' => 'سایز ۳-۶ ماه', 'body' => 'پارچه نرم است و بعد از چند بار شستن جمع نشد.'],
                    ['author_name' => 'مادر سام', 'rating' => 4, 'variant_label' => 'سایز ۰-۳ ماه', 'body' => 'سایز کمی بزرگ است؛ یک سایز کوچک‌تر بگیرید.'],
                    ['author_name' => 'مادر آوا', 'rating' => 5, 'variant_label' => 'سایز ۳-۶ ماه', 'body' => 'دکمه‌ها راحت باز و بسته می‌شوند و پوشک عوض کردن ساده است.'],
                ],
                'bought_with' => ['baby-socks-5-pairs', 'baby-hat-and-mittens', 'muslin-baby-blanket', 'snap-sleepsuit', 'baby-bottle-240ml'],
            ],
            [
                'slug' => 'snap-sleepsuit',
                'title' => 'سرهمی خواب دکمه‌دار',
                'brand' => 'panberiz',
                'categories' => ['sleepwear', 'baby-clothes'],
                'price' => 390_000,
                'compare' => 460_000,
                'illustration' => 'product-sleepsuit',
                'stages' => ['postpartum'],
                'short' => 'سرهمی خواب پنبه‌ای با دکمه‌های جلو برای پوشاندن و درآوردن راحت.',
                'description' => 'سرهمی خواب از پارچه پنبه‌ای با دکمه‌های فشاری در جلو. برای خواب شب و روز نوزاد.',
                'specs' => [['label' => 'جنس', 'value' => 'پنبه ۱۰۰٪'], ['label' => 'شست‌وشو', 'value' => 'ماشین با ۳۰ درجه']],
                'chart' => $babyChart,
                'variants' => $this->variants('DEMO-SLEEP', self::BABY_SIZES),
            ],
            [
                'slug' => 'baby-hat-and-mittens',
                'title' => 'کلاه و دستکش نوزادی',
                'brand' => 'mahno',
                'categories' => ['socks-hats', 'baby-clothes'],
                'price' => 160_000,
                'illustration' => 'product-hat',
                'stages' => ['postpartum'],
                'short' => 'ست کلاه و دستکش نخی برای روزهای اول نوزاد.',
                'description' => 'کلاه و دستکش نخی نرم برای نوزاد؛ دستکش‌ها کش ملایم دارند.',
            ],
            [
                'slug' => 'baby-socks-5-pairs',
                'title' => 'جوراب نوزادی · ۵ جفت',
                'brand' => 'panberiz',
                'categories' => ['socks-hats', 'baby-clothes'],
                'price' => 140_000,
                'illustration' => 'product-socks',
                'stages' => ['postpartum'],
                'short' => 'پنج جفت جوراب نخی نوزادی در رنگ‌های ملایم.',
                'description' => 'جوراب نخی نوزادی با کش نرم که روی پا جا نمی‌اندازد.',
                'variants' => $this->variants('DEMO-SOCK', ['۰-۳ ماه', '۳-۶ ماه', '۶-۱۲ ماه']),
            ],
            [
                'slug' => 'baby-bottle-240ml',
                'title' => 'شیشه شیر ۲۴۰ میلی‌لیتری',
                'brand' => 'mahno',
                'categories' => ['feeding'],
                'price' => 290_000,
                'illustration' => 'product-bottle',
                'stages' => ['postpartum'],
                'short' => 'شیشه شیر ۲۴۰ میلی‌لیتری با درجه‌بندی روی بدنه.',
                'description' => 'شیشه شیر با درجه‌بندی خوانا و سر شیشه قابل تعویض. قبل از اولین استفاده طبق راهنمای بسته‌بندی بشویید.',
                'specs' => [['label' => 'حجم', 'value' => '۲۴۰ میلی‌لیتر']],
            ],
            [
                'slug' => 'muslin-baby-blanket',
                'title' => 'پتوی نوزادی ململ',
                'brand' => 'panberiz',
                'categories' => ['blankets', 'baby-clothes'],
                'price' => 320_000,
                'illustration' => 'product-blanket',
                'stages' => ['pregnancy', 'postpartum'],
                'short' => 'پتوی سبک ململ برای قنداق و روانداز نوزاد.',
                'description' => 'پتوی ململ سبک و قابل شست‌وشو؛ برای قنداق کردن، روانداز یا سایه‌بان کالسکه.',
                'specs' => [['label' => 'جنس', 'value' => 'ململ پنبه‌ای']],
            ],
            [
                'slug' => 'tinted-free-sunscreen-spf50',
                'title' => 'ضدآفتاب بی‌رنگ SPF50',
                'brand' => 'sepideh',
                'categories' => ['sunscreen', 'skincare'],
                'price' => 550_000,
                'illustration' => 'product-sunscreen',
                'stages' => ['cycle', 'pregnancy', 'menopause'],
                'short' => 'ضدآفتاب بی‌رنگ با SPF50 برای استفاده روزانه.',
                'description' => 'ضدآفتاب بی‌رنگ برای استفاده روزانه. شناسه IRC و فهرست کامل ترکیبات روی بسته‌بندی درج شده است. در بارداری و شیردهی قبل از استفاده از محصولات جدید با پزشک یا ماما مشورت کن.',
                'specs' => [['label' => 'شناسه IRC', 'value' => '[شناسه IRC]'], ['label' => 'حجم', 'value' => '۵۰ میلی‌لیتر']],
            ],
            [
                'slug' => 'hyaluronic-hydrating-serum',
                'title' => 'سرم آبرسان هیالورونیک',
                'brand' => 'sepideh',
                'categories' => ['skincare'],
                'price' => 420_000,
                'compare' => 490_000,
                'illustration' => 'product-serum',
                'stages' => ['cycle', 'menopause'],
                'short' => 'سرم آبرسان با هیالورونیک اسید برای روتین روزانه پوست.',
                'description' => 'سرم سبک آبرسان برای روتین صبح و شب. شناسه IRC و ترکیبات کامل روی بسته‌بندی درج شده است.',
                'specs' => [['label' => 'شناسه IRC', 'value' => '[شناسه IRC]'], ['label' => 'حجم', 'value' => '۳۰ میلی‌لیتر']],
            ],
            [
                'slug' => 'daily-pads-20',
                'title' => 'نوار بهداشتی روزانه · ۲۰ عدد',
                'brand' => 'banoo',
                'categories' => ['feminine-hygiene'],
                'price' => 98_000,
                'illustration' => 'product-pad',
                'stages' => ['cycle', 'teen'],
                'short' => 'بسته ۲۰ عددی نوار بهداشتی روزانه، نازک و بی‌بو.',
                'description' => 'نوار بهداشتی روزانه نازک با بسته‌بندی تکی. بسته‌بندی ساده بدون نام محصول روی جعبه در مرحله پرداخت قابل انتخاب است.',
            ],
            [
                'slug' => 'silicone-menstrual-cup',
                'title' => 'کاپ قاعدگی سیلیکونی',
                'brand' => 'banoo',
                'categories' => ['menstrual-cups', 'feminine-hygiene'],
                'price' => 360_000,
                'illustration' => 'product-cup',
                'stages' => ['cycle'],
                'short' => 'کاپ قاعدگی قابل استفاده دوباره از سیلیکون نرم، در دو سایز.',
                'description' => 'کاپ قاعدگی سیلیکونی قابل استفاده دوباره. راهنمای انتخاب سایز، استفاده و ضدعفونی داخل بسته است.',
                'variants' => $this->variants('DEMO-CUP', ['کوچک', 'بزرگ']),
            ],
            [
                'slug' => 'long-lasting-matte-lipstick',
                'title' => 'رژلب مات ماندگار',
                'brand' => 'sepideh',
                'categories' => ['makeup'],
                'price' => 310_000,
                'illustration' => 'product-lipstick',
                'stages' => ['cycle'],
                'short' => 'رژلب مات با بافت سبک.',
                'description' => 'رژلب مات با بافت سبک. شناسه IRC و فهرست ترکیبات روی بسته‌بندی درج شده است.',
                'stock' => 0,
            ],
        ];
    }
}

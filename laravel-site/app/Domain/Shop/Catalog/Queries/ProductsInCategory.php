<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Queries;

use App\Domain\Search\Support\SearchTerms;
use App\Domain\Shop\Catalog\Data\CategoryTree;
use App\Domain\Shop\Catalog\Data\ProductCriteria;
use App\Domain\Shop\Catalog\Data\ProductFacets;
use App\Domain\Shop\Catalog\Data\ProductPage;
use App\Domain\Shop\Catalog\Enums\ProductSort;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Brand;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Support\Money\Money;
use App\Support\Text\PersianDigits;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Query\Builder as QueryBuilder;

/**
 * Published products matching ProductCriteria (shop listing L6-02, shop search). A category includes its visible
 * subcategories (CategoryTree). Filters run in SQL and work on SQLite + MySQL: brands, sizes/colours of active
 * variants, price bounds, in stock, life stage (JSON contains) and text tokens (portable LIKE over normalised title,
 * short description and brand name). Sorts as ProductSort, sold-out products last, no paid placement. Facets are
 * computed over the category's products before the other filters (ProductFacets).
 */
final class ProductsInCategory
{
    /** LIKE escape character; `\` is not portable (MySQL string literals treat it as an escape). */
    private const ESCAPE = '!';

    public function __construct(private readonly ProductCriteria $criteria, private readonly CategoryTree $tree) {}

    public function get(): ProductPage
    {
        $c = $this->criteria;
        $table = (new Product)->getTable();

        $filtered = $this->filtered();
        $total = (clone $filtered)->count();

        $ids = $this->sorted($filtered)
            ->forPage($c->page, $c->perPage)
            ->pluck("{$table}.id")
            ->map(intval(...))
            ->values()
            ->all();

        return new ProductPage(ProductCards::load($ids), $total, $c->page, $c->perPage, $this->facets());
    }

    /**
     * Published products of the category subtree (or all published products).
     *
     * @return Builder<Product>
     */
    private function base(): Builder
    {
        $table = (new Product)->getTable();
        $query = Product::query()->published();

        if ($this->criteria->categoryId !== null) {
            $categoryIds = $this->tree->descendantIds($this->criteria->categoryId) ?: [$this->criteria->categoryId];
            $query->whereExists(static function (QueryBuilder $sub) use ($table, $categoryIds): void {
                $sub->selectRaw('1')
                    ->from('shop_category_product')
                    ->whereColumn('shop_category_product.product_id', "{$table}.id")
                    ->whereIn('shop_category_product.category_id', $categoryIds);
            });
        }

        return $query;
    }

    /**
     * @return Builder<Product>
     */
    private function filtered(): Builder
    {
        $c = $this->criteria;
        $table = (new Product)->getTable();
        $query = $this->base();

        if ($c->brandIds !== []) {
            $query->whereIn("{$table}.brand_id", $c->brandIds);
        }
        if ($c->sizes !== [] || $c->colors !== []) {
            $variants = (new ProductVariant)->getTable();
            $query->whereExists(static function (QueryBuilder $sub) use ($c, $table, $variants): void {
                $sub->selectRaw('1')
                    ->from($variants)
                    ->whereColumn("{$variants}.product_id", "{$table}.id")
                    ->where("{$variants}.is_active", true)
                    ->when($c->sizes !== [], static fn (QueryBuilder $q) => $q->whereIn("{$variants}.size", $c->sizes))
                    ->when($c->colors !== [], static fn (QueryBuilder $q) => $q->whereIn("{$variants}.color", $c->colors))
                    ->when($c->inStockOnly, static fn (QueryBuilder $q) => $q->where("{$variants}.stock_qty", '>', 0));
            });
        }
        if ($c->minPrice !== null) {
            $query->where("{$table}.price", '>=', $c->minPrice->rial);
        }
        if ($c->maxPrice !== null) {
            $query->where("{$table}.price", '<=', $c->maxPrice->rial);
        }
        if ($c->inStockOnly) {
            $query->where("{$table}.stock_status", StockStatus::InStock->value);
        }
        if ($c->lifeStage !== null) {
            $query->whereJsonContains("{$table}.life_stages", $c->lifeStage->value);
        }

        $this->matchText($query, $table);

        return $query;
    }

    /**
     * @param  Builder<Product>  $query
     */
    private function matchText(Builder $query, string $table): void
    {
        if ($this->criteria->tokens === []) {
            return;
        }

        $columns = array_map($this->normalized(...), ["{$table}.title", "{$table}.short_description"]);
        $brands = (new Brand)->getTable();
        $brandName = $this->normalized("{$brands}.name");

        foreach ($this->criteria->tokens as $token) {
            $patterns = $this->patterns($token);
            $query->where(static function (Builder $q) use ($columns, $patterns, $table, $brands, $brandName): void {
                foreach ($columns as $column) {
                    foreach ($patterns as $pattern) {
                        $q->orWhereRaw("{$column} LIKE ? ESCAPE '".self::ESCAPE."'", [$pattern]);
                    }
                }
                $q->orWhereIn("{$table}.brand_id", static function (QueryBuilder $sub) use ($brands, $brandName, $patterns): void {
                    $sub->select("{$brands}.id")->from($brands)->where(static function (QueryBuilder $names) use ($brandName, $patterns): void {
                        foreach ($patterns as $pattern) {
                            $names->orWhereRaw("{$brandName} LIKE ? ESCAPE '".self::ESCAPE."'", [$pattern]);
                        }
                    });
                });
            });
        }
    }

    /**
     * @param  Builder<Product>  $query
     * @return Builder<Product>
     */
    private function sorted(Builder $query): Builder
    {
        $table = (new Product)->getTable();
        $query->orderByRaw(ProductCards::outOfStockLast($table));

        match ($this->criteria->sort) {
            ProductSort::Newest => $query->orderByDesc("{$table}.created_at"),
            ProductSort::Cheapest => $query->orderBy("{$table}.price"),
            ProductSort::TopRated => $query->orderByDesc("{$table}.rating_avg")->orderByDesc("{$table}.rating_count"),
            ProductSort::BestSelling => $query->orderByDesc("{$table}.sales_count")->orderByDesc("{$table}.is_featured")->orderBy("{$table}.sort_order"),
        };

        return $query->orderByDesc("{$table}.id");
    }

    private function facets(): ProductFacets
    {
        $table = (new Product)->getTable();
        $base = $this->base();
        $ids = (clone $base)->select("{$table}.id");

        $prices = (clone $base)->toBase()->selectRaw("MIN({$table}.price) as min_price, MAX({$table}.price) as max_price")->first();

        $variants = ProductVariant::query()
            ->whereIn('product_id', clone $ids)
            ->where('is_active', true)
            ->orderBy('sort_order')
            ->orderBy('id')
            ->get(['size', 'color', 'color_hex']);
        $sizes = [];
        $colors = [];
        foreach ($variants as $variant) {
            if ($variant->size !== null) {
                $sizes[$variant->size] = true;
            }
            if ($variant->color !== null && ! isset($colors[$variant->color])) {
                $colors[$variant->color] = ['name' => $variant->color, 'hex' => $variant->color_hex];
            }
        }

        $brandCounts = (clone $base)->toBase()
            ->whereNotNull("{$table}.brand_id")
            ->groupBy("{$table}.brand_id")
            ->selectRaw("{$table}.brand_id as brand_id, COUNT(*) as aggregate")
            ->pluck('aggregate', 'brand_id');
        $brands = Brand::query()->active()->whereIn('id', $brandCounts->keys()->map(intval(...))->all())
            ->orderBy('sort_order')->orderBy('name')
            ->get(['id', 'name', 'slug'])
            ->map(static fn (Brand $b): array => ['id' => $b->id, 'name' => $b->name, 'slug' => $b->slug, 'count' => (int) $brandCounts[$b->id]])
            ->values()
            ->all();

        return new ProductFacets(
            categoryCounts: $this->categoryCounts(clone $ids),
            sizes: array_map(strval(...), array_keys($sizes)),
            colors: array_values($colors),
            brands: $brands,
            minPrice: isset($prices->min_price) ? Money::fromRial((int) $prices->min_price) : null,
            maxPrice: isset($prices->max_price) ? Money::fromRial((int) $prices->max_price) : null,
        );
    }

    /**
     * Products per direct subcategory (counting the whole subtree; a product in two subcategories counts in both).
     *
     * @param  Builder<Product>  $ids
     * @return array<int, int>
     */
    private function categoryCounts(Builder $ids): array
    {
        $children = $this->tree->children($this->criteria->categoryId);
        if ($children === []) {
            return [];
        }

        $pairs = Product::query()->getConnection()->table('shop_category_product')
            ->whereIn('product_id', $ids)
            ->get(['product_id', 'category_id']);

        $counts = [];
        foreach ($children as $child) {
            $subtree = array_flip($this->tree->descendantIds($child->id));
            $products = [];
            foreach ($pairs as $pair) {
                if (isset($subtree[(int) $pair->category_id])) {
                    $products[(int) $pair->product_id] = true;
                }
            }
            $counts[$child->id] = count($products);
        }

        return $counts;
    }

    private function normalized(string $column): string
    {
        $sql = "LOWER(COALESCE({$column}, ''))";
        foreach (SearchTerms::CHARACTER_MAP as $from => $to) {
            $sql = "REPLACE({$sql}, '{$from}', '{$to}')";
        }

        return $sql;
    }

    /**
     * @return list<string>
     */
    private function patterns(string $token): array
    {
        $variants = array_values(array_unique([$token, PersianDigits::toPersian($token)]));

        return array_map(static fn (string $variant): string => '%'.strtr($variant, [
            self::ESCAPE => self::ESCAPE.self::ESCAPE, '%' => self::ESCAPE.'%', '_' => self::ESCAPE.'_',
        ]).'%', $variants);
    }
}

<?php

declare(strict_types=1);

namespace App\Http\Controllers\Shop;

use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Media\Data\MediaData;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\PageGraph;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Domain\Shop\Cart\Actions\AddToCart;
use App\Domain\Shop\Cart\Actions\RemoveFromCart;
use App\Domain\Shop\Cart\Actions\ResolveCart;
use App\Domain\Shop\Cart\Actions\UpdateCartLine;
use App\Domain\Shop\Cart\Data\Cart;
use App\Domain\Shop\Cart\Data\CartChange;
use App\Domain\Shop\Cart\Data\CartItemData;
use App\Domain\Shop\Cart\Data\CartNotice;
use App\Domain\Shop\Cart\Data\CartSummary;
use App\Domain\Shop\Cart\Exceptions\CartException;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Data\ProductCardData;
use App\Domain\Shop\Catalog\Support\ShopUrls;
use App\Support\Text\PersianDigits;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Http\Response;
use Symfony\Component\HttpFoundation\Cookie;
use Symfony\Component\HttpFoundation\Response as BaseResponse;

/**
 * `/shop/cart` (L6-04, design/html/shop-cart.html) and the cart mutations.
 *
 * Works without JS: every change is a plain POST form answered with a redirect (PRG) — to the cart page with a flash
 * status, or back to the product page with a flash error when an add fails. With `Accept: application/json` (the lazy
 * `cart` module) the same endpoints answer JSON {ok, message, count} (422 on a refused change). The session only holds
 * ids + quantities (Shop\Cart); prices and stock are re-read live on every request (ResolveCart / the actions), never
 * taken from the client.
 *
 * The page is noindex (SeoManager::NOINDEX_ROUTES), `no-store` and never page-cached (transactional route). Every cart
 * response writes the plain cookie `ritme_cart_count` (not HttpOnly, not encrypted) that the header badge reads.
 */
final class CartController
{
    public const COUNT_COOKIE = 'ritme_cart_count';

    public const MUTATIONS_PER_MINUTE = 60;

    public const LINE_PATTERN = 'p[0-9]{1,10}(-v[0-9]{1,10})?';

    private const SUGGESTIONS = 5;

    public function __construct(
        private readonly ResolveCart $resolve,
        private readonly ProductRepository $products,
        private readonly MediaRepository $media,
        private readonly ShopUrls $urls,
        private readonly ViewFactory $views,
        private readonly Config $config,
    ) {}

    public function show(Request $request, SeoManager $seo, SchemaGraph $graph): BaseResponse
    {
        $cart = $this->resolve->handle();
        $suggestions = $this->suggestions($cart);

        $seo->rawTitle(self::text('shop.cart.seo_title'))->description(self::text('shop.cart.seo_description'))->noindex();
        $graph->pageName(self::text('shop.cart.title'))->breadcrumbs(
            new BreadcrumbItem(PageGraph::HOME_LABEL, SchemaIds::root((string) $this->config->get('app.url'))),
            new BreadcrumbItem(self::text('shop.name'), $this->urls->absolute(route('shop.index'))),
            new BreadcrumbItem(self::text('shop.cart.title')),
        );

        $mediaIds = array_merge(
            array_map(static fn (CartItemData $item): ?int => $item->coverMediaId, $cart->items),
            array_map(static fn (ProductCardData $c): ?int => $c->coverMediaId, $suggestions),
        );

        $response = new Response($this->views->make('pages.shop.cart', [
            'cart' => $cart,
            'notices' => array_map(self::message(...), $cart->notices),
            'itemNotices' => $this->itemNotices($cart),
            'status' => $this->flash($request, 'cart_status'),
            'error' => $this->flash($request, 'cart_error'),
            'media' => $this->coverMedia($mediaIds),
            'suggestions' => $suggestions,
            'navRoute' => 'shop.index',
        ])->render(), 200, ['Content-Type' => 'text/html; charset=UTF-8', 'Cache-Control' => 'no-store, private']);

        return $this->withCount($response, $cart->count);
    }

    public function add(Request $request, AddToCart $add): BaseResponse
    {
        $product = self::integer($request->input('product'));
        $quantity = self::integer($request->input('quantity', '1'));
        $size = self::string($request->input('size'));
        $color = self::string($request->input('color'));

        if ($product === null || $product < 1 || $quantity === null || $quantity < 1 || $quantity > Cart::MAX_QUANTITY || $size === false || $color === false) {
            return $this->refused($request, self::text('shop.cart.invalid'), null);
        }

        try {
            $change = $add->handle($product, $size, $color, $quantity);
        } catch (CartException $e) {
            return $this->refused($request, self::message(new CartNotice($e->problem, $e->params)), $e->productSlug);
        }

        $message = $change->notice === null
            ? (string) __('shop.cart.added', ['name' => $change->title])
            : self::message($change->notice);

        return $this->done($request, $change, $message);
    }

    public function update(Request $request, string $line, UpdateCartLine $update): BaseResponse
    {
        $quantity = self::integer($request->input('quantity'));
        if ($quantity === null || $quantity < 0 || $quantity > Cart::MAX_QUANTITY) {
            return $this->refused($request, self::text('shop.cart.invalid'), null, toCart: true);
        }

        try {
            $change = $update->handle($line, $quantity);
        } catch (CartException $e) {
            return $this->refused($request, self::message(new CartNotice($e->problem, $e->params)), null, toCart: true);
        }

        $message = match (true) {
            $change->notice !== null => self::message($change->notice),
            $change->quantity === 0 => self::text('shop.cart.removed'),
            default => (string) __('shop.cart.updated', ['name' => $change->title]),
        };

        return $this->done($request, $change, $message);
    }

    public function remove(Request $request, string $line, RemoveFromCart $remove): BaseResponse
    {
        return $this->done($request, $remove->handle($line), self::text('shop.cart.removed'));
    }

    private function done(Request $request, CartChange $change, string $message): BaseResponse
    {
        if ($request->expectsJson()) {
            return $this->withCount(new JsonResponse([
                'ok' => true,
                'message' => $message,
                'count' => $change->count,
                'quantity' => $change->quantity,
                'cart_url' => route('shop.cart'),
            ]), $change->count);
        }

        return $this->withCount(redirect()->to(route('shop.cart'), 303)->with('cart_status', $message), $change->count);
    }

    /**
     * A refused change: JSON 422, or back to the product page (`#buy`) / the cart with the message flashed.
     */
    private function refused(Request $request, string $message, ?string $productSlug, bool $toCart = false): BaseResponse
    {
        $count = $this->resolve->handle()->count;

        if ($request->expectsJson()) {
            return $this->withCount(new JsonResponse(['ok' => false, 'message' => $message, 'count' => $count], 422), $count);
        }

        $target = match (true) {
            $toCart => route('shop.cart'),
            $productSlug !== null => route('shop.product', [$productSlug]).'#buy',
            default => route('shop.cart'), // never the Referer (L9-04: no header-controlled redirect target)
        };

        return $this->withCount(redirect()->to($target, 303)->with('cart_error', $message), $count);
    }

    private function withCount(BaseResponse $response, int $count): BaseResponse
    {
        $secure = $this->config->get('session.secure');
        $response->headers->setCookie($count > 0
            ? Cookie::create(self::COUNT_COOKIE, (string) $count, time() + 60 * (int) $this->config->get('session.lifetime', 120), '/', null, is_bool($secure) ? $secure : null, false, false, Cookie::SAMESITE_LAX)
            : Cookie::create(self::COUNT_COOKIE, null, 1, '/', null, is_bool($secure) ? $secure : null, false, false, Cookie::SAMESITE_LAX));

        return $response;
    }

    /**
     * «شاید لازم داشته باشی»: bought together with the cart's products, then best sellers — buyable and not in the
     * cart yet (cached Shop reads).
     *
     * @return list<ProductCardData>
     */
    private function suggestions(CartSummary $cart): array
    {
        $exclude = array_flip($cart->productIds());
        $cards = [];
        foreach (array_slice($cart->productIds(), 0, 3) as $id) {
            array_push($cards, ...$this->products->frequentlyBoughtWith($id, self::SUGGESTIONS));
        }
        array_push($cards, ...$this->products->bestSellers(self::SUGGESTIONS * 2));

        $picked = [];
        foreach ($cards as $card) {
            if (! isset($exclude[$card->id]) && ! isset($picked[$card->id]) && $card->isPurchasable()) {
                $picked[$card->id] = $card;
            }
        }

        return array_slice(array_values($picked), 0, self::SUGGESTIONS);
    }

    /**
     * @param  list<int|null>  $ids
     * @return array<int, MediaData>
     */
    private function coverMedia(array $ids): array
    {
        $ids = array_values(array_unique(array_filter($ids, static fn (?int $id): bool => $id !== null)));

        return $ids === [] ? [] : $this->media->findMany($ids);
    }

    /**
     * @return array<string, string> line key → translated notice
     */
    private function itemNotices(CartSummary $cart): array
    {
        $notices = [];
        foreach ($cart->items as $item) {
            if ($item->notice !== null) {
                $notices[$item->key] = self::message($item->notice);
            }
        }

        return $notices;
    }

    private function flash(Request $request, string $key): ?string
    {
        $value = $request->hasSession() ? $request->session()->get($key) : null;

        return is_string($value) && $value !== '' ? $value : null;
    }

    private static function message(CartNotice $notice): string
    {
        $text = __($notice->problem->translationKey(), array_map(static fn (string|int $v): string => is_int($v) ? fa_digits($v) : $v, $notice->params));

        return is_string($text) ? $text : '';
    }

    /** Latin or Persian digits → int; anything else null. */
    private static function integer(mixed $value): ?int
    {
        if (is_int($value)) {
            return $value;
        }
        if (! is_string($value)) {
            return null;
        }
        $value = trim(PersianDigits::toLatin($value));

        return preg_match('/^\d{1,9}$/', $value) === 1 ? (int) $value : null;
    }

    /** Optional short string field: null when absent / empty, false when invalid. */
    private static function string(mixed $value): string|false|null
    {
        if ($value === null || $value === '') {
            return null;
        }

        return is_string($value) && mb_strlen($value) <= 60 ? $value : false;
    }

    private static function text(string $key): string
    {
        $text = __($key);

        return is_string($text) ? $text : '';
    }
}

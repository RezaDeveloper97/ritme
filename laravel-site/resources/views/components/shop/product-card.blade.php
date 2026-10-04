{{--
    <x-shop.product-card :product="$card" :media="$media" :index="$loop->index" class="…"/>
    A ProductCardData as <x-cards.product> (L6-02): brand as the «seller» line (single seller), rating only from real
    reviews (ProductCardData::rating(), else none), price + compare-at in tomans, cover photo (MediaData prefetched by
    the controller, keyed by id) or the product's illustration on a stage tint cycling like the design. A sold-out
    product shows «ناموجود» in the corner badge. Extra attributes go on the card's <article>.
--}}
@props(['product', 'media' => [], 'index' => 0])
@php
    /** @var \App\Domain\Shop\Catalog\Data\ProductCardData $product */
    $tints = ['pregnancy', 'primary', 'postpartum', 'pregnancy', 'postpartum'];
    $rating = $product->rating();
    $cover = $product->coverMediaId === null ? null : ($media[$product->coverMediaId] ?? $product->coverMediaId);
    $badge = $product->isPurchasable() ? $product->badge : $product->stockStatus->label();
@endphp
<x-cards.product {{ $attributes }}
    :href="route('shop.product', [$product->slug])"
    :title="$product->title"
    :seller="$product->brand?->name"
    :rating="$rating?->ratingValue"
    :reviews="$rating?->ratingCount"
    :price="$product->price->toToman()"
    :compare="$product->compareAtPrice?->toToman()"
    :badge="$badge"
    :media="$cover"
    :illustration="$cover === null ? $product->illustration : null"
    :tint="$tints[$index % count($tints)]"/>

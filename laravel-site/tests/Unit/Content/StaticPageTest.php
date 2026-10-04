<?php

declare(strict_types=1);

use App\Domain\Content\Enums\HeaderVariant;
use App\Domain\Content\Enums\NavItem;
use App\Domain\Content\Enums\StaticPage;

it('puts the dark header on exactly the ten audited pages', function (): void {
    $dark = array_values(array_filter(StaticPage::cases(), static fn (StaticPage $page): bool => $page->headerVariant() === HeaderVariant::Dark));

    expect(array_map(static fn (StaticPage $page): string => $page->value, $dark))->toEqualCanonicalizing([
        'home', 'stage.cycle', 'stage.ttc', 'stage.pregnancy', 'stage.postpartum', 'stage.menopause', 'stage.teen',
        'about', 'privacy', 'social-responsibility',
    ]);
});

it('maps pages to the active nav item from the audit', function (string $route, ?NavItem $item): void {
    expect(StaticPage::from($route)->navItem())->toBe($item);
})->with([
    ['home', NavItem::Home],
    ['stage.teen', NavItem::Stages],
    ['directory.business', NavItem::Services],
    ['shop.cart', NavItem::Shop],
    ['blog.index', NavItem::Blog],
    ['tools', NavItem::Tools],
    ['about', NavItem::About],
    ['contact', null],
    ['faq', null],
    ['plus', null],
    ['privacy', null],
    ['social-responsibility', null],
]);

it('resolves the active item of parameterised routes by prefix', function (?string $route, ?NavItem $item): void {
    expect(NavItem::forRoute($route))->toBe($item);
})->with([
    ['blog.show', NavItem::Blog],
    ['directory.place', NavItem::Services],
    ['shop.product', NavItem::Shop],
    ['stage.anything', NavItem::Stages],
    ['unknown', null],
    [null, null],
    ['', null],
]);

it('links «مرحله‌ها» to the first stage with a decorative chevron only there', function (): void {
    expect(NavItem::Stages->page())->toBe(StaticPage::Cycle)
        ->and(array_values(array_filter(NavItem::cases(), static fn (NavItem $item): bool => $item->hasChevron())))->toBe([NavItem::Stages]);
});

it('builds breadcrumb trails from home through the parents', function (): void {
    expect(StaticPage::Home->trail())->toBe([StaticPage::Home])
        ->and(StaticPage::Faq->trail())->toBe([StaticPage::Home, StaticPage::Faq])
        ->and(StaticPage::DirectoryJoin->trail())->toBe([StaticPage::Home, StaticPage::Directory, StaticPage::DirectoryBusiness, StaticPage::DirectoryJoin])
        ->and(StaticPage::ShopCheckout->trail())->toBe([StaticPage::Home, StaticPage::Shop, StaticPage::ShopCart, StaticPage::ShopCheckout]);
});

it('keeps transactional pages out of the index and the app-CTA map in sync with the design', function (): void {
    expect(StaticPage::ShopCart->indexable())->toBeFalse()
        ->and(StaticPage::DirectoryJoinDone->indexable())->toBeFalse()
        ->and(StaticPage::Home->indexable())->toBeTrue()
        ->and(StaticPage::Home->sitemapPriority())->toBe(1.0)
        ->and(StaticPage::Faq->hasAppCta())->toBeTrue()
        ->and(StaticPage::Contact->hasAppCta())->toBeFalse();

    foreach (StaticPage::cases() as $page) {
        expect($page->path())->toStartWith('/')
            ->and($page->label())->not->toBe('')
            ->and($page->sitemapChangefreq())->toBeIn(['always', 'hourly', 'daily', 'weekly', 'monthly', 'yearly', 'never']);
    }
});

it('resolves header variants from strings with a default', function (): void {
    expect(HeaderVariant::resolve('dark'))->toBe(HeaderVariant::Dark)
        ->and(HeaderVariant::resolve('nope'))->toBe(HeaderVariant::Light)
        ->and(HeaderVariant::resolve(null, HeaderVariant::Dark))->toBe(HeaderVariant::Dark);
});

<?php

declare(strict_types=1);

use App\Domain\Seo\Data\SeoImage;
use App\Domain\Seo\Schema\Data\AggregateRatingData;
use App\Domain\Seo\Schema\Data\BlogPostingData;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\Data\FaqItem;
use App\Domain\Seo\Schema\Data\ListEntry;
use App\Domain\Seo\Schema\Data\LocalBusinessData;
use App\Domain\Seo\Schema\Data\OfferData;
use App\Domain\Seo\Schema\Data\OpeningHoursData;
use App\Domain\Seo\Schema\Data\PersonData;
use App\Domain\Seo\Schema\Data\PostalAddressData;
use App\Domain\Seo\Schema\Data\ProductData;
use App\Domain\Seo\Schema\Enums\DayOfWeek;
use App\Domain\Seo\Schema\Enums\ItemAvailability;
use App\Domain\Seo\Schema\Enums\LocalBusinessType;
use App\Domain\Seo\Schema\Enums\WebPageType;
use App\Domain\Seo\Schema\Nodes\BlogPostingNode;
use App\Domain\Seo\Schema\Nodes\BreadcrumbListNode;
use App\Domain\Seo\Schema\Nodes\FaqPageNode;
use App\Domain\Seo\Schema\Nodes\ItemListNode;
use App\Domain\Seo\Schema\Nodes\LocalBusinessNode;
use App\Domain\Seo\Schema\Nodes\MobileApplicationNode;
use App\Domain\Seo\Schema\Nodes\OrganizationNode;
use App\Domain\Seo\Schema\Nodes\ProductNode;
use App\Domain\Seo\Schema\Nodes\WebPageNode;
use App\Domain\Seo\Schema\Nodes\WebSiteNode;
use App\Domain\Settings\Contracts\SettingsRepository;
use Tests\Unit\Seo\Schema\SchemaFixtures;

const SCHEMA_SITE = SchemaFixtures::SITE;

it('builds Organization from settings: logo, merged sameAs, contactPoint fallback to contact settings', function (): void {
    $node = OrganizationNode::make(SchemaFixtures::settings(), SCHEMA_SITE, SchemaFixtures::logo());

    expect($node)->toMatchArray([
        '@type' => 'Organization', '@id' => 'https://ritme.test/#organization', 'name' => 'ریتمی',
        'alternateName' => 'Ritme', 'url' => 'https://ritme.test/', 'foundingDate' => '2025',
        'sameAs' => ['https://instagram.com/ritme', 'https://linkedin.com/company/ritme', 'https://aparat.com/ritme'],
    ])
        ->and($node['logo'])->toMatchArray(['@type' => 'ImageObject', '@id' => 'https://ritme.test/#logo', 'url' => 'https://ritme.test/media/logo.png', 'width' => 512, 'height' => 512])
        ->and($node['contactPoint'])->toMatchArray(['@type' => 'ContactPoint', 'contactType' => 'customer support', 'telephone' => '021-0000', 'email' => 'support@ritme.test']);

    $bare = OrganizationNode::make(SchemaFixtures::settings(['contact' => ['support_email' => null, 'phone' => null]]), SCHEMA_SITE);
    expect($bare)->not->toHaveKeys(['logo', 'image', 'contactPoint']);
});

it('builds WebSite with a SearchAction only when a valid template is given', function (): void {
    $plain = WebSiteNode::make(SchemaFixtures::settings(), SCHEMA_SITE);
    $search = WebSiteNode::make(SchemaFixtures::settings(), SCHEMA_SITE, 'https://ritme.test/search?q={search_term_string}');

    expect($plain)->toMatchArray(['@type' => 'WebSite', '@id' => 'https://ritme.test/#website', 'url' => 'https://ritme.test/', 'name' => 'ریتمی', 'inLanguage' => 'fa-IR', 'publisher' => ['@id' => 'https://ritme.test/#organization']])
        ->and($plain)->not->toHaveKey('potentialAction')
        ->and(WebSiteNode::make(SchemaFixtures::settings(), SCHEMA_SITE, 'https://ritme.test/search'))->not->toHaveKey('potentialAction')
        ->and($search['potentialAction'])->toBe([
            '@type' => 'SearchAction',
            'target' => ['@type' => 'EntryPoint', 'urlTemplate' => 'https://ritme.test/search?q={search_term_string}'],
            'query-input' => 'required name=search_term_string',
        ]);
});

it('builds every WebPage type with the YMYL review hook', function (WebPageType $type): void {
    $node = WebPageNode::make(
        url: 'https://ritme.test/about', siteUrl: SCHEMA_SITE, name: 'درباره ریتمی', type: $type, hasBreadcrumb: true,
        reviewedBy: new PersonData('دکتر نمونه', credential: 'متخصص زنان و زایمان'), lastReviewed: '2026-09-01',
    );

    expect($node)->toMatchArray([
        '@type' => $type->value, '@id' => 'https://ritme.test/about#webpage', 'url' => 'https://ritme.test/about',
        'isPartOf' => ['@id' => 'https://ritme.test/#website'], 'breadcrumb' => ['@id' => 'https://ritme.test/about#breadcrumb'],
        'lastReviewed' => '2026-09-01',
    ])
        ->and($node['reviewedBy'])->toMatchArray(['@type' => 'Person', 'name' => 'دکتر نمونه', 'hasCredential' => ['@type' => 'EducationalOccupationalCredential', 'credentialCategory' => 'متخصص زنان و زایمان']])
        ->and($node)->not->toHaveKey('about');
})->with([WebPageType::WebPage, WebPageType::AboutPage, WebPageType::ContactPage, WebPageType::CollectionPage]);

it('builds BreadcrumbList with 1-based positions, absolute items and the page URL for the last step', function (): void {
    $node = BreadcrumbListNode::make('https://ritme.test/blog/post', [
        new BreadcrumbItem('خانه', 'https://ritme.test/'),
        new BreadcrumbItem('بدون صفحه'),
        new BreadcrumbItem('مجله', 'https://ritme.test/blog'),
        new BreadcrumbItem('یک مقاله'),
    ]);

    expect($node)->toBe([
        '@type' => 'BreadcrumbList',
        '@id' => 'https://ritme.test/blog/post#breadcrumb',
        'itemListElement' => [
            ['@type' => 'ListItem', 'position' => 1, 'name' => 'خانه', 'item' => 'https://ritme.test/'],
            ['@type' => 'ListItem', 'position' => 2, 'name' => 'مجله', 'item' => 'https://ritme.test/blog'],
            ['@type' => 'ListItem', 'position' => 3, 'name' => 'یک مقاله', 'item' => 'https://ritme.test/blog/post'],
        ],
    ]);
});

it('builds MobileApplication: store links, free offer, health category, no invented rating', function (): void {
    $node = MobileApplicationNode::make(SchemaFixtures::settings(), SCHEMA_SITE);

    expect($node)->toMatchArray([
        '@type' => 'MobileApplication', '@id' => 'https://ritme.test/#app', 'name' => 'ریتمی',
        'applicationCategory' => 'HealthApplication', 'operatingSystem' => 'Android, Web',
        'installUrl' => 'https://cafebazaar.ir/app/ir.ritmeapp.ritme',
        'offers' => ['@type' => 'Offer', 'price' => 0, 'priceCurrency' => 'IRR', 'availability' => 'https://schema.org/InStock', 'itemCondition' => 'https://schema.org/NewCondition'],
    ])->and($node)->not->toHaveKey('aggregateRating');

    $rated = MobileApplicationNode::make(SchemaFixtures::settings(), SCHEMA_SITE, rating: new AggregateRatingData(4.62, 120));
    expect($rated['aggregateRating'])->toMatchArray(['@type' => 'AggregateRating', 'ratingValue' => 4.6, 'ratingCount' => 120]);
});

it('builds FAQPage with Question / acceptedAnswer pairs', function (): void {
    $node = FaqPageNode::make('https://ritme.test/faq', [new FaqItem(' داده‌هایم کجا می‌ماند؟ ', '<p>پیش خودت.</p>')]);

    expect($node)->toMatchArray(['@type' => 'FAQPage', '@id' => 'https://ritme.test/faq#faq', 'isPartOf' => ['@id' => 'https://ritme.test/faq#webpage']])
        ->and($node['mainEntity'])->toBe([
            ['@type' => 'Question', 'name' => 'داده‌هایم کجا می‌ماند؟', 'acceptedAnswer' => ['@type' => 'Answer', 'text' => '<p>پیش خودت.</p>']],
        ]);
});

it('builds BlogPosting with author Person, publisher, image and modified date fallback', function (): void {
    $node = BlogPostingNode::make(new BlogPostingData(
        url: 'https://ritme.test/blog/pms',
        headline: str_repeat('سندروم پیش از قاعدگی ', 10),
        datePublished: '2026-09-20T10:00:00+03:30',
        authors: [new PersonData('مریم نویسنده', url: 'https://ritme.test/authors/maryam', jobTitle: 'نویسنده سلامت')],
        image: new SeoImage('https://ritme.test/media/pms.jpg', 1200, 675, 'PMS'),
        keywords: ['PMS', 'PMS', 'قاعدگی'],
    ), SCHEMA_SITE);

    expect($node)->toMatchArray([
        '@type' => 'BlogPosting', '@id' => 'https://ritme.test/blog/pms#article',
        'mainEntityOfPage' => ['@id' => 'https://ritme.test/blog/pms#webpage'],
        'datePublished' => '2026-09-20T10:00:00+03:30', 'dateModified' => '2026-09-20T10:00:00+03:30',
        'publisher' => ['@id' => 'https://ritme.test/#organization'], 'keywords' => ['PMS', 'قاعدگی'],
    ])
        ->and(mb_strlen($node['headline']))->toBeLessThanOrEqual(110)
        ->and($node['image'])->toMatchArray(['@type' => 'ImageObject', 'url' => 'https://ritme.test/media/pms.jpg', 'width' => 1200, 'height' => 675])
        ->and($node['author'][0])->toMatchArray(['@type' => 'Person', 'name' => 'مریم نویسنده', 'url' => 'https://ritme.test/authors/maryam', 'jobTitle' => 'نویسنده سلامت'])
        ->and($node['author'][0]['@id'])->toStartWith('https://ritme.test/#/person/');
});

it('builds Product with Offer and only a real AggregateRating', function (): void {
    $product = new ProductData(
        url: 'https://ritme.test/shop/pad', name: 'پد قاعدگی', imageUrls: ['https://ritme.test/media/pad.jpg'],
        offer: new OfferData(1_250_000, availability: ItemAvailability::OutOfStock), sku: 'PAD-1', brand: 'ریتمی',
    );
    $node = ProductNode::make($product);

    expect($node)->toMatchArray([
        '@type' => 'Product', '@id' => 'https://ritme.test/shop/pad#product', 'name' => 'پد قاعدگی',
        'image' => ['https://ritme.test/media/pad.jpg'], 'sku' => 'PAD-1', 'brand' => ['@type' => 'Brand', 'name' => 'ریتمی'],
    ])
        ->and($node['offers'])->toMatchArray(['@type' => 'Offer', 'price' => 1_250_000, 'priceCurrency' => 'IRR', 'availability' => 'https://schema.org/OutOfStock', 'url' => 'https://ritme.test/shop/pad'])
        ->and($node)->not->toHaveKey('aggregateRating');
});

it('refuses ratings without real reviews or out of range', function (float $value, int $count): void {
    new AggregateRatingData($value, $count);
})->throws(InvalidArgumentException::class)->with([[5.0, 0], [6.0, 3], [0.5, 3]]);

it('builds LocalBusiness subtypes with address, geo and opening hours', function (): void {
    $node = LocalBusinessNode::make(new LocalBusinessData(
        url: 'https://ritme.test/directory/mehr', name: 'مهدکودک مهر',
        address: new PostalAddressData('خیابان نمونه، پلاک ۱', 'تهران', 'تهران', '1234567890'),
        type: LocalBusinessType::ChildCare, telephone: '021-1111', latitude: 35.6997, longitude: 51.3380,
        openingHours: [new OpeningHoursData([DayOfWeek::Saturday, DayOfWeek::Sunday], '07:30', '16:00')],
    ));

    expect($node)->toMatchArray([
        '@type' => 'ChildCare', '@id' => 'https://ritme.test/directory/mehr#place', 'name' => 'مهدکودک مهر',
        'address' => ['@type' => 'PostalAddress', 'streetAddress' => 'خیابان نمونه، پلاک ۱', 'addressLocality' => 'تهران', 'addressRegion' => 'تهران', 'postalCode' => '1234567890', 'addressCountry' => 'IR'],
        'geo' => ['@type' => 'GeoCoordinates', 'latitude' => 35.6997, 'longitude' => 51.338],
        'openingHoursSpecification' => [[
            '@type' => 'OpeningHoursSpecification',
            'dayOfWeek' => ['https://schema.org/Saturday', 'https://schema.org/Sunday'], 'opens' => '07:30', 'closes' => '16:00',
        ]],
    ])->and(LocalBusinessNode::make(new LocalBusinessData('https://ritme.test/d/g', 'باشگاه', new PostalAddressData('x', 'y'), LocalBusinessType::SportsActivityLocation))['@type'])
        ->toBe('SportsActivityLocation');
});

it('rejects malformed opening hours', function (): void {
    new OpeningHoursData([DayOfWeek::Monday], '7:30', '25:00');
})->throws(InvalidArgumentException::class);

it('builds ItemList in summary-page form', function (): void {
    $node = ItemListNode::make('https://ritme.test/blog', [new ListEntry('https://ritme.test/blog/a', 'الف'), new ListEntry('https://ritme.test/blog/b')], 'مجله');

    expect($node)->toMatchArray(['@type' => 'ItemList', '@id' => 'https://ritme.test/blog#itemlist', 'numberOfItems' => 2])
        ->and($node['itemListElement'])->toBe([
            ['@type' => 'ListItem', 'position' => 1, 'url' => 'https://ritme.test/blog/a', 'name' => 'الف'],
            ['@type' => 'ListItem', 'position' => 2, 'url' => 'https://ritme.test/blog/b'],
        ]);
});

<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Enums;

/**
 * schema.org type of the page's `#webpage` node.
 */
enum WebPageType: string
{
    case WebPage = 'WebPage';
    case AboutPage = 'AboutPage';
    case ContactPage = 'ContactPage';
    case CollectionPage = 'CollectionPage';
    case ItemPage = 'ItemPage';
    case FaqPage = 'FAQPage';
    case SearchResultsPage = 'SearchResultsPage';
    case ProfilePage = 'ProfilePage';
    case CheckoutPage = 'CheckoutPage';
    case MedicalWebPage = 'MedicalWebPage';
}

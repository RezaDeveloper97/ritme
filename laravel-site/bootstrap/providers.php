<?php

declare(strict_types=1);

use App\Providers\AppServiceProvider;
use App\Providers\Domain;

return [
    AppServiceProvider::class,

    // Bounded contexts (see docs/ARCHITECTURE.md).
    Domain\SettingsServiceProvider::class,
    Domain\SeoServiceProvider::class,
    Domain\MediaServiceProvider::class,
    Domain\ContentServiceProvider::class,
    Domain\FaqServiceProvider::class,
    Domain\ContactServiceProvider::class,
    Domain\BlogServiceProvider::class,
    Domain\NewsletterServiceProvider::class,
    Domain\DirectoryServiceProvider::class,
    Domain\ShopServiceProvider::class,
    Domain\PwaServiceProvider::class,
];

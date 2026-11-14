<?php

declare(strict_types=1);

namespace App\Providers\Domain;

use App\Domain\Content\Enums\SiteSection;
use App\Domain\Content\SiteSections;
use App\Providers\DomainServiceProvider;
use Illuminate\Support\Facades\Blade;

final class ContentServiceProvider extends DomainServiceProvider
{
    public function boot(): void
    {
        parent::boot();

        // In-page links into a section switched off in the admin «صفحه‌ها» settings:
        //   @siteSection('shop') … @endsiteSection   or, inside @php,  Blade::check('siteSection', 'shop')
        Blade::if('siteSection', fn (string $section): bool => $this->app->make(SiteSections::class)->enabled(SiteSection::from($section)));
    }
}

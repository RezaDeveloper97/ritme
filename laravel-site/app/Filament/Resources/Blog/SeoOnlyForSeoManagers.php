<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog;

use Filament\Facades\Filament;

/**
 * For edit pages of magazine resources: an SEO manager's save keeps the record's content as it is (only the SEO
 * relationship is written), whatever the Livewire payload contains.
 */
trait SeoOnlyForSeoManagers
{
    /**
     * @param  array<string, mixed>  $data
     * @return array<string, mixed>
     */
    protected function mutateFormDataBeforeSave(array $data): array
    {
        return BlogPolicy::canEditContent(Filament::auth()->user()) ? $data : [];
    }
}

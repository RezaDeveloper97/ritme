<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Reviews\Pages;

use App\Domain\Shop\Catalog\Enums\ReviewStatus;
use App\Filament\Resources\Shop\Reviews\ProductReviewResource;
use Filament\Resources\Pages\ListRecords;
use Filament\Schemas\Components\Tabs\Tab;
use Illuminate\Database\Eloquent\Builder;

final class ListProductReviews extends ListRecords
{
    protected static string $resource = ProductReviewResource::class;

    /**
     * @return array<string, Tab>
     */
    public function getTabs(): array
    {
        $tabs = [];
        foreach (ReviewStatus::cases() as $status) {
            $tabs[$status->value] = Tab::make($status->label())
                ->modifyQueryUsing(static fn (Builder $query): Builder => $query->where('status', $status->value));
        }
        $tabs['all'] = Tab::make('همه');

        return $tabs;
    }

    public function getDefaultActiveTab(): string
    {
        return ReviewStatus::Pending->value;
    }
}

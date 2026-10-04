<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Reviews\Pages;

use App\Domain\Directory\Enums\ReviewStatus;
use App\Filament\Resources\Directory\Reviews\PlaceReviewResource;
use Filament\Resources\Pages\ListRecords;
use Filament\Schemas\Components\Tabs\Tab;
use Illuminate\Database\Eloquent\Builder;

final class ListPlaceReviews extends ListRecords
{
    protected static string $resource = PlaceReviewResource::class;

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

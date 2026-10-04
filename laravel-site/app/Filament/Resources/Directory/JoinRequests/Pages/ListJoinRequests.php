<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\JoinRequests\Pages;

use App\Domain\Directory\Join\Enums\JoinRequestStatus;
use App\Filament\Resources\Directory\JoinRequests\JoinRequestResource;
use Filament\Resources\Pages\ListRecords;
use Filament\Schemas\Components\Tabs\Tab;
use Illuminate\Database\Eloquent\Builder;

final class ListJoinRequests extends ListRecords
{
    protected static string $resource = JoinRequestResource::class;

    /**
     * @return array<string, Tab>
     */
    public function getTabs(): array
    {
        $tabs = [];
        foreach (JoinRequestStatus::cases() as $status) {
            $tabs[$status->value] = Tab::make($status->label())
                ->modifyQueryUsing(static fn (Builder $query): Builder => $query->where('status', $status->value));
        }
        $tabs['all'] = Tab::make('همه');

        return $tabs;
    }

    public function getDefaultActiveTab(): string
    {
        return JoinRequestStatus::Pending->value;
    }
}

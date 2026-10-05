<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Orders\Pages;

use App\Domain\Shop\Ordering\Enums\OrderStatus;
use App\Domain\Shop\Ordering\Models\Order;
use App\Filament\Resources\Shop\Orders\OrderResource;
use Filament\Resources\Pages\ListRecords;
use Filament\Schemas\Components\Tabs\Tab;
use Illuminate\Database\Eloquent\Builder;

/**
 * Orders by status (with counts) in lifecycle order, plus «همه».
 */
final class ListOrders extends ListRecords
{
    protected static string $resource = OrderResource::class;

    protected function getHeaderActions(): array
    {
        return [OrderResource::exportAction()];
    }

    /**
     * @return array<string, Tab>
     */
    public function getTabs(): array
    {
        $counts = Order::query()->toBase()->selectRaw('status, COUNT(*) as aggregate')->groupBy('status')->pluck('aggregate', 'status');

        $tabs = [];
        foreach (OrderStatus::cases() as $status) {
            $tabs[$status->value] = Tab::make($status->label())
                ->badge(fa_digits((int) ($counts[$status->value] ?? 0)))
                ->badgeColor(OrderResource::statusColor($status))
                ->modifyQueryUsing(static fn (Builder $query): Builder => $query->where('status', $status->value));
        }
        $tabs['all'] = Tab::make('همه');

        return $tabs;
    }

    public function getDefaultActiveTab(): string
    {
        return OrderStatus::Pending->value;
    }

    /**
     * The current tab, placed-date range and search, passed on to the export link.
     *
     * @return array<string, string>
     */
    public function exportParameters(): array
    {
        $status = OrderStatus::tryFrom((string) $this->activeTab);
        $from = $this->tableFilters['placed']['from'] ?? null;
        $until = $this->tableFilters['placed']['until'] ?? null;
        $search = trim((string) $this->tableSearch);

        return array_filter([
            'status' => $status?->value,
            'from' => is_string($from) && $from !== '' ? $from : null,
            'until' => is_string($until) && $until !== '' ? $until : null,
            'search' => $search !== '' ? $search : null,
        ], static fn (?string $value): bool => $value !== null);
    }
}

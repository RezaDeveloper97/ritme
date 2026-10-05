<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Orders\Pages;

use App\Domain\Shop\Ordering\Models\Order;
use App\Filament\Resources\Shop\Orders\OrderResource;
use Filament\Actions\Action;
use Filament\Resources\Pages\ViewRecord;
use Filament\Support\Icons\Heroicon;

/**
 * One order with the recipient's full data (to call and deliver), its lines, notes + status history, the lifecycle
 * actions, an internal note and the printable invoice.
 */
final class ViewOrder extends ViewRecord
{
    protected static string $resource = OrderResource::class;

    protected function getHeaderActions(): array
    {
        $record = $this->getRecord();

        return [
            ...OrderResource::statusActions(),
            OrderResource::noteAction(),
            OrderResource::invoiceAction(),
            Action::make('call')
                ->label('تماس با گیرنده')
                ->icon(Heroicon::OutlinedPhone)
                ->color('gray')
                ->url($record instanceof Order ? 'tel:'.$record->mobile : null),
        ];
    }
}

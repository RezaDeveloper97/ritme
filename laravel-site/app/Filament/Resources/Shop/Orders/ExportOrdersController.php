<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Orders;

use App\Domain\Shop\Ordering\Actions\ExportOrders;
use App\Domain\Shop\Ordering\Enums\OrderStatus;
use App\Domain\Shop\Ordering\Models\Order;
use Filament\Facades\Filament;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Gate;
use Symfony\Component\HttpFoundation\StreamedResponse;

/**
 * GET {admin}/shop/orders/export?status=&from=&until=&search= — streamed CSV download (registered by
 * OrderResource::registerRoutes() inside the panel's auth middleware). Access: OrderPolicy::export().
 */
final class ExportOrdersController
{
    public function __invoke(Request $request, ExportOrders $export): StreamedResponse
    {
        $user = Filament::auth()->user();
        Gate::forUser($user)->authorize('export', Order::class);

        $status = OrderStatus::tryFrom((string) $request->query('status', ''));
        $from = self::day($request->query('from'));
        $until = self::day($request->query('until'));
        $search = $request->query('search');
        $search = is_string($search) ? mb_substr($search, 0, 64) : null;

        return response()->streamDownload(static function () use ($export, $status, $from, $until, $search, $user): void {
            $out = fopen('php://output', 'wb');
            if ($out === false) {
                return;
            }

            $export->handle($out, $status, $from, $until, $search, $user);
            fclose($out);
        }, ExportOrders::filename(), [
            'Content-Type' => 'text/csv; charset=UTF-8',
            'Cache-Control' => 'private, no-store',
            'X-Robots-Tag' => 'noindex, nofollow',
        ]);
    }

    private static function day(mixed $value): ?string
    {
        return is_string($value) && preg_match('/^\d{4}-\d{2}-\d{2}/', $value) === 1 ? substr($value, 0, 10) : null;
    }
}

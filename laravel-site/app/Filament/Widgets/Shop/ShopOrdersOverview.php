<?php

declare(strict_types=1);

namespace App\Filament\Widgets\Shop;

use App\Domain\Shop\Ordering\Enums\OrderStatus;
use App\Domain\Shop\Ordering\Models\Order;
use App\Filament\Auth\AdminAccess;
use App\Filament\Auth\AdminRole;
use App\Filament\Resources\Shop\Orders\OrderResource;
use App\Support\Money\Money;
use Filament\Facades\Filament;
use Filament\Support\Icons\Heroicon;
use Filament\Widgets\StatsOverviewWidget;
use Filament\Widgets\StatsOverviewWidget\Stat;
use Illuminate\Support\Carbon;

/**
 * Shop dashboard numbers (L6-06): orders placed today, orders waiting for confirmation, and revenue of the last 7 / 30
 * days (order totals, cancelled and demo orders excluded — cash on delivery, so this is ordered value, not cash in
 * hand). Shop managers + super-admins only. Dates are Tehran days (app timezone).
 */
final class ShopOrdersOverview extends StatsOverviewWidget
{
    protected static ?int $sort = 2;

    protected ?string $pollingInterval = null;

    protected ?string $heading = 'فروشگاه';

    public static function canView(): bool
    {
        return AdminAccess::allows(Filament::auth()->user(), [AdminRole::ShopManager]);
    }

    protected function getStats(): array
    {
        $today = Carbon::now()->startOfDay();
        $todayCount = Order::query()->where('created_at', '>=', $today)->where('is_demo', false)->count();
        $pending = Order::query()->where('status', OrderStatus::Pending->value)->count();

        return [
            Stat::make('سفارش‌های امروز', fa_digits($todayCount))
                ->icon(Heroicon::OutlinedShoppingCart),
            Stat::make('در انتظار تأیید', fa_digits($pending))
                ->description('تماس و تأیید با مشتری')
                ->color($pending > 0 ? 'warning' : 'gray')
                ->url(OrderResource::getUrl('index')),
            Stat::make('فروش ۷ روز اخیر', self::revenue(7)->format())
                ->description('بدون سفارش‌های لغوشده'),
            Stat::make('فروش ۳۰ روز اخیر', self::revenue(30)->format())
                ->description('بدون سفارش‌های لغوشده'),
        ];
    }

    public static function revenue(int $days): Money
    {
        $since = Carbon::now()->startOfDay()->subDays($days - 1);
        $rials = (int) Order::query()
            ->where('created_at', '>=', $since)
            ->where('status', '!=', OrderStatus::Cancelled->value)
            ->where('is_demo', false)
            ->sum('total');

        return Money::fromRial($rials);
    }
}

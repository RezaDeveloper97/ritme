<?php

declare(strict_types=1);

namespace App\Filament\Pages\Settings;

use App\Domain\Settings\Data\ShopSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Filament\Auth\AdminRole;
use BackedEnum;
use Filament\Forms\Components\TextInput;
use Filament\Schemas\Components\Section;
use Filament\Support\Icons\Heroicon;

/**
 * Shop business settings (L6-06): shipping fee, free-shipping threshold, cash-on-delivery cap (all in tomans; empty
 * = not set → config fallback) and the low-stock threshold of the admin widget. Shop managers + super-admins; saved
 * through UpdateSettings with an activity-log entry (SettingsPage).
 */
final class ShopSettingsPage extends SettingsPage
{
    protected static array $roles = [AdminRole::ShopManager];

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedShoppingBag;

    protected static ?string $navigationLabel = 'فروشگاه';

    protected static ?string $title = 'تنظیمات فروشگاه';

    protected static ?string $slug = 'settings/shop';

    protected static ?int $navigationSort = 60;

    protected static function group(): SettingGroup
    {
        return SettingGroup::Shop;
    }

    protected function fields(): array
    {
        $toman = static fn (string $name, string $label): TextInput => TextInput::make($name)
            ->label($label)
            ->integer()
            ->minValue(0)
            ->maxValue(ShopSettings::MAX_AMOUNT)
            ->suffix('تومان')
            ->extraInputAttributes(['dir' => 'ltr']);

        return [
            Section::make('ارسال و پرداخت')
                ->description('خالی = تعیین‌نشده (در سبد «محاسبه در مرحله بعد» و در فاکتور «هنگام تماس اعلام می‌شود»).')
                ->columns(3)
                ->schema([
                    $toman('shipping_flat_fee', 'هزینه ثابت ارسال')->helperText('۰ یعنی ارسال همیشه رایگان.'),
                    $toman('free_shipping_over', 'ارسال رایگان از مبلغ'),
                    $toman('cod_max_amount', 'سقف مبلغ پرداخت در محل')->helperText('سفارش بالاتر از این مبلغ ثبت نمی‌شود.'),
                ]),
            Section::make('موجودی')->schema([
                TextInput::make('low_stock_threshold')->label('هشدار موجودی کم از')
                    ->integer()->minValue(0)->maxValue(1000)->required()->suffix('عدد')
                    ->helperText('محصول یا تنوعی که حداکثر این تعداد موجودی دارد در داشبورد فهرست می‌شود.'),
            ]),
        ];
    }
}

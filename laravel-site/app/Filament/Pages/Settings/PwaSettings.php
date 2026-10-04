<?php

declare(strict_types=1);

namespace App\Filament\Pages\Settings;

use App\Domain\Media\Models\Media;
use App\Domain\Pwa\Actions\GeneratePwaIcons;
use App\Domain\Pwa\Support\PwaIconFiles;
use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\PwaSettings as PwaSettingsData;
use App\Domain\Settings\Enums\SettingGroup;
use App\Filament\Forms\Components\MediaPicker;
use BackedEnum;
use Closure;
use Filament\Forms\Components\ColorPicker;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Notifications\Notification;
use Filament\Support\Icons\Heroicon;
use Throwable;

/**
 * PWA settings (L8-01): manifest names, colours and the square logo the install icons are generated from. Saving
 * regenerates the icon set (GeneratePwaIcons) right away; the manifest cache is busted by SettingObserver.
 */
final class PwaSettings extends SettingsPage
{
    private const COLOR_RULE = 'regex:/^#[0-9A-Fa-f]{6}$/';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedDevicePhoneMobile;

    protected static ?string $navigationLabel = 'اپ وب (PWA)';

    protected static ?string $title = 'اپ وب قابل نصب (PWA)';

    protected static ?string $slug = 'settings/pwa';

    protected static ?int $navigationSort = 60;

    protected static function group(): SettingGroup
    {
        return SettingGroup::Pwa;
    }

    protected function fields(): array
    {
        return [
            TextInput::make('name')->label('نام کامل اپ')->required()->maxLength(60)
                ->helperText('در صفحه نصب و فهرست برنامه‌ها نمایش داده می‌شود.'),
            TextInput::make('short_name')->label('نام کوتاه')->required()->maxLength(12)
                ->helperText('زیر آیکن روی صفحه اصلی گوشی؛ حداکثر ۱۲ نویسه.'),
            Textarea::make('description')->label('توضیح')->rows(2)->maxLength(300),
            ColorPicker::make('theme_color')->label('رنگ نوار مرورگر (theme)')->required()->rule(self::COLOR_RULE),
            ColorPicker::make('background_color')->label('رنگ پس‌زمینه صفحه آغاز')->required()->rule(self::COLOR_RULE),
            MediaPicker::make('icon_media_id')
                ->label('لوگوی مربعی اپ')
                ->helperText('PNG یا WebP مربعی، دست‌کم '.PwaIconFiles::MIN_SOURCE_SIZE.' پیکسل، ترجیحاً با پس‌زمینه شفاف. آیکن‌ها، favicon و آیکن iOS از آن ساخته می‌شوند؛ خالی = آیکن‌های پیش‌فرض.')
                ->rule(static fn (): Closure => static function (string $attribute, mixed $value, Closure $fail): void {
                    $media = is_numeric($value) ? Media::query()->find((int) $value) : null;
                    if ($media instanceof Media && ! PwaIconFiles::usableSource($media)) {
                        $fail('لوگو باید تصویر PNG، WebP یا JPG با دست‌کم '.PwaIconFiles::MIN_SOURCE_SIZE.' پیکسل در هر ضلع باشد.');
                    }
                }),
        ];
    }

    public function save(UpdateSettings $update, SettingsRepository $settings): void
    {
        parent::save($update, $settings);

        $pwa = $settings->group(SettingGroup::Pwa);
        if (! $pwa instanceof PwaSettingsData || $pwa->iconMediaId === null) {
            return;
        }

        $media = Media::query()->find($pwa->iconMediaId);
        if (! $media instanceof Media) {
            return;
        }

        try {
            app(GeneratePwaIcons::class)->handle($media, $pwa->backgroundColor);
        } catch (Throwable $e) {
            report($e);
            Notification::make()->danger()->title('ساخت آیکن‌ها انجام نشد')->body('آیکن‌های پیش‌فرض نمایش داده می‌شوند.')->send();
        }
    }
}

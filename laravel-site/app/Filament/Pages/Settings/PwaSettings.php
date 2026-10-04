<?php

declare(strict_types=1);

namespace App\Filament\Pages\Settings;

use App\Domain\Media\Models\Media;
use App\Domain\Pwa\Actions\GeneratePwaIcons;
use App\Domain\Pwa\Support\PwaIconFiles;
use App\Domain\Pwa\Version\AppVersion;
use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\PwaSettings as PwaSettingsData;
use App\Domain\Settings\Enums\SettingGroup;
use App\Filament\Forms\Components\MediaPicker;
use BackedEnum;
use Closure;
use Filament\Actions\Action;
use Filament\Facades\Filament;
use Filament\Forms\Components\ColorPicker;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Infolists\Components\TextEntry;
use Filament\Notifications\Notification;
use Filament\Schemas\Components\Section;
use Filament\Support\Icons\Heroicon;
use Throwable;

/**
 * PWA settings (L8-01): manifest names, colours and the square logo the install icons are generated from. Saving
 * regenerates the icon set (GeneratePwaIcons) right away; the manifest cache is busted by SettingObserver.
 *
 * App updates (L8-02): the deployed build id (read-only), the minimum build (older builds get the blocking update
 * screen) and an optional update message — all served by /pwa/version.json. The «اجبار به به‌روزرسانی» header action
 * sets the minimum to the deployed build in one click.
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
            Section::make('به‌روزرسانی اپ')
                ->description('هر انتشار تازه، به بازدیدکننده‌ها پیام غیرمزاحم «نسخه جدید آماده است» نشان می‌دهد. فقط برای انتشارهای حیاتی (رفع مشکل امنیتی یا داده) حداقل نسخه را بالا ببر: نسخه‌های قدیمی‌تر تا به‌روزرسانی نکنند صفحه مسدودکننده می‌بینند.')
                ->schema([
                    TextEntry::make('deployed_build')
                        ->label('نسخه منتشرشده فعلی')
                        ->state(static fn (): string => app(AppVersion::class)->deployedBuildId() ?? 'هنوز بیلدی منتشر نشده است'),
                    TextInput::make('min_build_id')
                        ->label('حداقل نسخه مجاز (اجباری)')
                        ->placeholder('خالی = هیچ به‌روزرسانی اجباری')
                        ->maxLength(60)
                        ->regex(PwaSettingsData::BUILD_ID_PATTERN)
                        ->validationMessages(['regex' => 'شناسه نسخه باید به شکل 20261004120000-abc1234 باشد.'])
                        ->helperText('شناسه یک بیلد (زمان + کد). نسخه‌های قدیمی‌تر از این، صفحه به‌روزرسانی اجباری می‌بینند. بالاتر از نسخه منتشرشده اثری ندارد.'),
                    Textarea::make('update_message')
                        ->label('پیام به‌روزرسانی (اختیاری)')
                        ->rows(2)
                        ->maxLength(200)
                        ->helperText('به جای متن پیش‌فرض در پیام و صفحه به‌روزرسانی نمایش داده می‌شود.'),
                ]),
        ];
    }

    /**
     * @return array<Action>
     */
    protected function getHeaderActions(): array
    {
        return [
            Action::make('forceUpdate')
                ->label('اجبار به به‌روزرسانی')
                ->icon(Heroicon::OutlinedArrowPath)
                ->color('danger')
                ->requiresConfirmation()
                ->modalHeading('اجبار همه به نسخه فعلی؟')
                ->modalDescription('همه بازدیدکننده‌هایی که نسخه قدیمی‌تری باز کرده‌اند، تا به‌روزرسانی نکنند صفحه مسدودکننده می‌بینند. فقط برای انتشارهای حیاتی.')
                ->visible(static fn (): bool => self::canAccess() && app(AppVersion::class)->deployedBuildId() !== null)
                ->action(function (UpdateSettings $update, SettingsRepository $settings, AppVersion $version): void {
                    abort_unless(static::canAccess(), 403);
                    $build = $version->deployedBuildId();
                    if ($build === null) {
                        return;
                    }

                    $old = $settings->group(SettingGroup::Pwa)->toArray()['min_build_id'] ?? null;
                    $new = $update->handle(SettingGroup::Pwa, ['min_build_id' => $build])->toArray();
                    activity('settings')
                        ->causedBy(Filament::auth()->user())
                        ->event('updated')
                        ->withProperties(['group' => SettingGroup::Pwa->value, 'attributes' => ['min_build_id' => $build], 'old' => ['min_build_id' => $old]])
                        ->log('settings.'.SettingGroup::Pwa->value);

                    $this->form->fill($new);
                    Notification::make()->success()->title('به‌روزرسانی اجباری فعال شد.')->body('حداقل نسخه: '.$build)->send();
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

<?php

declare(strict_types=1);

namespace App\Filament\Pages\Settings;

use App\Domain\Media\Models\Media;
use App\Domain\Settings\Data\SeoDefaults;
use App\Domain\Settings\Enums\SettingGroup;
use App\Filament\Auth\AdminRole;
use App\Filament\Components\Seo\SerpMeasure;
use App\Filament\Forms\Components\MediaPicker;
use BackedEnum;
use Closure;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Infolists\Components\TextEntry;
use Filament\Schemas\Components\Section;
use Filament\Schemas\Components\Utilities\Get;
use Filament\Support\Icons\Heroicon;
use UnitEnum;

/**
 * Site-wide SEO defaults (L7-06, settings group `seo`): title template + separator, default title / description, the
 * default share image (OG 1200×630 variant from the media pipeline) and the X/Twitter handle. They feed SeoManager →
 * `<x-seo.head/>` (title, description, og:image, twitter:site) and the JSON-LD WebPage node. Verification codes,
 * robots and sitemaps live on the indexing page (L7-04) — only this page's keys are written, the rest of the group is
 * kept. Saved through UpdateSettings (SettingObserver bumps `settings`, `seo`, `pages`) and logged (`settings`).
 * Access: SEO managers + super-admins.
 */
final class SeoSettings extends SettingsPage
{
    public const OG_WIDTH = 1200;

    public const OG_HEIGHT = 630;

    protected static array $roles = [AdminRole::SeoManager];

    protected static string|UnitEnum|null $navigationGroup = 'سئو';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedAdjustmentsHorizontal;

    protected static ?string $navigationLabel = 'پیش‌فرض‌های سئو';

    protected static ?string $title = 'پیش‌فرض‌های سئو';

    protected static ?string $slug = 'settings/seo';

    protected static ?int $navigationSort = 40;

    protected static function group(): SettingGroup
    {
        return SettingGroup::Seo;
    }

    protected function fields(): array
    {
        return [
            Section::make('عنوان صفحه‌ها')
                ->description('عنوان هر صفحه از این الگو ساخته می‌شود؛ صفحه‌هایی که عنوان کامل دارند (مثل صفحه اصلی) از الگو عبور نمی‌کنند.')
                ->schema([
                    TextInput::make('title_template')
                        ->label('الگوی عنوان')
                        ->required()
                        ->maxLength(120)
                        ->live(debounce: 400)
                        ->rule(static fn (): Closure => static function (string $attribute, mixed $value, Closure $fail): void {
                            if (! str_contains(str_replace(SeoDefaults::SEPARATOR_TOKEN, '', (string) $value), '%s')) {
                                $fail('الگو باید %s (جای عنوان صفحه) را داشته باشد.');
                            }
                        })
                        ->helperText('%s = عنوان صفحه، %sep% = جداکننده. مثال: «%s %sep% ریتمی».'),
                    TextInput::make('separator')
                        ->label('جداکننده')
                        ->required()
                        ->maxLength(5)
                        ->live(debounce: 400)
                        ->helperText('مثلاً — یا | یا ·'),
                    TextInput::make('default_title')
                        ->label('عنوان پیش‌فرض')
                        ->required()
                        ->maxLength(120)
                        ->helperText('وقتی صفحه‌ای عنوانی ندارد (و برای صفحه اصلی) استفاده می‌شود.'),
                    TextEntry::make('title_preview')
                        ->label('پیش‌نمایش')
                        ->state(static fn (Get $get): string => self::preview($get)),
                ]),
            Section::make('توضیح و تصویر اشتراک')->schema([
                Textarea::make('default_description')
                    ->label('توضیح متای پیش‌فرض')
                    ->rows(3)
                    ->maxLength(300)
                    ->live(debounce: 500)
                    ->hint(static fn (?string $state): string => SerpMeasure::hint($state, SerpMeasure::DESCRIPTION_FONT_PX, SerpMeasure::DESCRIPTION_MAX_PX))
                    ->hintColor(static fn (?string $state): string => SerpMeasure::status($state, SerpMeasure::DESCRIPTION_CHARS, SerpMeasure::DESCRIPTION_FONT_PX, SerpMeasure::DESCRIPTION_MAX_PX))
                    ->helperText('برای صفحه‌هایی که توضیح خودشان را ندارند؛ ۷۰ تا ۱۶۰ نویسه.'),
                MediaPicker::make('default_og_media_id')
                    ->label('تصویر اشتراک پیش‌فرض (Open Graph)')
                    ->helperText('دست‌کم '.SerpMeasure::digits(self::OG_WIDTH).'×'.SerpMeasure::digits(self::OG_HEIGHT).' پیکسل؛ نسخه ۱۲۰۰×۶۳۰ برای تلگرام، واتس‌اپ، X و گوگل خودکار ساخته می‌شود.')
                    ->rule(static fn (): Closure => static function (string $attribute, mixed $value, Closure $fail): void {
                        $media = is_numeric($value) ? Media::query()->find((int) $value) : null;
                        if ($media instanceof Media && (! str_starts_with($media->mime, 'image/') || (int) $media->width < self::OG_WIDTH || (int) $media->height < self::OG_HEIGHT)) {
                            $fail('تصویر اشتراک باید دست‌کم '.SerpMeasure::digits(self::OG_WIDTH).'×'.SerpMeasure::digits(self::OG_HEIGHT).' پیکسل باشد.');
                        }
                    }),
                TextInput::make('twitter_handle')
                    ->label('حساب X (توییتر)')
                    ->prefix('@')
                    ->maxLength(16)
                    ->regex('/^@?[A-Za-z0-9_]{1,15}$/')
                    ->validationMessages(['regex' => 'فقط حروف انگلیسی، عدد و _ (حداکثر ۱۵ نویسه).'])
                    ->dehydrateStateUsing(static fn (?string $state): ?string => $state === null || trim($state) === '' ? null : ltrim(trim($state), '@'))
                    ->helperText('در twitter:site همه صفحه‌ها می‌آید؛ خالی = بدون آن.'),
            ]),
        ];
    }

    private static function preview(Get $get): string
    {
        $template = (string) $get('title_template');
        $separator = trim((string) $get('separator'));
        if (! str_contains(str_replace(SeoDefaults::SEPARATOR_TOKEN, '', $template), '%s')) {
            return 'الگو باید %s داشته باشد.';
        }

        $title = str_replace(['%sep%', '%s'], [$separator, 'راهنمای چرخه قاعدگی'], $template);

        return $title.' — '.SerpMeasure::hint($title, SerpMeasure::TITLE_FONT_PX, SerpMeasure::TITLE_MAX_PX);
    }
}

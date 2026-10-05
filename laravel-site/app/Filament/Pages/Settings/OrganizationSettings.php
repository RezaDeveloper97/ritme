<?php

declare(strict_types=1);

namespace App\Filament\Pages\Settings;

use App\Domain\Media\Models\Media;
use App\Domain\Settings\Enums\SettingGroup;
use App\Filament\Auth\AdminRole;
use App\Filament\Forms\Components\MediaPicker;
use BackedEnum;
use Closure;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\TagsInput;
use Filament\Forms\Components\TextInput;
use Filament\Schemas\Components\Section;
use Filament\Support\Icons\Heroicon;
use UnitEnum;

/**
 * The publisher behind the site (L7-06, settings group `organization`): legal name, logo, founding date, extra sameAs
 * profiles and the contact point. Feeds the JSON-LD Organization node on every page (OrganizationNode / PageGraph;
 * social URLs from the social settings are merged into sameAs, `[…]` placeholders are never emitted). Saved through
 * UpdateSettings (SettingObserver bumps `settings`, `seo`, `pages`) and logged (`settings`).
 * Access: SEO managers + super-admins.
 */
final class OrganizationSettings extends SettingsPage
{
    /** Google: Organization logo at least 112×112. */
    public const LOGO_MIN = 112;

    public const CONTACT_TYPES = [
        'customer support' => 'پشتیبانی کاربران',
        'customer service' => 'خدمات مشتریان',
        'technical support' => 'پشتیبانی فنی',
        'sales' => 'فروش',
    ];

    protected static array $roles = [AdminRole::SeoManager];

    protected static string|UnitEnum|null $navigationGroup = 'سئو';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedBuildingOffice2;

    protected static ?string $navigationLabel = 'اطلاعات سازمان';

    protected static ?string $title = 'اطلاعات سازمان (داده ساختاریافته)';

    protected static ?string $slug = 'settings/organization';

    protected static ?int $navigationSort = 45;

    protected static function group(): SettingGroup
    {
        return SettingGroup::Organization;
    }

    protected function fields(): array
    {
        return [
            Section::make('سازمان')
                ->description('در داده ساختاریافته (JSON-LD) همه صفحه‌ها به‌عنوان ناشر سایت می‌آید. فقط اطلاعات واقعی وارد کنید.')
                ->schema([
                    TextInput::make('legal_name')->label('نام حقوقی')->required()->maxLength(191),
                    MediaPicker::make('logo_media_id')
                        ->label('لوگو')
                        ->helperText('مربعی یا افقی، دست‌کم ۱۱۲×۱۱۲ پیکسل، PNG / WebP / JPG.')
                        ->rule(static fn (): Closure => static function (string $attribute, mixed $value, Closure $fail): void {
                            $media = is_numeric($value) ? Media::query()->find((int) $value) : null;
                            if ($media instanceof Media && (! str_starts_with($media->mime, 'image/') || (int) $media->width < self::LOGO_MIN || (int) $media->height < self::LOGO_MIN)) {
                                $fail('لوگو باید تصویری با دست‌کم ۱۱۲×۱۱۲ پیکسل باشد.');
                            }
                        }),
                    TextInput::make('founding_date')
                        ->label('تاریخ تأسیس (میلادی)')
                        ->placeholder('2024 یا 2024-03-21')
                        ->maxLength(10)
                        ->regex('/^\d{4}(-(0[1-9]|1[0-2])(-(0[1-9]|[12]\d|3[01]))?)?$/')
                        ->validationMessages(['regex' => 'به شکل سال (2024)، سال-ماه (2024-03) یا سال-ماه-روز (2024-03-21) میلادی.'])
                        ->extraInputAttributes(['dir' => 'ltr']),
                    TagsInput::make('same_as')
                        ->label('پروفایل‌های دیگر (sameAs)')
                        ->placeholder('https://…')
                        ->nestedRecursiveRules(['url:http,https', 'max:255'])
                        ->helperText('نشانی کامل با http یا https؛ هر کدام با Enter. صفحه‌های شبکه‌های اجتماعی از تنظیمات شبکه‌های اجتماعی خودکار اضافه می‌شوند.'),
                ]),
            Section::make('راه تماس (contactPoint)')
                ->description('خالی = از تنظیمات تماس سایت استفاده می‌شود.')
                ->statePath('contact_point')
                ->schema([
                    Select::make('contact_type')->label('نوع')->options(self::CONTACT_TYPES)->required()->native(false),
                    TextInput::make('telephone')
                        ->label('تلفن')
                        ->tel()
                        ->maxLength(20)
                        ->regex('/^\+?[0-9][0-9 \-]{5,19}$/')
                        ->validationMessages(['regex' => 'شماره با ارقام انگلیسی، مثلاً +982112345678.'])
                        ->extraInputAttributes(['dir' => 'ltr']),
                    TextInput::make('email')->label('ایمیل')->email()->maxLength(191)->extraInputAttributes(['dir' => 'ltr']),
                ]),
        ];
    }
}

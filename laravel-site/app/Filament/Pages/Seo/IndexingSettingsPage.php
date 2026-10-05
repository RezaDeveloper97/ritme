<?php

declare(strict_types=1);

namespace App\Filament\Pages\Seo;

use App\Domain\Seo\Indexing\Actions\RegenerateIndexNowKey;
use App\Domain\Seo\Indexing\Actions\RegenerateSitemaps;
use App\Domain\Seo\Indexing\Actions\ResetRobotsTxt;
use App\Domain\Seo\Indexing\Actions\SaveIndexingSettings;
use App\Domain\Seo\Indexing\HeadCode;
use App\Domain\Seo\Indexing\IndexingType;
use App\Domain\Seo\Indexing\IndexNow\IndexNow;
use App\Domain\Seo\Indexing\InvalidIndexingSettings;
use App\Domain\Seo\Indexing\RobotsTxtValidator;
use App\Domain\Seo\Indexing\SettingsRobotsRules;
use App\Domain\Seo\Sitemap\SitemapUrl;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\SeoDefaults;
use App\Domain\Settings\Enums\SettingGroup;
use App\Filament\Auth\AdminAccess;
use App\Filament\Auth\AdminRole;
use BackedEnum;
use Closure;
use Filament\Actions\Action;
use Filament\Facades\Filament;
use Filament\Forms\Components\CheckboxList;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\TagsInput;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Notifications\Notification;
use Filament\Pages\Page;
use Filament\Schemas\Components\Actions;
use Filament\Schemas\Components\Callout;
use Filament\Schemas\Components\EmbeddedSchema;
use Filament\Schemas\Components\Form;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Components\Html;
use Filament\Schemas\Components\Section;
use Filament\Schemas\Components\Tabs;
use Filament\Schemas\Components\Tabs\Tab;
use Filament\Schemas\Components\Text;
use Filament\Schemas\Components\Utilities\Get;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Illuminate\Support\HtmlString;
use Illuminate\Validation\ValidationException;
use UnitEnum;

/**
 * Indexing controls (L7-04): robots.txt editor (validated, live production preview, reset), per-type robots defaults,
 * sitemap settings (exclude / priority / changefreq / ping list) + "regenerate", search-engine verification codes,
 * IndexNow switch + key, and the super-admin-only head code. Access: SEO managers + super-admins (deny by default).
 * Writes go through SaveIndexingSettings (settings `seo` group → caches bumped by the settings observers); every
 * change is recorded in the activity log (`seo`).
 *
 * @property-read Schema $form
 */
final class IndexingSettingsPage extends Page
{
    /** Verification codes edited here (other keys already stored are kept). */
    public const ENGINES = [
        'google' => 'Google Search Console',
        'bing' => 'Bing Webmaster Tools',
        'yandex' => 'Yandex Webmaster',
    ];

    /**
     * @var array<string, mixed>|null
     */
    public ?array $data = [];

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedGlobeAlt;

    protected static string|UnitEnum|null $navigationGroup = 'سئو';

    protected static ?int $navigationSort = 30;

    protected static ?string $navigationLabel = 'کنترل ایندکس';

    protected static ?string $title = 'کنترل ایندکس و خزش';

    protected static ?string $slug = 'seo/indexing';

    public static function canAccess(): bool
    {
        return AdminAccess::allows(Filament::auth()->user(), [AdminRole::SeoManager]);
    }

    public static function canEditHeadCode(): bool
    {
        return AdminAccess::allows(Filament::auth()->user());
    }

    public function mount(): void
    {
        abort_unless(self::canAccess(), 403);
        $this->fillForm();
    }

    public function form(Schema $schema): Schema
    {
        return $schema->statePath('data')->components([
            Tabs::make('indexing')->persistTabInQueryString()->tabs([
                $this->robotsTab(),
                $this->typesTab(),
                $this->sitemapTab(),
                $this->verificationTab(),
                $this->indexNowTab(),
                $this->headCodeTab(),
            ]),
        ]);
    }

    public function content(Schema $schema): Schema
    {
        return $schema->components([
            Form::make([EmbeddedSchema::make('form')])
                ->id('form')
                ->livewireSubmitHandler('save')
                ->footer([
                    Actions::make([
                        Action::make('save')->label('ذخیره')->submit('save')->keyBindings(['mod+s']),
                    ])->key('form-actions'),
                ]),
        ]);
    }

    public function save(SaveIndexingSettings $save): void
    {
        abort_unless(self::canAccess(), 403);

        /** @var array<string, mixed> $state */
        $state = $this->form->getState();

        try {
            $changes = $save->handle($this->toSettings($state), self::canEditHeadCode());
        } catch (InvalidIndexingSettings $e) {
            $messages = [];
            foreach ($e->errors() as $key => $errors) {
                $messages['data.'.$key] = $errors;
            }
            throw ValidationException::withMessages($messages);
        }

        if ($changes['new'] !== []) {
            $this->log('seo.indexing.updated', $changes);
        }

        $this->fillForm();
        Notification::make()->success()->title('تنظیمات ایندکس ذخیره شد.')->send();
    }

    private function robotsTab(): Tab
    {
        return Tab::make('robots.txt')->icon(Heroicon::OutlinedDocumentText)->schema([
            Callout::make('فقط روی محیط production اعمال می‌شود')
                ->info()
                ->description('روی لوکال و استیج robots.txt همیشه «Disallow: /» است تا نسخه آزمایشی ایندکس نشود. خط Sitemap خودکار اضافه می‌شود. خالی گذاشتن یعنی قانون‌های پیش‌فرض سایت.'),
            Textarea::make('robots_txt')
                ->label('قانون‌های robots.txt')
                ->rows(14)
                ->placeholder(fn (): string => app(SettingsRobotsRules::class)->defaults())
                ->extraInputAttributes(['dir' => 'ltr', 'class' => 'font-mono text-sm'])
                ->live(onBlur: true)
                ->rules([self::rule(static fn (string $value): array => app(RobotsTxtValidator::class)->errors($value))]),
            Section::make('پیش‌نمایش robots.txt در production')
                ->compact()
                ->schema([
                    Html::make(fn (Get $get): HtmlString => $this->robotsPreview($get('robots_txt'))),
                ]),
            Actions::make([
                Action::make('resetRobots')
                    ->label('بازگشت به قانون‌های پیش‌فرض')
                    ->icon(Heroicon::OutlinedArrowUturnLeft)
                    ->color('gray')
                    ->requiresConfirmation()
                    ->modalDescription('قانون‌های سفارشی robots.txt حذف و قانون‌های پیش‌فرض سایت جایگزین می‌شوند.')
                    ->action(function (ResetRobotsTxt $reset): void {
                        abort_unless(static::canAccess(), 403);
                        $old = $this->seo()->indexing->robotsTxt;
                        $reset->handle();
                        $this->log('seo.robots.reset', ['old' => ['robots_txt' => $old], 'new' => ['robots_txt' => null]]);
                        $this->fillForm();
                        Notification::make()->success()->title('robots.txt به حالت پیش‌فرض برگشت.')->send();
                    }),
            ])->key('robotsActions'),
        ]);
    }

    private function typesTab(): Tab
    {
        $fields = [];
        foreach (IndexingType::robotsTypeOptions() as $type => $label) {
            $fields[] = Select::make('robots_types.'.self::formKey($type))
                ->label($label)
                ->options(IndexingType::ROBOTS_OPTIONS)
                ->placeholder('پیش‌فرض سایت (ایندکس شود)');
        }

        return Tab::make('robots هر نوع محتوا')->icon(Heroicon::OutlinedTag)->schema([
            Text::make('مقدار پیش‌فرض متای robots برای هر نوع صفحه. تنظیم سئوی خودِ هر آیتم (یا تصمیم صفحه، مثل فهرست خالی) بر این مقدار مقدم است. نوعی که «ایندکس نشود» باشد از نقشه سایت هم حذف می‌شود.'),
            Grid::make(2)->schema($fields),
        ]);
    }

    private function sitemapTab(): Tab
    {
        $rows = [];
        foreach (IndexingType::options() as $type => $label) {
            $key = self::formKey($type);
            $rows[] = Grid::make(3)->schema([
                Text::make($label),
                TextInput::make('sitemap_priorities.'.$key)
                    ->label('اولویت')
                    ->numeric()->minValue(0)->maxValue(1)->step(0.1)
                    ->placeholder('مقدار خودکار'),
                Select::make('sitemap_changefreq.'.$key)
                    ->label('بسامد تغییر')
                    ->options(IndexingType::CHANGEFREQ_OPTIONS)
                    ->placeholder('مقدار خودکار'),
            ]);
        }

        return Tab::make('نقشه سایت')->icon(Heroicon::OutlinedMap)->schema([
            CheckboxList::make('sitemap_exclude')
                ->label('حذف از نقشه سایت')
                ->helperText('نوع‌های علامت‌خورده در /sitemap.xml نمی‌آیند و فایلشان ۴۰۴ می‌شود.')
                ->options(IndexingType::options())
                ->columns(3),
            Section::make('اولویت و بسامد تغییر')
                ->description('خالی یعنی مقداری که هر بخش خودش تعیین می‌کند (برای هر آیتم از تب سئوی آن).')
                ->collapsible()
                ->schema($rows),
            TagsInput::make('sitemap_ping_urls')
                ->label('فهرست پینگ')
                ->placeholder('https://example.com/ping?sitemap={sitemap}')
                ->helperText('پس از «بازسازی نقشه سایت» از سرور (صف) و فقط در production فراخوانی می‌شوند؛ {sitemap} با نشانی نقشه سایت جایگزین می‌شود. گوگل و بینگ پینگ را کنار گذاشته‌اند؛ برای بینگ و یاندکس از IndexNow استفاده کنید.')
                ->nestedRecursiveRules(['url', 'starts_with:https://']),
            Actions::make([
                Action::make('regenerateSitemap')
                    ->label('بازسازی نقشه سایت')
                    ->icon(Heroicon::OutlinedArrowPath)
                    ->requiresConfirmation()
                    ->modalDescription('کش همه فایل‌های نقشه سایت و robots.txt پاک و نمایه از نو ساخته می‌شود.')
                    ->action(function (RegenerateSitemaps $regenerate): void {
                        abort_unless(static::canAccess(), 403);
                        $files = $regenerate->handle();
                        $this->log('seo.sitemap.regenerated', ['old' => [], 'new' => ['files' => $files]]);
                        Notification::make()->success()->title('نقشه سایت بازسازی شد ('.$files.' فایل).')->send();
                    }),
                Action::make('viewSitemap')
                    ->label('مشاهده /sitemap.xml')
                    ->color('gray')
                    ->icon(Heroicon::OutlinedArrowTopRightOnSquare)
                    ->url(fn (): string => route('sitemap.index'))
                    ->openUrlInNewTab(),
            ])->key('sitemapActions'),
        ]);
    }

    private function verificationTab(): Tab
    {
        $fields = [];
        foreach (self::ENGINES as $engine => $label) {
            $fields[] = TextInput::make('verification.'.$engine)
                ->label($label)
                ->maxLength(200)
                ->regex('/^[A-Za-z0-9_\-.=+\/:]+$/')
                ->extraInputAttributes(['dir' => 'ltr'])
                ->helperText('فقط مقدار content متا را وارد کنید، نه کل تگ.');
        }

        return Tab::make('تأیید مالکیت')->icon(Heroicon::OutlinedCheckBadge)->schema([
            Text::make('کدها به صورت تگ <meta> در head همه صفحه‌ها قرار می‌گیرند (بدون هیچ درخواست بیرونی).'),
            ...$fields,
        ]);
    }

    private function indexNowTab(): Tab
    {
        return Tab::make('IndexNow')->icon(Heroicon::OutlinedBolt)->schema([
            Callout::make('ارسال از سمت سرور')
                ->info()
                ->description('با روشن بودن، انتشار، ویرایش، عدم انتشار یا حذف مقاله، محصول و مکان از طریق صف به api.indexnow.org (بینگ، یاندکس و …) اعلام می‌شود. این یک درخواست خروجی از سرور است و صفحه‌های عمومی همچنان هیچ درخواست بیرونی ندارند. فقط در production ارسال می‌شود.'),
            Toggle::make('indexnow_enabled')->label('IndexNow فعال باشد'),
            Text::make(function (): string {
                $indexNow = app(IndexNow::class);
                $location = $indexNow->keyLocation();

                return $location === null
                    ? 'کلید پس از فعال‌سازی و ذخیره ساخته می‌شود.'
                    : 'فایل کلید: '.$location.($indexNow->active() ? '' : ' — (ارسال فقط در production)');
            })->extraAttributes(['dir' => 'auto']),
            Actions::make([
                Action::make('regenerateIndexNowKey')
                    ->label('ساخت کلید تازه')
                    ->icon(Heroicon::OutlinedKey)
                    ->color('gray')
                    ->requiresConfirmation()
                    ->modalDescription('فایل کلید فعلی دیگر پاسخ نمی‌دهد و کلید تازه جایگزین می‌شود.')
                    ->visible(fn (): bool => $this->seo()->indexing->indexNowKey !== null)
                    ->action(function (RegenerateIndexNowKey $regenerate): void {
                        abort_unless(static::canAccess(), 403);
                        $regenerate->handle();
                        $this->log('seo.indexnow.key_regenerated', ['old' => [], 'new' => []]);
                        Notification::make()->success()->title('کلید IndexNow تازه شد.')->send();
                    }),
            ])->key('indexNowActions'),
        ]);
    }

    private function headCodeTab(): Tab
    {
        return Tab::make('کد سفارشی head')
            ->icon(Heroicon::OutlinedCodeBracket)
            ->visible(static fn (): bool => self::canEditHeadCode())
            ->schema([
                Callout::make('هشدار: فقط مدیر کل')
                    ->danger()
                    ->description('اسکریپت‌ها و استایل‌های بیرونی (آنالیتیکس، چت، فونت و …) امتیاز GTmetrix/Lighthouse را پایین می‌آورند و با CSP سخت‌گیرانه سایت مسدود می‌شوند. CSP عمداً از این فرم باز نمی‌شود و سایت هیچ درخواست بیرونی نمی‌فرستد؛ بنابراین فقط تگ‌های <meta> و <link> هم‌دامنه پذیرفته می‌شوند و بقیه رد می‌شوند.'),
                Textarea::make('head_code')
                    ->label('تگ‌های اضافه head')
                    ->rows(8)
                    ->placeholder('<meta name="example-verification" content="…">')
                    ->extraInputAttributes(['dir' => 'ltr', 'class' => 'font-mono text-sm'])
                    ->rules([self::rule(static fn (string $value): array => app(HeadCode::class)->errors($value))]),
            ]);
    }

    private function robotsPreview(mixed $rules): HtmlString
    {
        $rules = is_string($rules) ? trim($rules) : '';
        $errors = $rules === '' ? [] : app(RobotsTxtValidator::class)->errors($rules);
        $body = ($rules === '' ? app(SettingsRobotsRules::class)->defaults() : str_replace("\r\n", "\n", $rules))
            ."\n\nSitemap: ".app(SitemapUrl::class)->file('/sitemap.xml');

        $html = '<pre dir="ltr" class="overflow-x-auto whitespace-pre-wrap font-mono text-sm">'.e($body).'</pre>';
        if ($errors !== []) {
            $html .= '<ul class="mt-2 text-sm text-danger-600">'.implode('', array_map(static fn (string $error): string => '<li>'.e($error).'</li>', $errors)).'</ul>';
        }

        return new HtmlString($html);
    }

    private function fillForm(): void
    {
        $seo = $this->seo();
        $indexing = $seo->indexing;

        $this->form->fill([
            'robots_txt' => $indexing->robotsTxt,
            'robots_types' => self::toForm($indexing->robotsTypes),
            'sitemap_exclude' => $indexing->sitemapExclude,
            'sitemap_priorities' => self::toForm(array_map(static fn (float $p): string => number_format($p, 1, '.', ''), $indexing->sitemapPriorities)),
            'sitemap_changefreq' => self::toForm($indexing->sitemapChangefreq),
            'sitemap_ping_urls' => $indexing->sitemapPingUrls,
            'verification' => array_intersect_key($seo->verification, self::ENGINES),
            'indexnow_enabled' => $indexing->indexNowEnabled,
            'head_code' => $indexing->headCode,
        ]);
    }

    /**
     * Form state → setting values (form keys use `_`, sitemap keys `-`; other verification codes are kept).
     *
     * @param  array<string, mixed>  $state
     * @return array<string, mixed>
     */
    private function toSettings(array $state): array
    {
        $map = static function (mixed $values): array {
            $out = [];
            foreach (is_array($values) ? $values : [] as $key => $value) {
                if ($value !== null && $value !== '') {
                    $out[str_replace('_', '-', (string) $key)] = is_scalar($value) ? (string) $value : $value;
                }
            }

            return $out;
        };

        $verification = array_diff_key($this->seo()->verification, self::ENGINES);
        foreach (is_array($state['verification'] ?? null) ? $state['verification'] : [] as $engine => $code) {
            if (is_string($code) && trim($code) !== '') {
                $verification[(string) $engine] = trim($code);
            }
        }

        $values = [
            'robots_txt' => is_string($state['robots_txt'] ?? null) ? $state['robots_txt'] : null,
            'robots_types' => $map($state['robots_types'] ?? []),
            'sitemap_exclude' => array_values(array_filter((array) ($state['sitemap_exclude'] ?? []), 'is_string')),
            'sitemap_priorities' => $map($state['sitemap_priorities'] ?? []),
            'sitemap_changefreq' => $map($state['sitemap_changefreq'] ?? []),
            'sitemap_ping_urls' => array_values(array_filter((array) ($state['sitemap_ping_urls'] ?? []), 'is_string')),
            'verification' => $verification,
            'indexnow_enabled' => (bool) ($state['indexnow_enabled'] ?? false),
        ];
        if (self::canEditHeadCode()) {
            $values['head_code'] = is_string($state['head_code'] ?? null) ? $state['head_code'] : null;
        }

        return $values;
    }

    /**
     * @param  array<string, mixed>  $changes  ['old' => …, 'new' => …]
     */
    private function log(string $description, array $changes): void
    {
        activity('seo')
            ->causedBy(Filament::auth()->user())
            ->event('updated')
            ->withProperties(['group' => SettingGroup::Seo->value, 'attributes' => $changes['new'] ?? [], 'old' => $changes['old'] ?? []])
            ->log($description);
    }

    private function seo(): SeoDefaults
    {
        $seo = app(SettingsRepository::class)->group(SettingGroup::Seo);
        assert($seo instanceof SeoDefaults);

        return $seo;
    }

    private static function formKey(string $type): string
    {
        return str_replace('-', '_', $type);
    }

    /**
     * @param  array<string, mixed>  $values
     * @return array<string, mixed>
     */
    private static function toForm(array $values): array
    {
        $out = [];
        foreach ($values as $key => $value) {
            $out[self::formKey($key)] = $value;
        }

        return $out;
    }

    /**
     * A Laravel validation rule closure from a validator returning Persian messages.
     *
     * @param  Closure(string): list<string>  $errors
     */
    private static function rule(Closure $errors): Closure
    {
        // Filament evaluates closures in rules(); the returned closure is the Laravel rule.
        return static fn (): Closure => static function (string $attribute, mixed $value, Closure $fail) use ($errors): void {
            if (is_string($value) && trim($value) !== '') {
                foreach ($errors($value) as $error) {
                    $fail($error);
                }
            }
        };
    }
}

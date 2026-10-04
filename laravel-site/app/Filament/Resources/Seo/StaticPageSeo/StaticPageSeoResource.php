<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\StaticPageSeo;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Actions\ListStaticPageSeo;
use App\Domain\Seo\Actions\ResetStaticPageSeo;
use App\Domain\Seo\Data\StaticPageSeoData;
use App\Domain\Seo\Models\SeoMeta;
use App\Filament\Components\Seo\SerpMeasure;
use App\Filament\Resources\Seo\StaticPageSeo\Pages\EditStaticPageSeo;
use App\Filament\Resources\Seo\StaticPageSeo\Pages\ListStaticPageSeoPages;
use BackedEnum;
use Filament\Actions\Action;
use Filament\Facades\Filament;
use Filament\Notifications\Notification;
use Filament\Resources\Resource;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Table;
use UnitEnum;

/**
 * SEO of the marketing pages (L7-01). Static pages have no model: the list is the StaticPage registry (indexable
 * pages, with or without a seo_meta row) built by ListStaticPageSeo; editing writes the page's `seo_meta` row (by
 * route name) through SaveStaticPageSeo, "reset" deletes it (ResetStaticPageSeo). Both bump `seo` + `sitemap` +
 * `pages` (SeoMetaObserver), so the live page shows the change on its next request. Access: StaticPageSeoPolicy
 * (SEO manager + super-admin).
 */
final class StaticPageSeoResource extends Resource
{
    protected static ?string $model = SeoMeta::class;

    protected static ?string $slug = 'seo/static-pages';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedMagnifyingGlass;

    protected static string|UnitEnum|null $navigationGroup = 'سئو';

    protected static ?int $navigationSort = 10;

    protected static ?string $navigationLabel = 'سئوی صفحه‌های ثابت';

    protected static ?string $modelLabel = 'سئوی صفحه';

    protected static ?string $pluralModelLabel = 'سئوی صفحه‌های ثابت';

    /** URL-safe key of a page (route name with dots → dashes). */
    public static function key(StaticPage $page): string
    {
        return str_replace('.', '-', $page->routeName());
    }

    public static function pageFor(string $key): ?StaticPage
    {
        foreach (ListStaticPageSeo::pages() as $page) {
            if (self::key($page) === $key) {
                return $page;
            }
        }

        return null;
    }

    /** The policy decision for one page (its seo_meta row, or a new one). */
    public static function canManage(StaticPage $page): bool
    {
        $user = Filament::auth()->user();

        return $user !== null && $user->can('update', new SeoMeta(['route_name' => $page->routeName()]));
    }

    public static function table(Table $table): Table
    {
        return $table
            ->records(static function (?string $search): array {
                $records = [];
                foreach (app(ListStaticPageSeo::class)->handle() as $row) {
                    $record = self::rowFor($row);
                    if (self::matches($record, $search)) {
                        $records[$record['key']] = $record;
                    }
                }

                return $records;
            })
            ->searchable()
            ->paginated(false)
            ->columns([
                TextColumn::make('label')
                    ->label('صفحه')
                    ->weight('medium')
                    ->description(static fn (array $record): string => (string) $record['path']),
                TextColumn::make('title')
                    ->label('عنوان مؤثر')
                    ->wrap()
                    ->description(static fn (array $record): string => ($record['title_overridden'] ? 'سفارشی' : 'پیش‌فرض').' · '.SerpMeasure::digits((string) $record['title_length']).' نویسه'),
                TextColumn::make('description')
                    ->label('توضیح مؤثر')
                    ->wrap()
                    ->lineClamp(2)
                    ->description(static fn (array $record): string => ($record['description_overridden'] ? 'سفارشی' : 'پیش‌فرض').' · '.SerpMeasure::digits((string) $record['description_length']).' نویسه'),
                IconColumn::make('title_length_ok')
                    ->label('طول عنوان')
                    ->boolean()
                    ->tooltip('۳۰ تا ۶۰ نویسه'),
                IconColumn::make('description_length_ok')
                    ->label('طول توضیح')
                    ->boolean()
                    ->tooltip('۷۰ تا ۱۶۰ نویسه'),
                IconColumn::make('unique')
                    ->label('یکتا')
                    ->boolean()
                    ->tooltip('عنوان و توضیح با صفحه دیگری تکراری نیست'),
                IconColumn::make('og_image_set')
                    ->label('تصویر اشتراک')
                    ->boolean()
                    ->falseColor('warning')
                    ->tooltip(static fn (array $record): string => $record['og_image_set'] ? 'تصویر اختصاصی انتخاب شده' : 'تصویر پیش‌فرض سایت استفاده می‌شود'),
                IconColumn::make('indexable')
                    ->label('قابل نمایه')
                    ->boolean()
                    ->tooltip(static fn (array $record): string => $record['indexable'] ? 'index' : 'noindex'),
            ])
            ->recordUrl(static fn (array $record): string => self::getUrl('edit', ['page' => $record['key']]))
            ->recordActions([
                Action::make('live')
                    ->label('مشاهده زنده')
                    ->icon(Heroicon::OutlinedArrowTopRightOnSquare)
                    ->url(static fn (array $record): string => (string) $record['url'])
                    ->openUrlInNewTab(),
                self::resetAction()
                    ->visible(static fn (array $record): bool => (bool) $record['has_overrides']),
            ]);
    }

    /**
     * "Reset to defaults" for a table row (array record) or the edit page (page passed in).
     */
    public static function resetAction(?StaticPage $page = null): Action
    {
        $resolve = static fn (mixed $record): ?StaticPage => $page ?? (is_array($record) ? self::pageFor((string) ($record['key'] ?? '')) : null);

        return Action::make('reset')
            ->label('بازگشت به پیش‌فرض')
            ->icon(Heroicon::OutlinedArrowUturnRight)
            ->color('danger')
            ->requiresConfirmation()
            ->modalHeading('بازگشت به پیش‌فرض')
            ->modalDescription('همه تنظیمات سئوی این صفحه پاک می‌شود و عنوان و توضیح پیش‌فرض صفحه دوباره استفاده می‌شود.')
            ->modalSubmitActionLabel('بازگشت به پیش‌فرض')
            ->authorize(static fn (mixed $record = null): bool => ($target = $resolve($record)) !== null && self::canManage($target))
            ->action(static function (mixed $record = null) use ($resolve): void {
                $target = $resolve($record);
                abort_if($target === null || ! self::canManage($target), 403);

                if (app(ResetStaticPageSeo::class)->handle($target)) {
                    activity('seo')
                        ->causedBy(Filament::auth()->user())
                        ->event('deleted')
                        ->withProperties(['route' => $target->routeName()])
                        ->log('seo.static_page.reset');
                }

                Notification::make()->success()->title('تنظیمات سئوی «'.$target->label().'» به پیش‌فرض برگشت.')->send();
            });
    }

    public static function canCreate(): bool
    {
        return false;
    }

    public static function getPages(): array
    {
        return [
            'index' => ListStaticPageSeoPages::route('/'),
            'edit' => EditStaticPageSeo::route('/{page}'),
        ];
    }

    /**
     * @param  array<string, mixed>  $record
     */
    private static function matches(array $record, ?string $search): bool
    {
        $search = trim((string) $search);
        if ($search === '') {
            return true;
        }

        foreach (['label', 'path', 'route', 'title', 'description'] as $field) {
            if (mb_stripos((string) $record[$field], $search) !== false) {
                return true;
            }
        }

        return false;
    }

    /**
     * @return array<string, mixed>
     */
    public static function rowFor(StaticPageSeoData $row): array
    {
        return ['key' => self::key($row->page)] + $row->toArray();
    }
}

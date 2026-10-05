<?php

declare(strict_types=1);

namespace App\Filament\Pages\Seo;

use App\Domain\Seo\Actions\BulkSaveSeoMeta;
use App\Domain\Seo\Actions\ExportBulkSeoCsv;
use App\Domain\Seo\Actions\ImportBulkSeoCsv;
use App\Domain\Seo\Actions\ListBulkSeoRows;
use App\Domain\Seo\Actions\RecomputeSeoScores;
use App\Domain\Seo\Actions\ResetSeoMeta;
use App\Domain\Seo\Actions\ResolveSeoTarget;
use App\Domain\Seo\Analysis\SeoAnalysis;
use App\Domain\Seo\Analysis\TextWidth;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Filament\Auth\AdminAccess;
use App\Filament\Auth\AdminRole;
use App\Filament\Components\Seo\SerpMeasure;
use App\Filament\Resources\Blog\Categories\CategoryResource as BlogCategoryResource;
use App\Filament\Resources\Blog\Posts\PostResource;
use App\Filament\Resources\Directory\Places\PlaceResource;
use App\Filament\Resources\Seo\StaticPageSeo\StaticPageSeoResource;
use App\Filament\Resources\Shop\Categories\ShopCategoryResource;
use App\Filament\Resources\Shop\Products\ProductResource;
use BackedEnum;
use Filament\Actions\Action;
use Filament\Actions\BulkAction;
use Filament\Facades\Filament;
use Filament\Forms\Components\FileUpload;
use Filament\Notifications\Notification;
use Filament\Pages\Page;
use Filament\Schemas\Components\EmbeddedTable;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Concerns\InteractsWithTable;
use Filament\Tables\Contracts\HasTable;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Table;
use Illuminate\Pagination\LengthAwarePaginator;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Blade;
use Illuminate\Support\HtmlString;
use Livewire\Features\SupportFileUploads\TemporaryUploadedFile;
use Symfony\Component\HttpFoundation\StreamedResponse;
use Throwable;
use UnitEnum;

/**
 * Bulk SEO editor (L7-06): posts, products, places, blog + shop categories and the indexable static pages in one
 * table. SEO title / meta description / robots / cornerstone are edited inline into `$drafts` (live character + SERP
 * pixel counters in the browser, same widths as the SEO tab) and saved together by «ذخیره تغییرات» through
 * BulkSaveSeoMeta — one transaction, every row through SeoMetaObserver, caches (`seo`, `sitemap`, `pages`) bumped once.
 * Filters: type, «نیاز به کار», missing, too long, duplicate, noindex, unscored. Bulk actions: noindex / index, reset
 * to default, recompute score. CSV export / import. Every write is recorded in the activity log (`seo`).
 * Access: SEO managers + super-admins (deny by default).
 */
final class BulkEditor extends Page implements HasTable
{
    use InteractsWithTable;

    /** Upper bound of rows saved by one click (a page holds ≤ 100 rows; imports have their own limit). */
    public const MAX_DRAFTS = 500;

    public const ISSUES = [
        'needs_work' => 'نیاز به کار',
        'missing' => 'بدون عنوان یا توضیح اختصاصی',
        'too_long' => 'طولانی (بریده می‌شود)',
        'duplicate' => 'تکراری',
        'noindex' => 'noindex',
        'unscored' => 'بدون امتیاز',
    ];

    public const ROBOTS = [
        '' => 'پیش‌فرض',
        'index,follow' => 'index, follow',
        'noindex,follow' => 'noindex, follow',
        'noindex,nofollow' => 'noindex, nofollow',
        'index,nofollow' => 'index, nofollow',
    ];

    /**
     * Unsaved inline edits: key => [title?, description?, robots?, cornerstone?]. Client-writable: validated on save.
     *
     * @var array<string, mixed>
     */
    public array $drafts = [];

    /** @var list<array<string, mixed>>|null rows of this request (not persisted between requests) */
    private ?array $rows = null;

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedTableCells;

    protected static string|UnitEnum|null $navigationGroup = 'سئو';

    protected static ?int $navigationSort = 15;

    protected static ?string $navigationLabel = 'ویرایش گروهی سئو';

    protected static ?string $title = 'ویرایش گروهی سئو';

    protected static ?string $slug = 'seo/bulk';

    public static function canAccess(): bool
    {
        return AdminAccess::allows(Filament::auth()->user(), [AdminRole::SeoManager]);
    }

    public function mount(): void
    {
        abort_unless(self::canAccess(), 403);
    }

    public function content(Schema $schema): Schema
    {
        return $schema->components([EmbeddedTable::make()]);
    }

    public function table(Table $table): Table
    {
        return $table
            ->records(fn (?string $search, ?array $filters, ?string $sortColumn, ?string $sortDirection, int|string $page, int|string $recordsPerPage): LengthAwarePaginator => $this->paginate(
                $this->filtered($search, $filters ?? [], $sortColumn, $sortDirection),
                (int) $page,
                $recordsPerPage,
            ))
            ->resolveSelectedRecordsUsing(fn (array $keys, bool $isTrackingDeselectedKeys, array $deselectedKeys): Collection => $isTrackingDeselectedKeys
                ? collect($this->filtered($this->getTableSearch(), $this->tableFilters ?? [], null, null))->keyBy('key')->except($deselectedKeys)
                : collect($this->allRows())->keyBy('key')->only($keys))
            ->searchable()
            ->paginated([25, 50, 100])
            ->defaultPaginationPageOption(50)
            ->striped()
            ->columns([
                TextColumn::make('name')
                    ->label('صفحه')
                    ->weight('medium')
                    ->wrap()
                    ->sortable()
                    ->description(static fn (array $record): string => $record['type_label'].($record['published'] ? '' : ' · پیش‌نویس')),
                TextColumn::make('title')
                    ->label('عنوان سئو')
                    ->state(static fn (array $record): string => $record['key'])
                    ->formatStateUsing(fn (array $record): HtmlString => $this->textCell($record, 'title')),
                TextColumn::make('description')
                    ->label('توضیح متا')
                    ->state(static fn (array $record): string => $record['key'])
                    ->formatStateUsing(fn (array $record): HtmlString => $this->textCell($record, 'description')),
                TextColumn::make('robots')
                    ->label('ربات‌ها')
                    ->state(static fn (array $record): string => $record['key'])
                    ->formatStateUsing(fn (array $record): HtmlString => $this->robotsCell($record)),
                TextColumn::make('score')
                    ->label('امتیاز')
                    ->sortable()
                    ->badge()
                    ->state(static fn (array $record): string => $record['score'] === null ? '—' : SerpMeasure::digits((int) $record['score']))
                    ->color(static fn (array $record): string => match (true) {
                        $record['score'] === null => 'gray',
                        $record['score'] < SeoAnalysis::NEEDS_WORK_SCORE => 'danger',
                        $record['score'] < SeoAnalysis::GOOD_SCORE => 'warning',
                        default => 'success',
                    })
                    ->tooltip(static fn (array $record): string => $record['score'] === null ? 'هنوز بررسی نشده' : 'امتیاز تحلیل محتوا از ۱۰۰'),
                TextColumn::make('issues')
                    ->label('بررسی‌ها')
                    ->badge()
                    ->state(static fn (array $record): array => self::issueLabels($record))
                    ->color(static fn (string $state): string => $state === 'noindex' ? 'gray' : 'warning'),
            ])
            ->filters([
                SelectFilter::make('type')->label('نوع صفحه')->options(ResolveSeoTarget::typeOptions()),
                SelectFilter::make('issue')->label('وضعیت')->options(self::ISSUES),
            ])
            ->headerActions([
                $this->saveAction(),
                Action::make('discard')
                    ->label('لغو تغییرات')
                    ->color('gray')
                    ->visible(fn (): bool => $this->drafts !== [])
                    ->action(function (): void {
                        $this->drafts = [];
                        $this->rows = null;
                    }),
                $this->recomputeAllAction(),
                $this->exportAction(),
                $this->importAction(),
            ])
            ->recordActions([
                Action::make('live')
                    ->label('مشاهده')
                    ->icon(Heroicon::OutlinedArrowTopRightOnSquare)
                    ->iconButton()
                    ->url(static fn (array $record): string => (string) $record['url'])
                    ->openUrlInNewTab(),
                Action::make('edit')
                    ->label('ویرایش کامل')
                    ->icon(Heroicon::OutlinedPencilSquare)
                    ->iconButton()
                    ->url(static fn (array $record): ?string => self::editUrl($record))
                    ->visible(static fn (array $record): bool => self::editUrl($record) !== null),
            ])
            ->toolbarActions([
                $this->robotsBulkAction('noindex', 'noindex کردن', 'noindex,follow', Heroicon::OutlinedEyeSlash),
                $this->robotsBulkAction('index', 'index کردن (پیش‌فرض)', '', Heroicon::OutlinedEye),
                BulkAction::make('reset')
                    ->label('بازگشت به پیش‌فرض')
                    ->icon(Heroicon::OutlinedArrowUturnRight)
                    ->color('danger')
                    ->requiresConfirmation()
                    ->modalDescription('همه تنظیمات سئوی صفحه‌های انتخاب‌شده (عنوان، توضیح، ربات‌ها، تصویر اشتراک و …) پاک می‌شود و پیش‌فرض‌ها دوباره استفاده می‌شوند.')
                    ->authorize(static fn (): bool => self::canAccess())
                    ->action(function (Collection $records, ResetSeoMeta $reset): void {
                        abort_unless(self::canAccess(), 403);
                        $keys = self::keys($records);
                        $done = $reset->handle($keys);
                        $this->forgetDrafts($keys);
                        if ($done !== []) {
                            $this->log('deleted', 'seo.bulk.reset', ['keys' => $done]);
                        }
                        Notification::make()->success()->title(SerpMeasure::digits(count($done)).' صفحه به پیش‌فرض برگشت.')->send();
                    })
                    ->deselectRecordsAfterCompletion(),
                BulkAction::make('rescore')
                    ->label('محاسبه دوباره امتیاز')
                    ->icon(Heroicon::OutlinedArrowPath)
                    ->authorize(static fn (): bool => self::canAccess())
                    ->action(function (Collection $records, RecomputeSeoScores $scores): void {
                        abort_unless(self::canAccess(), 403);
                        $done = $scores->handle(self::keys($records));
                        $this->rows = null;
                        Notification::make()->success()->title('امتیاز '.SerpMeasure::digits(count($done)).' صفحه به‌روز شد.')->send();
                    })
                    ->deselectRecordsAfterCompletion(),
            ]);
    }

    /**
     * Saves every pending inline edit in one action (one transaction, one cache bump) and logs it.
     */
    public function saveDrafts(BulkSaveSeoMeta $save): void
    {
        abort_unless(self::canAccess(), 403);

        $drafts = [];
        foreach (array_slice($this->drafts, 0, self::MAX_DRAFTS, true) as $key => $fields) {
            if (is_array($fields) && ResolveSeoTarget::parse((string) $key) !== null) {
                $drafts[(string) $key] = array_intersect_key($fields, array_flip(BulkSaveSeoMeta::FIELDS));
            }
        }

        $changed = $drafts === [] ? [] : $save->handle($drafts);
        if ($changed !== []) {
            $this->log('updated', 'seo.bulk.updated', ['changes' => $changed]);
        }

        $this->drafts = [];
        $this->rows = null;

        Notification::make()->success()->title($changed === [] ? 'تغییری برای ذخیره نبود.' : 'سئوی '.SerpMeasure::digits(count($changed)).' صفحه ذخیره شد.')->send();
    }

    private function saveAction(): Action
    {
        return Action::make('saveDrafts')
            ->label(fn (): string => 'ذخیره تغییرات'.($this->drafts !== [] ? ' ('.SerpMeasure::digits(count($this->drafts)).')' : ''))
            ->icon(Heroicon::OutlinedCheck)
            ->color(fn (): string => $this->drafts !== [] ? 'primary' : 'gray')
            ->keyBindings(['mod+s'])
            ->action(fn (BulkSaveSeoMeta $save) => $this->saveDrafts($save));
    }

    private function recomputeAllAction(): Action
    {
        return Action::make('rescoreAll')
            ->label('محاسبه همه امتیازها')
            ->icon(Heroicon::OutlinedArrowPath)
            ->color('gray')
            ->requiresConfirmation()
            ->modalDescription('تحلیل محتوا برای همه صفحه‌ها اجرا و امتیازها ذخیره می‌شود. ممکن است چند ثانیه طول بکشد.')
            ->action(function (RecomputeSeoScores $scores): void {
                abort_unless(self::canAccess(), 403);
                $done = $scores->handle();
                $this->rows = null;
                Notification::make()->success()->title('امتیاز '.SerpMeasure::digits(count($done)).' صفحه به‌روز شد.')->send();
            });
    }

    private function exportAction(): Action
    {
        return Action::make('export')
            ->label('خروجی CSV')
            ->icon(Heroicon::OutlinedArrowDownTray)
            ->color('gray')
            ->action(function (ExportBulkSeoCsv $export): StreamedResponse {
                abort_unless(self::canAccess(), 403);
                $rows = $this->allRows();
                $this->log('exported', 'seo.bulk.exported', ['rows' => count($rows)]);

                return response()->streamDownload(static function () use ($export, $rows): void {
                    $out = fopen('php://output', 'wb');
                    if ($out !== false) {
                        $export->handle($out, $rows);
                        fclose($out);
                    }
                }, ExportBulkSeoCsv::filename(), ['Content-Type' => 'text/csv; charset=UTF-8']);
            });
    }

    private function importAction(): Action
    {
        return Action::make('import')
            ->label('درون‌ریزی CSV')
            ->icon(Heroicon::OutlinedArrowUpTray)
            ->color('gray')
            ->modalDescription('همان قالب خروجی CSV: ستون key الزامی است؛ title، description، robots و cornerstone هر کدام که باشند ذخیره می‌شوند (خانه خالی = پیش‌فرض). ستون‌های دیگر نادیده گرفته می‌شوند.')
            ->schema([
                FileUpload::make('file')->label('فایل CSV')->required()->storeFiles(false)
                    ->acceptedFileTypes(['text/csv', 'text/plain', 'application/vnd.ms-excel'])->maxSize(2048),
            ])
            ->action(function (array $data, ImportBulkSeoCsv $import): void {
                abort_unless(self::canAccess(), 403);
                $file = $data['file'] ?? null;
                if (! $file instanceof TemporaryUploadedFile) {
                    return;
                }

                $result = $import->handle((string) $file->getRealPath());
                if ($result['changed'] !== []) {
                    $this->log('updated', 'seo.bulk.imported', ['changes' => $result['changed']]);
                }
                $this->rows = null;

                $body = 'به‌روز: '.SerpMeasure::digits(count($result['changed'])).' — ردشده: '.SerpMeasure::digits($result['skipped']);
                if ($result['errors'] !== []) {
                    $body .= "\n".implode("\n", $result['errors']);
                }
                Notification::make()->title('درون‌ریزی انجام شد')->body($body)
                    ->status($result['skipped'] > 0 ? 'warning' : 'success')->persistent()->send();
            });
    }

    private function robotsBulkAction(string $name, string $label, string $robots, Heroicon $icon): BulkAction
    {
        return BulkAction::make($name)
            ->label($label)
            ->icon($icon)
            ->color('gray')
            ->requiresConfirmation()
            ->authorize(static fn (): bool => self::canAccess())
            ->action(function (Collection $records, BulkSaveSeoMeta $save) use ($robots, $name): void {
                abort_unless(self::canAccess(), 403);
                $keys = self::keys($records);
                $changed = $save->handle(array_fill_keys($keys, ['robots' => $robots]));
                $this->forgetDrafts($keys, 'robots');
                if ($changed !== []) {
                    $this->log('updated', 'seo.bulk.'.$name, ['changes' => $changed]);
                }
                Notification::make()->success()->title(SerpMeasure::digits(count($changed)).' صفحه به‌روز شد.')->send();
            })
            ->deselectRecordsAfterCompletion();
    }

    /**
     * @return list<array<string, mixed>>
     */
    private function allRows(): array
    {
        return $this->rows ??= app(ListBulkSeoRows::class)->handle();
    }

    /**
     * @param  array<string, mixed>  $filters
     * @return list<array<string, mixed>>
     */
    private function filtered(?string $search, array $filters, ?string $sortColumn, ?string $sortDirection): array
    {
        $type = $filters['type']['value'] ?? null;
        $issue = $filters['issue']['value'] ?? null;
        $search = mb_strtolower(trim((string) $search));

        $rows = array_values(array_filter($this->allRows(), static function (array $row) use ($type, $issue, $search): bool {
            if (is_string($type) && $type !== '' && $row['type'] !== $type) {
                return false;
            }
            if (is_string($issue) && $issue !== '' && ! self::hasIssue($row, $issue)) {
                return false;
            }
            if ($search === '') {
                return true;
            }
            foreach (['name', 'title', 'description', 'effective_title', 'url'] as $field) {
                if (str_contains(mb_strtolower((string) $row[$field]), $search)) {
                    return true;
                }
            }

            return false;
        }));

        if (in_array($sortColumn, ['name', 'score'], true)) {
            usort($rows, static fn (array $a, array $b): int => $sortColumn === 'score'
                ? ($a['score'] ?? -1) <=> ($b['score'] ?? -1)
                : strcmp((string) $a['name'], (string) $b['name']));
            if ($sortDirection === 'desc') {
                $rows = array_reverse($rows);
            }
        }

        return $rows;
    }

    /**
     * @param  list<array<string, mixed>>  $rows
     * @return LengthAwarePaginator<string, array<string, mixed>>
     */
    private function paginate(array $rows, int $page, int|string $perPage): LengthAwarePaginator
    {
        $perPage = is_numeric($perPage) && (int) $perPage > 0 ? (int) $perPage : max(1, count($rows));
        $page = max(1, $page);
        $items = [];
        foreach (array_slice($rows, ($page - 1) * $perPage, $perPage) as $row) {
            $items[$row['key']] = $row;
        }

        return new LengthAwarePaginator($items, count($rows), $perPage, $page);
    }

    /**
     * @param  array<string, mixed>  $row
     */
    private static function hasIssue(array $row, string $issue): bool
    {
        return match ($issue) {
            'needs_work' => (bool) $row['needs_work'],
            'missing' => (bool) $row['missing'],
            'too_long' => (bool) $row['too_long'],
            'duplicate' => $row['duplicate_title'] || $row['duplicate_description'],
            'noindex' => ! $row['indexable'],
            'unscored' => $row['score'] === null,
            default => true,
        };
    }

    /**
     * @param  array<string, mixed>  $row
     * @return list<string>
     */
    private static function issueLabels(array $row): array
    {
        return array_values(array_filter([
            $row['duplicate_title'] ? 'عنوان تکراری' : null,
            $row['duplicate_description'] ? 'توضیح تکراری' : null,
            $row['too_long'] ? 'طولانی' : null,
            $row['missing'] ? 'متای ناقص' : null,
            $row['indexable'] ? null : 'noindex',
            $row['cornerstone'] ? 'محتوای اصلی' : null,
        ]));
    }

    /**
     * @param  array<string, mixed>  $row
     */
    private static function editUrl(array $row): ?string
    {
        $target = ResolveSeoTarget::parse((string) $row['key']);
        if ($target === null) {
            return null;
        }

        try {
            if ($target['page'] !== null) {
                return StaticPageSeoResource::canViewAny() ? StaticPageSeoResource::getUrl('edit', ['page' => StaticPageSeoResource::key($target['page'])]) : null;
            }

            $resource = match ($target['type']) {
                'blog_post' => PostResource::class,
                'shop_product' => ProductResource::class,
                'directory_place' => PlaceResource::class,
                'blog_category' => BlogCategoryResource::class,
                'shop_category' => ShopCategoryResource::class,
                default => null,
            };

            return $resource !== null && $resource::canViewAny() ? $resource::getUrl('edit', ['record' => $target['id']]) : null;
        } catch (Throwable) {
            return null;
        }
    }

    /**
     * @param  Collection<array-key, mixed>  $records
     * @return list<string>
     */
    private static function keys(Collection $records): array
    {
        return array_values($records->map(static fn (mixed $r): string => is_array($r) ? (string) ($r['key'] ?? '') : '')
            ->filter(static fn (string $key): bool => ResolveSeoTarget::parse($key) !== null)
            ->all());
    }

    /**
     * @param  list<string>  $keys
     */
    private function forgetDrafts(array $keys, ?string $field = null): void
    {
        foreach ($keys as $key) {
            if ($field === null) {
                unset($this->drafts[$key]);
            } else {
                $fields = $this->drafts[$key] ?? null;
                if (is_array($fields)) {
                    unset($fields[$field]);
                    $this->drafts[$key] = $fields;
                }
                if (($this->drafts[$key] ?? null) === []) {
                    unset($this->drafts[$key]);
                }
            }
        }
        $this->rows = null;
    }

    /**
     * @param  array<string, mixed>  $properties
     */
    private function log(string $event, string $description, array $properties): void
    {
        activity('seo')
            ->causedBy(Filament::auth()->user())
            ->event($event)
            ->withProperties($properties)
            ->log($description);
    }

    /**
     * The unsaved draft value of a row's field, null when there is none.
     *
     * @param  array<string, mixed>  $record
     */
    private function draft(array $record, string $field): mixed
    {
        $fields = $this->drafts[(string) $record['key']] ?? null;

        return is_array($fields) ? ($fields[$field] ?? null) : null;
    }

    /**
     * The current value of a field: the unsaved draft if any, else the stored override.
     *
     * @param  array<string, mixed>  $record
     */
    private function value(array $record, string $field): string
    {
        $draft = $this->draft($record, $field);

        return is_scalar($draft) ? (string) $draft : (string) $record[$field];
    }

    /**
     * Inline input + live counters. Counters measure what Google shows: the override through the title template (or
     * the inherited value when empty), same char ranges / pixel widths as TextWidth.
     *
     * @param  array<string, mixed>  $record
     */
    private function textCell(array $record, string $field): HtmlString
    {
        $isTitle = $field === 'title';
        $seo = app(SettingsRepository::class)->all()->seo;
        $value = $this->value($record, $field);

        return new HtmlString(Blade::render(self::TEXT_CELL, [
            'key' => (string) $record['key'],
            'field' => $field,
            'value' => $value,
            'dirty' => $this->draft($record, $field) !== null && $value !== (string) $record[$field],
            'multiline' => ! $isTitle,
            'config' => [
                'v' => $value,
                'tpl' => $isTitle ? $seo->template() : '',
                'fallback' => $record['effective_'.$field] ?? '',
                'stored' => (string) $record[$field],
                'font' => $isTitle ? TextWidth::TITLE_FONT_PX : TextWidth::DESCRIPTION_FONT_PX,
                'max' => $isTitle ? TextWidth::TITLE_MAX_PX : TextWidth::DESCRIPTION_MAX_PX,
                'lo' => ($isTitle ? TextWidth::TITLE_CHARS : TextWidth::DESCRIPTION_CHARS)[0],
                'hi' => ($isTitle ? TextWidth::TITLE_CHARS : TextWidth::DESCRIPTION_CHARS)[1],
                'narrow' => self::NARROW,
            ],
            'placeholder' => (string) ($record['effective_'.$field] ?? ''),
            'label' => ($isTitle ? 'عنوان سئو: ' : 'توضیح متا: ').$record['name'],
            'duplicate' => (bool) $record['duplicate_'.$field],
        ]));
    }

    /**
     * @param  array<string, mixed>  $record
     */
    private function robotsCell(array $record): HtmlString
    {
        $value = $this->value($record, 'robots');
        $cornerstoneDraft = $this->draft($record, 'cornerstone');
        $cornerstone = $cornerstoneDraft !== null ? filter_var($cornerstoneDraft, FILTER_VALIDATE_BOOLEAN) : (bool) $record['cornerstone'];

        return new HtmlString(Blade::render(self::ROBOTS_CELL, [
            'key' => (string) $record['key'],
            'value' => array_key_exists($value, self::ROBOTS) ? $value : '',
            'options' => self::ROBOTS + (array_key_exists($value, self::ROBOTS) ? [] : [$value => $value]),
            'cornerstone' => $cornerstone,
            'name' => (string) $record['name'],
        ]));
    }

    /** Narrow Latin characters of TextWidth (0.28 em). */
    private const NARROW = "ijlI.,:;'|!`()[]{}/\\ ";

    /** Inline cell; the per-character widths mirror App\Domain\Seo\Analysis\TextWidth::em(). */
    private const TEXT_CELL = <<<'BLADE'
        <div class="min-w-64 space-y-1"
             x-data="{
                ...@js($config),
                full() { const t = this.v.trim(); return t === '' ? this.fallback : (this.tpl !== '' ? this.tpl.replace('%s', t) : t); },
                px(t) {
                    let w = 0;
                    for (const ch of t.trim()) {
                        const c = ch.codePointAt(0);
                        w += ((c >= 0x64B && c <= 0x65F) || c === 0x670 || (c >= 0x200B && c <= 0x200F)) ? 0
                            : this.narrow.includes(ch) ? 0.28
                            : ((c >= 0x600 && c <= 0x6FF) || (c >= 0xFB50 && c <= 0xFEFF)) ? 0.45
                            : /[0-9]/.test(ch) ? 0.556 : /[mwMW]/.test(ch) ? 0.83 : /[A-Z]/.test(ch) ? 0.68 : /[a-z]/.test(ch) ? 0.5 : 0.55;
                    }
                    return Math.round(w * this.font);
                },
                chars() { return [...this.full().trim()].length; },
                status() { const n = this.chars(); if (n === 0) return 'gray'; if (this.px(this.full()) > this.max) return 'danger'; return (n < this.lo || n > this.hi) ? 'warning' : 'success'; },
                fa(n) { return Number(n).toLocaleString('fa-IR', { useGrouping: false }); },
                push() { if (this.v !== this.stored || $wire.get('drafts.{{ $key }}.{{ $field }}') !== undefined) { $wire.set('drafts.{{ $key }}.{{ $field }}', this.v, false); } },
             }">
            @if ($multiline)
                <textarea rows="3" dir="rtl" aria-label="{{ $label }}" placeholder="{{ $placeholder }}" maxlength="500"
                          class="fi-input block w-full rounded-lg border border-gray-300 bg-white px-2 py-1 text-sm dark:border-white/10 dark:bg-white/5 @if ($dirty) ring-2 ring-warning-500 @endif"
                          x-on:input="v = $event.target.value" x-on:change="push()">{{ $value }}</textarea>
            @else
                <input type="text" dir="rtl" aria-label="{{ $label }}" placeholder="{{ $placeholder }}" maxlength="255" value="{{ $value }}"
                       class="fi-input block w-full rounded-lg border border-gray-300 bg-white px-2 py-1 text-sm dark:border-white/10 dark:bg-white/5 @if ($dirty) ring-2 ring-warning-500 @endif"
                       x-on:input="v = $event.target.value" x-on:change="push()">
            @endif
            <p class="text-xs"
               x-bind:class="{ 'text-success-600 dark:text-success-400': status() === 'success', 'text-warning-600 dark:text-warning-400': status() === 'warning', 'text-danger-600 dark:text-danger-400': status() === 'danger', 'text-gray-500': status() === 'gray' }">
                <span x-text="fa(chars()) + ' نویسه · ' + fa(px(full())) + ' از ' + fa(max) + ' پیکسل'"></span>
                <span x-show="v.trim() === ''" class="text-gray-500"> · پیش‌فرض</span>
                @if ($duplicate)<span class="text-danger-600 dark:text-danger-400"> · تکراری</span>@endif
            </p>
        </div>
        BLADE;

    private const ROBOTS_CELL = <<<'BLADE'
        <div class="min-w-36 space-y-2">
            <select aria-label="ربات‌ها: {{ $name }}"
                    class="fi-select-input block w-full rounded-lg border border-gray-300 bg-white px-2 py-1 text-sm dark:border-white/10 dark:bg-white/5"
                    x-on:change="$wire.set('drafts.{{ $key }}.robots', $event.target.value, false)">
                @foreach ($options as $optionValue => $optionLabel)
                    <option value="{{ $optionValue }}" @selected($optionValue === $value)>{{ $optionLabel }}</option>
                @endforeach
            </select>
            <label class="flex items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
                <input type="checkbox" class="fi-checkbox-input rounded" @checked($cornerstone)
                       x-on:change="$wire.set('drafts.{{ $key }}.cornerstone', $event.target.checked, false)">
                محتوای اصلی (cornerstone)
            </label>
        </div>
        BLADE;
}

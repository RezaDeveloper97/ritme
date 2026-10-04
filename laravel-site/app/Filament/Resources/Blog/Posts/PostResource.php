<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Posts;

use App\Domain\Blog\Actions\DuplicatePost;
use App\Domain\Blog\Actions\FindOrCreateTag;
use App\Domain\Blog\Actions\SetPostsStatus;
use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Blog\Enums\PostStatus;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\Tag;
use App\Domain\Blog\Support\BlogUrls;
use App\Filament\Components\Seo\SeoFields;
use App\Filament\Components\Seo\SerpMeasure;
use App\Filament\Forms\Components\MediaPicker;
use App\Filament\Resources\Blog\BlogPolicy;
use App\Filament\Resources\Blog\Posts\Pages\CreatePost;
use App\Filament\Resources\Blog\Posts\Pages\EditPost;
use App\Filament\Resources\Blog\Posts\Pages\ListPosts;
use App\Filament\Resources\Media\MediaUploads;
use App\Support\Html\TextSlug;
use BackedEnum;
use Filament\Actions\Action;
use Filament\Actions\BulkAction;
use Filament\Actions\BulkActionGroup;
use Filament\Actions\DeleteBulkAction;
use Filament\Actions\EditAction;
use Filament\Facades\Filament;
use Filament\Forms\Components\DatePicker;
use Filament\Forms\Components\DateTimePicker;
use Filament\Forms\Components\RichEditor;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Notifications\Notification;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Components\Section;
use Filament\Schemas\Components\Tabs;
use Filament\Schemas\Components\Tabs\Tab;
use Filament\Schemas\Components\Utilities\Get;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Filters\TernaryFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Support\Facades\Gate;
use Livewire\Features\SupportFileUploads\TemporaryUploadedFile;
use UnitEnum;

/**
 * Magazine posts (thin delivery over the Blog actions): rich editor with media-library images (stored as
 * data-media-id, sanitised by PostObserver on save), status/schedule, featured, author + medical reviewer, life
 * stage, category and tags (both creatable inline), cover + mobile cover, the reusable SEO tab, signed draft
 * preview, duplicate, revision trail and bulk publish/unpublish. Access: PostPolicy (editors write, SEO managers
 * edit the SEO tab only).
 */
final class PostResource extends Resource
{
    protected static ?string $model = Post::class;

    protected static ?string $slug = 'blog/posts';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedNewspaper;

    protected static string|UnitEnum|null $navigationGroup = 'مجله';

    protected static ?int $navigationSort = 10;

    protected static ?string $modelLabel = 'مقاله';

    protected static ?string $pluralModelLabel = 'مقاله‌ها';

    protected static ?string $recordTitleAttribute = 'title';

    public static function canEditContent(): bool
    {
        return BlogPolicy::canEditContent(Filament::auth()->user());
    }

    public static function form(Schema $schema): Schema
    {
        $locked = static fn (): bool => ! self::canEditContent();

        return $schema->columns(1)->components([
            Tabs::make('post')
                ->persistTabInQueryString()
                ->tabs([
                    Tab::make('محتوا')->icon(Heroicon::OutlinedDocumentText)->disabled($locked)->schema([
                        TextInput::make('title')->label('عنوان')->required()->maxLength(255)->live(onBlur: true),
                        TextInput::make('slug')
                            ->label('نامک (slug)')
                            ->maxLength(120)
                            ->live(onBlur: true)
                            ->helperText('خالی بماند، از عنوان ساخته می‌شود؛ فارسی مجاز است. تغییر نامک مقاله منتشرشده نشانی قبلی را ۳۰۱ می‌کند.'),
                        Textarea::make('excerpt')
                            ->label('خلاصه')
                            ->rows(3)
                            ->maxLength(500)
                            ->live(onBlur: true)
                            ->helperText('در فهرست مقاله‌ها و، اگر توضیح متا خالی باشد، در نتایج جست‌وجو نمایش داده می‌شود.'),
                        self::bodyEditor(),
                        RichEditor::make('sources')
                            ->label('منابع')
                            ->toolbarButtons([['bold', 'italic', 'link'], ['bulletList', 'orderedList'], ['undo', 'redo']])
                            ->fileAttachments(false),
                    ]),
                    Tab::make('انتشار')->icon(Heroicon::OutlinedCalendarDays)->disabled($locked)->schema([
                        Grid::make(2)->schema([
                            Select::make('status')
                                ->label('وضعیت')
                                ->options(self::statusOptions())
                                ->default(PostStatus::Draft->value)
                                ->required()
                                ->native(false)
                                ->live(),
                            DateTimePicker::make('published_at')
                                ->label('زمان انتشار')
                                ->seconds(false)
                                ->required(static fn (Get $get): bool => $get('status') === PostStatus::Scheduled->value || $get('status') === PostStatus::Scheduled)
                                ->helperText('زمان آینده با وضعیت «منتشرشده» یا «زمان‌بندی‌شده» خودکار در همان زمان منتشر می‌شود.'),
                            Toggle::make('is_featured')->label('مقاله ویژه')->helperText('در صدر صفحه مجله.'),
                            Select::make('life_stage')
                                ->label('مرحله زندگی')
                                ->options(self::lifeStageOptions())
                                ->placeholder('همه'),
                        ]),
                        Section::make('دسته و برچسب‌ها')->schema([
                            Select::make('category_id')
                                ->label('دسته')
                                ->relationship('category', 'name')
                                ->searchable()
                                ->preload()
                                ->createOptionForm([
                                    TextInput::make('name')->label('نام دسته')->required()->maxLength(191),
                                    TextInput::make('label')->label('برچسب کوتاه روی کارت')->maxLength(64),
                                ])
                                ->createOptionModalHeading('دسته جدید'),
                            Select::make('tag_ids')
                                ->label('برچسب‌ها')
                                ->multiple()
                                ->searchable()
                                ->options(static fn (): array => Tag::query()->orderBy('name')->limit(200)->pluck('name', 'id')->all())
                                ->getSearchResultsUsing(static fn (string $search): array => Tag::query()
                                    ->where('name', 'like', '%'.addcslashes($search, '%_\\').'%')
                                    ->orderBy('name')->limit(50)->pluck('name', 'id')->all())
                                ->getOptionLabelsUsing(static fn (array $values): array => Tag::query()->whereKey($values)->pluck('name', 'id')->all())
                                ->createOptionForm([TextInput::make('name')->label('نام برچسب')->required()->maxLength(191)])
                                ->createOptionUsing(static fn (array $data): int => app(FindOrCreateTag::class)->handle((string) $data['name'])->id)
                                ->createOptionModalHeading('برچسب جدید'),
                        ]),
                        Section::make('نویسنده و بازبینی پزشکی')->columns(3)->schema([
                            Select::make('author_id')
                                ->label('نویسنده')
                                ->relationship('author', 'name')
                                ->searchable()
                                ->preload(),
                            Select::make('reviewer_id')
                                ->label('بازبین پزشکی')
                                ->relationship('reviewer', 'name', static fn (Builder $query): Builder => $query->where('is_medical_reviewer', true))
                                ->searchable()
                                ->preload()
                                ->helperText('فقط نویسندگانی که «بازبین پزشکی» علامت خورده‌اند.'),
                            DatePicker::make('reviewed_at')->label('تاریخ بازبینی'),
                        ]),
                    ]),
                    Tab::make('تصاویر')->icon(Heroicon::OutlinedPhoto)->disabled($locked)->schema([
                        MediaPicker::make('cover_media_id')->label('تصویر شاخص')->live(),
                        MediaPicker::make('cover_mobile_media_id')->mobileOverride(),
                    ]),
                    Tab::make('سئو')->icon(Heroicon::OutlinedMagnifyingGlass)->schema([
                        SeoFields::make()
                            ->titleFrom('title')
                            ->descriptionFrom('excerpt')
                            ->imageFrom('cover_media_id')
                            ->urlUsing(static fn (Get $get): string => app(BlogUrls::class)->post(self::previewSlug($get))),
                    ]),
                ]),
        ]);
    }

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->with(['category', 'author']))
            ->defaultSort('id', 'desc')
            ->columns([
                TextColumn::make('title')->label('عنوان')->searchable()->limit(60)->wrap(),
                TextColumn::make('status')->label('وضعیت')->badge()
                    ->formatStateUsing(static fn (PostStatus $state): string => $state->label())
                    ->color(static fn (PostStatus $state): string => match ($state) {
                        PostStatus::Published => 'success',
                        PostStatus::Scheduled => 'info',
                        PostStatus::Archived => 'gray',
                        PostStatus::Draft => 'warning',
                    }),
                TextColumn::make('category.name')->label('دسته')->placeholder('—'),
                TextColumn::make('author.name')->label('نویسنده')->placeholder('—')->toggleable(),
                IconColumn::make('is_featured')->label('ویژه')->boolean()->toggleable(),
                TextColumn::make('published_at')->label('انتشار')->dateTime('Y-m-d H:i')->placeholder('—')->sortable(),
                TextColumn::make('views')->label('بازدید')->numeric()->sortable()->toggleable(isToggledHiddenByDefault: true),
            ])
            ->filters([
                SelectFilter::make('status')->label('وضعیت')->options(self::statusOptions()),
                SelectFilter::make('category_id')->label('دسته')->relationship('category', 'name'),
                SelectFilter::make('life_stage')->label('مرحله زندگی')->options(self::lifeStageOptions()),
                TernaryFilter::make('is_featured')->label('ویژه'),
            ])
            ->recordActions([
                EditAction::make(),
                self::previewAction(),
                self::duplicateAction(),
            ])
            ->toolbarActions([
                BulkActionGroup::make([
                    self::bulkStatusAction('publish', 'انتشار', PostStatus::Published, Heroicon::OutlinedCheckCircle),
                    self::bulkStatusAction('unpublish', 'لغو انتشار (پیش‌نویس)', PostStatus::Draft, Heroicon::OutlinedEyeSlash),
                    DeleteBulkAction::make(),
                ]),
            ]);
    }

    public static function getPages(): array
    {
        return [
            'index' => ListPosts::route('/'),
            'create' => CreatePost::route('/create'),
            'edit' => EditPost::route('/{record}/edit'),
        ];
    }

    public static function previewAction(): Action
    {
        return Action::make('preview')
            ->label('پیش‌نمایش')
            ->icon(Heroicon::OutlinedEye)
            ->color('gray')
            ->url(static fn (Post $record): string => PostPreviewController::url($record), shouldOpenInNewTab: true);
    }

    public static function duplicateAction(): Action
    {
        return Action::make('duplicate')
            ->label('نسخه‌برداری')
            ->icon(Heroicon::OutlinedDocumentDuplicate)
            ->color('gray')
            ->requiresConfirmation()
            ->modalDescription('یک پیش‌نویس تازه با همین محتوا، برچسب‌ها و تنظیمات سئو ساخته می‌شود.')
            ->authorize(static fn (Post $record): bool => Gate::forUser(Filament::auth()->user())->allows('replicate', $record))
            ->action(static function (Post $record, Action $action): void {
                $copy = app(DuplicatePost::class)->handle($record);
                PostRevisions::event($copy, 'duplicated', ['source_id' => $record->id]);
                Notification::make()->success()->title('نسخه پیش‌نویس ساخته شد.')->send();
                $action->redirect(self::getUrl('edit', ['record' => $copy]));
            });
    }

    private static function bulkStatusAction(string $name, string $label, PostStatus $status, Heroicon $icon): BulkAction
    {
        return BulkAction::make($name)
            ->label($label)
            ->icon($icon)
            ->requiresConfirmation()
            ->authorize(static fn (): bool => self::canEditContent())
            ->deselectRecordsAfterCompletion()
            ->action(static function (Collection $records) use ($status, $name): void {
                /** @var Collection<int, Post> $records */
                $changed = app(SetPostsStatus::class)->handle($records, $status);
                foreach ($records as $post) {
                    if (in_array($post->id, $changed, true)) {
                        PostRevisions::event($post, $name === 'publish' ? 'published' : 'unpublished', ['attributes' => ['status' => $post->status->value]]);
                    }
                }
                Notification::make()->success()->title(SerpMeasure::digits(count($changed)).' مقاله به‌روز شد.')->send();
            });
    }

    private static function bodyEditor(): RichEditor
    {
        return RichEditor::make('body')
            ->label('متن مقاله')
            ->required()
            ->toolbarButtons([
                ['bold', 'italic', 'underline', 'strike', 'link'],
                ['h2', 'h3'],
                ['blockquote', 'bulletList', 'orderedList', 'table'],
                ['attachFiles', InsertMediaAction::NAME],
                ['undo', 'redo'],
            ])
            ->tools([InsertMediaAction::tool()])
            ->registerActions([InsertMediaAction::make()])
            ->fileAttachments(true)
            ->fileAttachmentsAcceptedFileTypes(array_values(array_diff(MediaUploads::acceptedTypes(), ['image/svg+xml'])))
            ->fileAttachmentsMaxSize(MediaUploads::maxKilobytes())
            ->saveUploadedFileAttachmentUsing(static fn (TemporaryUploadedFile $file): int => MediaUploads::store($file)->id)
            ->getFileAttachmentUrlUsing(static fn (mixed $file): ?string => EditorImages::url($file))
            ->helperText('تیتر اصلی (h1) همان عنوان مقاله است؛ داخل متن از تیتر ۲ و ۳ استفاده کنید. تصویرها از کتابخانه رسانه درج و بهینه می‌شوند.');
    }

    private static function previewSlug(Get $get): string
    {
        $slug = trim((string) $get('slug'));
        if ($slug === '') {
            $slug = TextSlug::make((string) $get('title'), 120);
        }

        return $slug === '' ? 'slug' : $slug;
    }

    /**
     * @return array<string, string>
     */
    private static function statusOptions(): array
    {
        $options = [];
        foreach (PostStatus::cases() as $status) {
            $options[$status->value] = $status->label();
        }

        return $options;
    }

    /**
     * @return array<string, string>
     */
    public static function lifeStageOptions(): array
    {
        $options = [];
        foreach (LifeStage::cases() as $stage) {
            $options[$stage->value] = $stage->label();
        }

        return $options;
    }
}

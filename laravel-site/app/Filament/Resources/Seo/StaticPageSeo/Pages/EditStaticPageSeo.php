<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\StaticPageSeo\Pages;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Actions\ListStaticPageSeo;
use App\Domain\Seo\Actions\SaveStaticPageSeo;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Filament\Components\Seo\SeoFields;
use App\Filament\Resources\Seo\StaticPageSeo\StaticPageSeoResource;
use Filament\Actions\Action;
use Filament\Facades\Filament;
use Filament\Notifications\Notification;
use Filament\Resources\Pages\Page;
use Filament\Schemas\Components\Actions;
use Filament\Schemas\Components\EmbeddedSchema;
use Filament\Schemas\Components\Form;
use Filament\Schemas\Components\Section;
use Filament\Schemas\Components\Text;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;

/**
 * SEO editor of one static page: the reusable SeoFields (standalone — no model) with the page's lang defaults as
 * fallbacks, so counters and the SERP / share previews show exactly what the live page will render. Saves through
 * SaveStaticPageSeo (activity log `seo`).
 *
 * @property-read Schema $form
 */
final class EditStaticPageSeo extends Page
{
    protected static string $resource = StaticPageSeoResource::class;

    public string $page = '';

    /**
     * @var array<string, mixed>|null
     */
    public ?array $data = [];

    public function mount(string $page): void
    {
        $target = StaticPageSeoResource::pageFor($page);
        abort_if($target === null, 404);
        abort_unless(StaticPageSeoResource::canManage($target), 403);

        $this->page = $page;
        $this->fillFromRow($target);
    }

    /** The stored overrides (cached repository) into the form; no row = every field empty (inherit). */
    private function fillFromRow(StaticPage $page): void
    {
        $stored = app(SeoMetaRepository::class)->forRoute($page->routeName())?->toArray() ?? ['sitemap_include' => true];
        $this->form->fill(['seoMeta' => SeoFields::fillRobots($stored)]);
    }

    public function getTitle(): string
    {
        return 'سئوی «'.$this->staticPage()->label().'»';
    }

    public function getBreadcrumb(): string
    {
        return $this->staticPage()->label();
    }

    public function form(Schema $schema): Schema
    {
        $page = $this->staticPage();
        $row = app(ListStaticPageSeo::class)->find($page);

        return $schema->statePath('data')->components([
            Section::make('مقدار پیش‌فرض صفحه')
                ->description('وقتی فیلدی خالی بماند، این متن‌ها (از فایل زبان صفحه) استفاده می‌شوند.')
                ->collapsible()
                ->schema([
                    Text::make('عنوان: '.$row->defaults->title.($row->defaults->titleIsComplete ? '' : ' (+ قالب عنوان سایت)')),
                    Text::make('توضیح: '.$row->defaults->description),
                    Text::make('نشانی: '.$row->url)->extraAttributes(['dir' => 'ltr']),
                ]),
            SeoFields::standalone()
                ->titleFrom(static fn (): string => $row->defaults->title)
                ->fallbackTitleIsComplete($row->defaults->titleIsComplete)
                ->descriptionFrom(static fn (): string => $row->defaults->description)
                ->urlUsing(static fn (): string => $row->url),
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

    protected function getHeaderActions(): array
    {
        $page = $this->staticPage();

        return [
            Action::make('live')
                ->label('مشاهده زنده')
                ->icon(Heroicon::OutlinedArrowTopRightOnSquare)
                ->color('gray')
                ->url(app(ListStaticPageSeo::class)->url($page))
                ->openUrlInNewTab(),
            StaticPageSeoResource::resetAction($page)
                ->after(fn () => $this->fillFromRow($page)),
        ];
    }

    public function save(SaveStaticPageSeo $save): void
    {
        $page = $this->staticPage();
        abort_unless(StaticPageSeoResource::canManage($page), 403);

        /** @var array<string, mixed> $state */
        $state = $this->form->getState();
        /** @var array<string, mixed> $values */
        $values = is_array($state['seoMeta'] ?? null) ? $state['seoMeta'] : [];
        $meta = $save->handle($page, SeoFields::dehydrateRobots($values));

        $changes = $meta->getChanges();
        unset($changes['updated_at']);
        if ($meta->wasRecentlyCreated || $changes !== []) {
            activity('seo')
                ->causedBy(Filament::auth()->user())
                ->performedOn($meta)
                ->event($meta->wasRecentlyCreated ? 'created' : 'updated')
                ->withProperties(['route' => $page->routeName(), 'attributes' => $meta->wasRecentlyCreated ? $meta->only(SaveStaticPageSeo::FIELDS) : $changes])
                ->log('seo.static_page.saved');
        }

        Notification::make()->success()->title('سئوی صفحه ذخیره شد.')->send();
    }

    private function staticPage(): StaticPage
    {
        $page = StaticPageSeoResource::pageFor($this->page);
        abort_if($page === null, 404);

        return $page;
    }
}

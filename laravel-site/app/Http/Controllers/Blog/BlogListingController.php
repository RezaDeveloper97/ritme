<?php

declare(strict_types=1);

namespace App\Http\Controllers\Blog;

use App\Domain\Blog\Contracts\AuthorRepository;
use App\Domain\Blog\Contracts\CategoryRepository;
use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Contracts\TagRepository;
use App\Domain\Blog\Data\AuthorData;
use App\Domain\Blog\Data\CategoryData;
use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Data\PostPage;
use App\Domain\Blog\Models\Author;
use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Tag;
use App\Domain\Blog\Support\BlogUrls;
use App\Domain\Blog\Support\PostListIndexing;
use App\Domain\Blog\Support\TagIndexing;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Data\SeoMetaData;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\Data\ListEntry;
use App\Domain\Seo\Schema\Enums\WebPageType;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\Nodes\ItemListNode;
use App\Domain\Seo\Schema\Nodes\PersonNode;
use App\Domain\Seo\Schema\PageGraph;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\Support\CanonicalUrl;
use App\Domain\Seo\Support\DescriptionText;
use App\Domain\Settings\Contracts\SettingsRepository;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\View\View;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;

/**
 * Magazine lists (L4-02): `/blog` (featured hero + latest), `/blog/category/{slug}`, `/blog/tag/{slug}`,
 * `/blog/author/{slug}`. All reads go through the cached Blog repositories; views get DTOs only.
 *
 * Pagination `?page=n`: n > 1 is a self-canonical, indexable page with the page number in title + description;
 * `?page=1` 301s to the clean URL; a non-numeric or out-of-range page is a 404. Any other query parameter makes the
 * page a filtered listing → SeoManager forces noindex (and the page cache skips it). Tag pages below the
 * TagIndexing threshold, empty lists and authors without posts (unless a medical reviewer with a real bio —
 * PostListIndexing) are `noindex,follow`. Admin seo_meta (route `blog.index`, or the category /
 * tag / author row) wins over the defaults here.
 */
final class BlogListingController
{
    public const PER_PAGE = 12;

    public function __construct(
        private readonly PostRepository $posts,
        private readonly CategoryRepository $categories,
        private readonly SeoManager $seo,
        private readonly SchemaGraph $graph,
        private readonly SeoMetaRepository $seoMeta,
        private readonly SettingsRepository $settings,
        private readonly BlogUrls $urls,
        private readonly Config $config,
    ) {}

    public function index(Request $request): View|RedirectResponse
    {
        $page = $this->page($request);
        if ($page instanceof RedirectResponse) {
            return $page;
        }

        $list = $this->guard($this->posts->latest($page, self::PER_PAGE));
        $featured = $page === 1 ? $this->posts->featured() : null;
        $items = $featured === null ? $list->items : array_values(array_filter(
            $list->items,
            static fn (PostCardData $card): bool => $card->id !== $featured->id,
        ));
        $reviewer = $featured === null ? null : $this->posts->findPublishedBySlug($featured->slug)?->reviewer?->name;

        $this->applySeo(
            meta: $this->seoMeta->forRoute('blog.index'),
            title: $page === 1 ? __('blog.index.seo_title') : __('blog.index.seo_title_paged'),
            description: __('blog.index.seo_description'),
            page: $page,
            indexable: PostListIndexing::listIndexable($list),
        );
        $this->schema($request, $this->blogTrail(), $featured === null ? $items : [$featured, ...$items], __('blog.name'));

        return $this->view($request, $list, $items, [
            'intro' => ['eyebrow' => __('blog.index.eyebrow'), 'title' => __('blog.index.title'), 'lead' => __('blog.index.lead')],
            'chips' => $this->chips(null),
            'featured' => $featured,
            'featuredReviewer' => $reviewer,
        ]);
    }

    public function category(Request $request, string $slug): View|RedirectResponse
    {
        $category = $this->categories->findBySlug($slug) ?? abort(404);
        $page = $this->page($request);
        if ($page instanceof RedirectResponse) {
            return $page;
        }

        $list = $this->guard($this->posts->byCategory($category->id, $page, self::PER_PAGE));
        $title = __('blog.category.seo_title', ['name' => $category->name]);

        $this->seo->for($this->subject(Category::class, $category->id));
        $this->applySeo(
            meta: $this->seoMeta->forModel((new Category)->getMorphClass(), $category->id),
            title: $title,
            description: $category->description ?? __('blog.category.seo_description', ['name' => $category->name]),
            page: $page,
            indexable: PostListIndexing::listIndexable($list),
        );
        $this->schema($request, [...$this->blogTrail(), new BreadcrumbItem($category->name, $this->urls->category($category->slug))], $list->items, $category->name);

        return $this->view($request, $list, $list->items, [
            'intro' => [
                'eyebrow' => __('blog.category.eyebrow'),
                'title' => $category->name,
                'lead' => $category->description ?? __('blog.category.lead', ['name' => $category->name]),
            ],
            'chips' => $this->chips($category),
        ]);
    }

    public function tag(Request $request, string $slug, TagRepository $tags, TagIndexing $indexing): View|RedirectResponse
    {
        $tag = $tags->findBySlug($slug) ?? abort(404);
        $page = $this->page($request);
        if ($page instanceof RedirectResponse) {
            return $page;
        }

        $list = $this->guard($this->posts->byTag($tag->id, $page, self::PER_PAGE));

        $this->seo->for($this->subject(Tag::class, $tag->id));
        $this->applySeo(
            meta: $this->seoMeta->forModel((new Tag)->getMorphClass(), $tag->id),
            title: __('blog.tag.seo_title', ['name' => $tag->name]),
            description: __('blog.tag.seo_description', ['name' => $tag->name]),
            page: $page,
            indexable: $indexing->indexable($tag),
        );
        $trail = [...$this->blogTrail(), new BreadcrumbItem($tag->name, $this->urls->tag($tag->slug))];
        $this->schema($request, $trail, $list->items, $tag->name);

        return $this->view($request, $list, $list->items, [
            'intro' => ['eyebrow' => __('blog.tag.eyebrow'), 'title' => $tag->name, 'lead' => __('blog.tag.lead', ['name' => $tag->name])],
            'breadcrumbs' => $trail,
        ]);
    }

    public function author(Request $request, string $slug, AuthorRepository $authors): View|RedirectResponse
    {
        $author = $authors->findBySlug($slug) ?? abort(404);
        $page = $this->page($request);
        if ($page instanceof RedirectResponse) {
            return $page;
        }

        $list = $this->guard($this->posts->byAuthor($author->id, $page, self::PER_PAGE));
        $reviewer = $author->isMedicalReviewer && $list->total === 0;

        $this->seo->for($this->subject(Author::class, $author->id));
        $this->applySeo(
            meta: $this->seoMeta->forModel((new Author)->getMorphClass(), $author->id),
            title: __($reviewer ? 'blog.author.seo_title_reviewer' : 'blog.author.seo_title', ['name' => $author->name]),
            // A placeholder («[…]») or too-short bio is no description; the lang default is used instead.
            description: PostListIndexing::hasRealBio($author->bio) ? trim((string) $author->bio) : __('blog.author.seo_description', ['name' => $author->name]),
            page: $page,
            // A medical reviewer's credentials page is worth indexing even without own articles (E-E-A-T) — once the
            // bio is real.
            indexable: PostListIndexing::authorIndexable($author, $list),
        );
        $this->seo->type('profile');
        $trail = [...$this->blogTrail(), new BreadcrumbItem($author->name, $this->urls->author($author->slug))];
        $this->schema($request, $trail, $list->items, __('blog.author.posts_title', ['name' => $author->name]), WebPageType::ProfilePage, $author);

        return $this->view($request, $list, $list->items, [
            'intro' => [
                'eyebrow' => __($author->isMedicalReviewer ? 'blog.author.eyebrow_reviewer' : 'blog.author.eyebrow'),
                'title' => $author->name,
                'lead' => $author->jobTitle,
            ],
            'author' => $author,
            'breadcrumbs' => $trail,
            'postsHeading' => __('blog.author.posts_title', ['name' => $author->name]),
        ]);
    }

    /**
     * @param  list<PostCardData>  $items
     * @param  array<string, mixed>  $data
     */
    private function view(Request $request, PostPage $list, array $items, array $data): View
    {
        $appLinks = $this->settings->all()->appLinks;

        return view('pages.blog.index', [
            'intro' => $data['intro'],
            'chips' => $data['chips'] ?? [],
            'featured' => $data['featured'] ?? null,
            'featuredReviewer' => $data['featuredReviewer'] ?? null,
            'author' => $data['author'] ?? null,
            'breadcrumbs' => $data['breadcrumbs'] ?? null,
            'postsHeading' => $data['postsHeading'] ?? __('blog.posts_heading'),
            'posts' => $items,
            'pagination' => $this->pagination($request, $list),
            'newsletterSource' => trim($request->path(), '/'),
            'appLinks' => $appLinks,
            'qrUrl' => $appLinks->webApp ?? $this->urls->absolute('/'),
            'navRoute' => 'blog.index',
        ]);
    }

    /**
     * Requested page number, a 301 for `?page=1`, or a 404 for anything that is not a positive integer.
     */
    private function page(Request $request): int|RedirectResponse
    {
        if (! $request->query->has('page')) {
            return 1;
        }

        $raw = $request->query->all()['page'] ?? null;
        if (! is_string($raw) || ! ctype_digit($raw) || (int) $raw < 1 || strlen($raw) > 6) {
            abort(404);
        }

        if ((int) $raw === 1) {
            $query = $request->query->all();
            unset($query['page']);

            return new RedirectResponse($request->url().($query === [] ? '' : '?'.http_build_query($query)), 301);
        }

        return (int) $raw;
    }

    /**
     * Out-of-range pages are 404s (page 1 of an empty list is a valid, empty page).
     */
    private function guard(PostPage $list): PostPage
    {
        if ($list->page > 1 && $list->isOutOfRange()) {
            abort(404);
        }

        return $list;
    }

    private function applySeo(?SeoMetaData $meta, string $title, string $description, int $page, bool $indexable): void
    {
        $title = $meta->title ?? $title;
        $description = $meta->description ?? $description;

        if ($page > 1) {
            $suffix = __('blog.page_suffix', ['page' => self::digits($page)]);
            $this->seo->title($title.' — '.$suffix)->description(DescriptionText::fromExcerpt($suffix.' — '.$description));
        } else {
            $this->seo->title($title)->description($meta?->description !== null ? $description : DescriptionText::fromExcerpt($description));
        }

        if (! $indexable) {
            $this->seo->noindex();
        }
    }

    /**
     * BreadcrumbList trail, CollectionPage / ProfilePage type and the ItemList of the cards shown.
     *
     * @param  list<BreadcrumbItem>  $trail
     * @param  list<PostCardData>  $items
     */
    private function schema(Request $request, array $trail, array $items, string $name, WebPageType $type = WebPageType::CollectionPage, ?AuthorData $author = null): void
    {
        $this->graph->breadcrumbs(...$trail)->pageType($type)->pageName($name);
        $siteUrl = (string) $this->config->get('app.url');
        $pageUrl = CanonicalUrl::normalize($request->fullUrl(), $siteUrl);
        $page = ['@id' => SchemaIds::webPage($pageUrl)];

        if ($items !== []) {
            $this->graph->add(ItemListNode::make($pageUrl, array_map(
                fn (PostCardData $card): ListEntry => new ListEntry($this->urls->post($card->slug), $card->title),
                $items,
            ), $name));
            $page['mainEntity'] = Node::ref(SchemaIds::itemList($pageUrl));
        }

        if ($author !== null) {
            $person = PersonNode::make($author->toPerson($this->urls->author($author->slug)), SchemaIds::root($siteUrl));
            $this->graph->add($person);
            $page['mainEntity'] = Node::ref((string) $person['@id']);
        }

        if (count($page) > 1) {
            $this->graph->add($page);
        }
    }

    /**
     * Category chips: «همه» + top-level categories (the design's tab row is plain links, AUDIT §2.2).
     *
     * @return list<array{label: string, href: string, active: bool}>
     */
    private function chips(?CategoryData $active): array
    {
        $chips = [['label' => __('blog.all'), 'href' => route('blog.index'), 'active' => $active === null]];
        foreach ($this->categories->all() as $category) {
            if ($category->parentId === null) {
                $chips[] = [
                    'label' => $category->name,
                    'href' => route('blog.category', $category->slug),
                    'active' => $active !== null && ($active->id === $category->id || $active->parentId === $category->id),
                ];
            }
        }

        return $chips;
    }

    /**
     * @return array{label: string, previous: string|null, next: string|null, pages: list<array{number: string, href: string|null, current: bool}>}|null
     */
    private function pagination(Request $request, PostPage $list): ?array
    {
        $last = $list->lastPage();
        if ($last < 2) {
            return null;
        }

        $url = static fn (int $page): string => $request->url().($page > 1 ? '?page='.$page : '');
        $pages = [];
        $previous = 0;
        foreach (range(1, $last) as $n) {
            if ($n !== 1 && $n !== $last && abs($n - $list->page) > 2) {
                continue; // window of ±2 around the current page, plus first and last
            }
            if ($previous !== 0 && $n - $previous > 1) {
                $pages[] = ['number' => '…', 'href' => null, 'current' => false];
            }
            $pages[] = ['number' => self::digits($n), 'href' => $url($n), 'current' => $n === $list->page];
            $previous = $n;
        }

        return [
            'label' => __('blog.pagination.label'),
            'previous' => $list->page > 1 ? $url($list->page - 1) : null,
            'next' => $list->page < $last ? $url($list->page + 1) : null,
            'pages' => $pages,
        ];
    }

    /**
     * @return list<BreadcrumbItem>
     */
    private function blogTrail(): array
    {
        return [$this->home(), new BreadcrumbItem(__('blog.name'), $this->urls->absolute(route('blog.index')))];
    }

    private function home(): BreadcrumbItem
    {
        return new BreadcrumbItem(PageGraph::HOME_LABEL, SchemaIds::root((string) $this->config->get('app.url')));
    }

    /**
     * Keyed, unsaved model reference so SeoManager::for() can layer the row's seo_meta (morph type + id) without
     * loading the model — the list data itself comes from the cached repositories.
     *
     * @param  class-string<Model>  $class
     */
    private function subject(string $class, int $id): Model
    {
        return (new $class)->forceFill(['id' => $id]);
    }

    /**
     * Persian digits for page numbers (swap for fa_digits() once L3-01b's helpers are in).
     */
    private static function digits(int $number): string
    {
        return strtr((string) $number, ['0' => '۰', '1' => '۱', '2' => '۲', '3' => '۳', '4' => '۴', '5' => '۵', '6' => '۶', '7' => '۷', '8' => '۸', '9' => '۹']);
    }
}

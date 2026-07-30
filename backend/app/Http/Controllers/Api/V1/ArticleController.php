<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Concerns\ResolvesLocale;
use App\Http\Controllers\Controller;
use App\Models\Article;
use App\Services\Content\HtmlSanitizer;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * The reader-facing side of the articles an admin publishes: the full library
 * (browse / filter by category / search) and a single article by slug.
 *
 * The home page has its own phase-matched row (HomePage\Sections\ArticlesSection);
 * this controller is deliberately phase-blind — a browsable list keyed by the
 * user's cycle phase would put health data in a shareable URL (§11).
 */
class ArticleController extends Controller
{
    use ResolvesLocale;

    private const DEFAULT_PER_PAGE = 12;

    private const MAX_PER_PAGE = 50;

    /** How many further reads the article page suggests at the bottom. */
    private const RELATED_LIMIT = 4;

    /**
     * @OA\Get(
     *     path="/articles",
     *     summary="List published articles",
     *     description="Paginated library of published educational articles, newest first. Supports a category filter and a title/excerpt search, and returns the set of categories in use so the client can render filter chips without a second call.",
     *     tags={"Articles"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(name="page", in="query", required=false, @OA\Schema(type="integer", minimum=1, default=1)),
     *     @OA\Parameter(name="per_page", in="query", required=false, @OA\Schema(type="integer", minimum=1, maximum=50, default=12)),
     *     @OA\Parameter(name="category", in="query", required=false, description="Exact category to filter by.", @OA\Schema(type="string")),
     *     @OA\Parameter(name="q", in="query", required=false, description="Free-text search over the localized title and excerpt.", @OA\Schema(type="string", maxLength=100)),
     *     @OA\Parameter(name="Accept-Language", in="header", required=false, @OA\Schema(type="string", default="fa", enum={"fa","en"})),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Articles retrieved successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="items", type="array", @OA\Items(type="object")),
     *                 @OA\Property(property="categories", type="array", @OA\Items(type="string")),
     *                 @OA\Property(property="meta", type="object",
     *                     @OA\Property(property="current_page", type="integer", example=1),
     *                     @OA\Property(property="last_page", type="integer", example=3),
     *                     @OA\Property(property="per_page", type="integer", example=12),
     *                     @OA\Property(property="total", type="integer", example=27)
     *                 )
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=422, description="Invalid query parameters")
     * )
     */
    public function index(Request $request): JsonResponse
    {
        $filters = $request->validate([
            'page' => ['nullable', 'integer', 'min:1'],
            'per_page' => ['nullable', 'integer', 'min:1', 'max:'.self::MAX_PER_PAGE],
            'category' => ['nullable', 'string', 'max:255'],
            'q' => ['nullable', 'string', 'max:100'],
        ]);

        $locale = $this->resolveLocale($request);
        $perPage = (int) ($filters['per_page'] ?? self::DEFAULT_PER_PAGE);

        $paginator = $this->listQuery($filters, $locale)
            ->paginate($perPage)
            ->withQueryString();

        return response()->json([
            'success' => true,
            'data' => [
                'items' => $paginator->getCollection()
                    ->map(fn (Article $article) => $this->summary($article, $locale))
                    ->all(),
                'categories' => $this->categories(),
                'meta' => [
                    'current_page' => $paginator->currentPage(),
                    'last_page' => $paginator->lastPage(),
                    'per_page' => $paginator->perPage(),
                    'total' => $paginator->total(),
                ],
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/articles/{slug}",
     *     summary="Get a single published article",
     *     description="Full body of one published article, plus a few related reads (same category first, then articles sharing a cycle-phase tag). The body is sanitized HTML — clients may render it directly.",
     *     tags={"Articles"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(name="slug", in="path", required=true, @OA\Schema(type="string", example="period-blood-clots")),
     *     @OA\Parameter(name="Accept-Language", in="header", required=false, @OA\Schema(type="string", default="fa", enum={"fa","en"})),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Article retrieved successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="article", type="object"),
     *                 @OA\Property(property="related", type="array", @OA\Items(type="object"))
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="No published article with this slug")
     * )
     */
    public function show(Request $request, string $slug): JsonResponse
    {
        $locale = $this->resolveLocale($request);

        $article = Article::query()->published()->where('slug', $slug)->first();

        if ($article === null) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'مقاله یافت نشد.' : 'Article not found.',
            ], 404);
        }

        return response()->json([
            'success' => true,
            'data' => [
                'article' => $this->detail($article, $locale),
                'related' => $this->related($article)
                    ->map(fn (Article $related) => $this->summary($related, $locale))
                    ->all(),
            ],
        ]);
    }

    /**
     * @param  array<string, mixed>  $filters
     */
    private function listQuery(array $filters, string $locale): Builder
    {
        $query = Article::query()->published();

        if (($category = $filters['category'] ?? null) !== null && $category !== '') {
            $query->where('category', $category);
        }

        if (($term = trim((string) ($filters['q'] ?? ''))) !== '') {
            $query->where(function (Builder $q) use ($term, $locale) {
                // Title and excerpt are {fa, en} JSON, so the search runs against
                // the reader's own locale (with fa as the always-present fallback)
                // instead of matching a translation they will never see.
                foreach (array_unique([$locale, 'fa']) as $lang) {
                    $q->orWhere('title->'.$lang, 'like', '%'.$term.'%')
                        ->orWhere('excerpt->'.$lang, 'like', '%'.$term.'%');
                }
            });
        }

        // Admin-chosen order first, then most recently published.
        return $query->orderBy('sort_order')
            ->orderByDesc('published_at')
            ->orderByDesc('id');
    }

    /**
     * Categories that currently have something published, for the filter chips.
     *
     * @return array<int, string>
     */
    private function categories(): array
    {
        return Article::query()
            ->published()
            ->whereNotNull('category')
            ->where('category', '!=', '')
            ->distinct()
            ->orderBy('category')
            ->pluck('category')
            ->all();
    }

    /**
     * Further reading for the article page: same category first, then anything
     * tagged with a phase this article also carries.
     *
     * @return \Illuminate\Support\Collection<int, Article>
     */
    private function related(Article $article): \Illuminate\Support\Collection
    {
        $phases = $article->cycle_phases ?? [];

        return Article::query()
            ->published()
            ->whereKeyNot($article->getKey())
            ->where(function (Builder $q) use ($article, $phases) {
                if ($article->category !== null && $article->category !== '') {
                    $q->orWhere('category', $article->category);
                }

                foreach ($phases as $phase) {
                    $q->orWhereJsonContains('cycle_phases', $phase);
                }

                // A general article with no category has nothing to match on;
                // fall back to the newest reads rather than an empty rail.
                if ($q->getQuery()->wheres === []) {
                    $q->whereRaw('1 = 1');
                }
            })
            ->orderByDesc('published_at')
            ->orderByDesc('id')
            ->limit(self::RELATED_LIMIT)
            ->get();
    }

    /**
     * Card shape — everything a list item renders, and nothing heavy.
     *
     * @return array<string, mixed>
     */
    private function summary(Article $article, string $locale): array
    {
        return [
            'id' => $article->id,
            'slug' => $article->slug,
            'title' => $article->localized('title', $locale),
            'excerpt' => HtmlSanitizer::toPlainText($article->localized('excerpt', $locale)),
            'image_url' => $article->image_url,
            'read_time_minutes' => $article->read_time_minutes,
            'category' => $article->category,
            'cycle_phases' => $article->cycle_phases ?? [],
            'published_at' => $article->published_at?->toDateString(),
        ];
    }

    /**
     * Card shape plus the sanitized body the reader came for.
     *
     * @return array<string, mixed>
     */
    private function detail(Article $article, string $locale): array
    {
        return [
            ...$this->summary($article, $locale),
            'body' => HtmlSanitizer::clean($article->localized('body', $locale)),
        ];
    }
}

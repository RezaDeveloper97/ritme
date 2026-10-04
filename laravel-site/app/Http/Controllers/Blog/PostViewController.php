<?php

declare(strict_types=1);

namespace App\Http\Controllers\Blog;

use App\Domain\Blog\Actions\RecordPostView;
use App\Domain\Blog\Contracts\PostRepository;
use Illuminate\Http\Request;
use Illuminate\Http\Response;

/**
 * `POST /blog/{slug}/view` — the article view beacon (L4-03b), sent by `resources/js/modules/view-beacon.js` with
 * `navigator.sendBeacon` once per post per tab session. It counts page-cache HITs too: the cached HTML never reaches
 * ShowPostController, so this is the only place a view is recorded.
 *
 * Cookie-free and CSRF-free (route without session/cookie/CSRF middleware, throttled per IP). Abuse limits instead
 * of a token: the body `id` must be the id of the published post at `{slug}` (cached repository lookup, 404
 * otherwise), cross-site requests (`Sec-Fetch-Site` / `Origin`) are ignored, and so are bots, prefetches and
 * `Save-Data` visitors. Ignored beacons still answer 204 so nothing can be learnt from the response.
 */
final class PostViewController
{
    public const PER_MINUTE = 30;

    /** Crawlers, link previewers, monitors, headless browsers and HTTP libraries (empty UA is handled separately). */
    private const BOT_PATTERN = '/bot|crawl|spider|slurp|archiver|preview|facebookexternalhit|embedly|headless|phantom|'
        .'lighthouse|pagespeed|gtmetrix|pingdom|uptime|monitor|curl|wget|python|java\/|go-http|okhttp|axios|node-fetch|'
        .'httpclient|libwww|scrapy|feedfetcher|validator/i';

    public function __construct(
        private readonly PostRepository $posts,
        private readonly RecordPostView $recordView,
    ) {}

    public function __invoke(Request $request, string $slug): Response
    {
        $post = $this->posts->findPublishedBySlug($slug);
        $id = $request->input('id');

        if ($post === null || ! is_scalar($id) || (string) $post->id !== (string) $id) {
            abort(404);
        }

        if ($this->countable($request)) {
            $this->recordView->handle($post->id);
        }

        return new Response('', 204, ['Cache-Control' => 'no-store']);
    }

    private function countable(Request $request): bool
    {
        $site = $request->headers->get('Sec-Fetch-Site');
        if ($site !== null && $site !== 'same-origin') {
            return false;
        }

        $origin = $request->headers->get('Origin');
        if ($origin !== null && $origin !== $request->getSchemeAndHttpHost()) {
            return false;
        }

        if (strtolower((string) $request->headers->get('Save-Data')) === 'on') {
            return false;
        }

        $purpose = strtolower((string) ($request->headers->get('Sec-Purpose') ?? $request->headers->get('Purpose')));
        if (str_contains($purpose, 'prefetch') || str_contains($purpose, 'prerender')) {
            return false;
        }

        $agent = trim((string) $request->userAgent());

        return $agent !== '' && preg_match(self::BOT_PATTERN, $agent) !== 1;
    }
}

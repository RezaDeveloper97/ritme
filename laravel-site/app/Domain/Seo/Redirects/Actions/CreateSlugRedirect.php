<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Actions;

use App\Domain\Seo\Redirects\Enums\RedirectCode;
use App\Domain\Seo\Redirects\InvalidRedirect;
use App\Domain\Seo\Redirects\Models\Redirect;
use App\Domain\Seo\Redirects\Support\RedirectPath;

/**
 * The automatic 301 after a slug change (SlugRedirectObserver). The new URL is live, so redirects *from* it are
 * obsolete and removed first (renaming back and forth never builds a loop); an existing row for the old URL is
 * re-pointed; SaveRedirect then collapses chains (older URLs that pointed at the old one now point at the new one).
 * Never throws: a slug change must not fail because of a redirect.
 */
final class CreateSlugRedirect
{
    public function __construct(private readonly SaveRedirect $save) {}

    /**
     * @param  string  $from  old path, or a regex pattern when $isRegex
     * @param  string  $livePath  a path that is live after the change (used to drop obsolete rows)
     */
    public function handle(string $from, string $to, string $livePath, bool $isRegex = false, string $note = ''): ?Redirect
    {
        $this->dropRedirectsFrom($livePath);

        $hash = $isRegex ? sha1(trim($from)) : RedirectPath::hash($from);
        $redirect = Redirect::query()->where('from_hash', $hash)->where('is_regex', $isRegex)->first() ?? new Redirect;

        try {
            return $this->save->handle($redirect, [
                'from_path' => $from,
                'to_url' => $to,
                'code' => RedirectCode::Permanent->value,
                'is_regex' => $isRegex,
                'is_auto' => true,
                'note' => $note,
            ]);
        } catch (InvalidRedirect $e) {
            report($e);

            return null;
        }
    }

    private function dropRedirectsFrom(string $path): void
    {
        $path = RedirectPath::normalize($path);

        $obsolete = Redirect::query()
            ->where(static fn ($q) => $q->where('is_regex', false)->where('from_hash', RedirectPath::hash($path)))
            ->orWhere(static fn ($q) => $q->where('is_regex', true)->where('is_auto', true))
            ->get();

        foreach ($obsolete as $redirect) {
            if ($redirect->is_regex && @preg_match(RedirectPath::compile($redirect->from_path), $path) !== 1) {
                continue;
            }
            $redirect->delete();
        }
    }
}

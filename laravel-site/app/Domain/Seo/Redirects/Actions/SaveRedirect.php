<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Actions;

use App\Domain\Seo\Redirects\Enums\RedirectCode;
use App\Domain\Seo\Redirects\InvalidRedirect;
use App\Domain\Seo\Redirects\Models\Redirect;
use App\Domain\Seo\Redirects\Support\RedirectPath;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Database\ConnectionInterface;
use Illuminate\Database\Eloquent\Model;

/**
 * Creates or updates one redirect (admin form, CSV import, slug-change auto redirects):
 *
 *  - normalises the source (decoded path, no trailing slash / query) or validates the regex pattern;
 *  - a local target (`/path` or an absolute URL on the site's host) is normalised too; 410 has no target;
 *  - chain collapse: when the target is itself the source of another redirect, the row points straight at the final
 *    destination (A→B + B→C ⇒ A→C; reaching a 410 makes the row 410), and rows that pointed at this source are
 *    re-pointed at the new target (X→A + A→B ⇒ X→B) — every visitor gets one hop;
 *  - loops (A→A, A→B→A, a regex whose fixed target matches itself) and duplicate sources are rejected with
 *    InvalidRedirect.
 *
 * Every save is written to the activity log (`seo`); RedirectObserver bumps the cached map.
 */
final class SaveRedirect
{
    public const LOG = 'seo';

    private const MAX_HOPS = 20;

    public function __construct(private readonly ConnectionInterface $db, private readonly Config $config) {}

    /**
     * @param  array<string, mixed>  $data  from_path, to_url, code, is_regex, note, is_auto
     *
     * @throws InvalidRedirect
     */
    public function handle(Redirect $redirect, array $data, ?Model $causer = null, bool $log = true): Redirect
    {
        $isRegex = (bool) ($data['is_regex'] ?? $redirect->is_regex);
        $from = $this->source((string) ($data['from_path'] ?? $redirect->from_path ?? ''), $isRegex);
        $code = $this->code($data['code'] ?? $redirect->code);
        $fromHash = $isRegex ? sha1($from) : RedirectPath::hash($from);

        $duplicate = Redirect::query()->where('from_hash', $fromHash)->where('is_regex', $isRegex)
            ->when($redirect->exists, static fn ($q) => $q->whereKeyNot($redirect->id))->exists();
        if ($duplicate) {
            throw new InvalidRedirect('from_path', 'برای این آدرس قبلاً ریدایرکت ثبت شده است.');
        }

        $target = null;
        $local = null;
        if (! $code->isGone()) {
            [$target, $local] = $this->target((string) ($data['to_url'] ?? $redirect->to_url ?? ''));

            if ($isRegex) {
                $this->guardRegexLoop($from, $target, $local);
            } elseif ($local !== null) {
                [$target, $local, $code] = $this->collapse($from, $target, $local, $code, $redirect->exists ? $redirect->id : null);
            }
        }

        $note = trim((string) ($data['note'] ?? $redirect->note ?? ''));
        $isNew = ! $redirect->exists;
        $before = $isNew ? [] : ['from_path' => $redirect->from_path, 'to_url' => $redirect->to_url, 'code' => $redirect->code->value];

        $redirect->fill([
            'from_path' => $from,
            'from_hash' => $fromHash,
            'to_url' => $target,
            'to_hash' => $local !== null && $local['query'] === null ? RedirectPath::hash($local['path']) : null,
            'code' => $code,
            'is_regex' => $isRegex,
            'is_auto' => (bool) ($data['is_auto'] ?? ($isNew ? false : $redirect->is_auto)),
            'note' => $note === '' ? null : mb_substr($note, 0, 255),
        ]);

        $this->db->transaction(function () use ($redirect, $isRegex, $fromHash): void {
            $redirect->save();

            if (! $isRegex) {
                $this->repointIncoming($redirect, $fromHash);
            }
        });

        if (! $log) {
            return $redirect; // bulk callers (CSV import) log one summary entry instead
        }

        activity(self::LOG)
            ->causedBy($causer)
            ->performedOn($redirect)
            ->event($isNew ? 'created' : 'updated')
            ->withProperties([
                'old' => $before,
                'attributes' => ['from_path' => $redirect->from_path, 'to_url' => $redirect->to_url, 'code' => $redirect->code->value],
            ])
            ->log($redirect->is_auto ? 'ریدایرکت خودکار (تغییر نامک)' : ($isNew ? 'ریدایرکت ایجاد شد' : 'ریدایرکت ویرایش شد'));

        return $redirect;
    }

    private function source(string $from, bool $isRegex): string
    {
        $from = trim($from);

        if ($isRegex) {
            if (mb_strlen($from) > RedirectPath::MAX_LENGTH || ! RedirectPath::isValidRegex($from)) {
                throw new InvalidRedirect('from_path', 'الگوی عبارت باقاعده معتبر نیست.');
            }

            return $from;
        }

        if ($from === '' || preg_match('~^[a-z][a-z0-9+.-]*://~i', $from) === 1) {
            $from = $from === '' ? '' : (string) (RedirectPath::local($from, $this->appUrl())['path'] ?? '');
        }
        $path = RedirectPath::normalize($from);

        if ($from === '' || $path === '/') {
            throw new InvalidRedirect('from_path', 'آدرس مبدأ باید یک مسیر از همین سایت باشد (نه صفحهٔ اصلی).');
        }
        if (mb_strlen($path) > RedirectPath::MAX_LENGTH) {
            throw new InvalidRedirect('from_path', 'آدرس مبدأ خیلی طولانی است.');
        }

        return $path;
    }

    private function code(mixed $code): RedirectCode
    {
        if ($code instanceof RedirectCode) {
            return $code;
        }

        return RedirectCode::tryFrom(is_numeric($code) ? (int) $code : 0)
            ?? throw new InvalidRedirect('code', 'کد وضعیت باید ۳۰۱، ۳۰۲، ۳۰۷ یا ۴۱۰ باشد.');
    }

    /**
     * @return array{0: string, 1: array{path: string, query: string|null}|null}
     */
    private function target(string $to): array
    {
        $to = trim($to);
        if ($to === '') {
            throw new InvalidRedirect('to_url', 'آدرس مقصد را وارد کنید (یا کد ۴۱۰ را انتخاب کنید).');
        }

        $local = RedirectPath::local($to, $this->appUrl());
        if ($local !== null) {
            return [$this->join($local), $local];
        }

        $scheme = strtolower((string) parse_url($to, PHP_URL_SCHEME));
        if (! in_array($scheme, ['http', 'https'], true) || parse_url($to, PHP_URL_HOST) === null || mb_strlen($to) > 2048) {
            throw new InvalidRedirect('to_url', 'مقصد باید مسیری مثل ‎/blog‎ یا نشانی کامل http(s) باشد.');
        }

        return [$to, null];
    }

    /**
     * Follows exact redirects from the target to the final destination.
     *
     * @param  array{path: string, query: string|null}  $local
     * @return array{0: string|null, 1: array{path: string, query: string|null}|null, 2: RedirectCode}
     */
    private function collapse(string $from, string $target, array $local, RedirectCode $code, ?int $ignoreId): array
    {
        $seen = [RedirectPath::key($from) => true];

        for ($hop = 0; $hop <= self::MAX_HOPS; $hop++) {
            $key = RedirectPath::key($local['path']);
            if (isset($seen[$key])) {
                throw new InvalidRedirect('to_url', 'این ریدایرکت یک حلقه می‌سازد (مقصد دوباره به مبدأ برمی‌گردد).');
            }
            $seen[$key] = true;

            $next = Redirect::query()->where('from_hash', sha1($key))->where('is_regex', false)
                ->when($ignoreId !== null, static fn ($q) => $q->whereKeyNot($ignoreId))->first();
            if ($next === null) {
                return [$target, $local, $code];
            }
            if ($next->code->isGone()) {
                return [null, null, RedirectCode::Gone];
            }

            $target = (string) $next->to_url;
            $local = RedirectPath::local($target, $this->appUrl());
            if ($local === null) {
                return [$target, null, $code]; // ends on an external URL
            }
        }

        throw new InvalidRedirect('to_url', 'زنجیرهٔ ریدایرکت‌ها بیش از حد طولانی است.');
    }

    /**
     * @param  array{path: string, query: string|null}|null  $local
     */
    private function guardRegexLoop(string $pattern, string $target, ?array $local): void
    {
        if ($local !== null && ! str_contains($target, '$') && preg_match(RedirectPath::compile($pattern), $local['path']) === 1) {
            throw new InvalidRedirect('to_url', 'مقصد با همین الگو هم مطابقت دارد و حلقه می‌سازد.');
        }
    }

    /**
     * Rows that redirected to this row's source now go straight to its target (or become 410 with it).
     */
    private function repointIncoming(Redirect $redirect, string $fromHash): void
    {
        $incoming = Redirect::query()->where('to_hash', $fromHash)->where('is_regex', false)->whereKeyNot($redirect->id);

        if ($redirect->code->isGone()) {
            $incoming->toBase()->update(['to_url' => null, 'to_hash' => null, 'code' => RedirectCode::Gone->value]);

            return;
        }

        $incoming->toBase()->update(['to_url' => $redirect->to_url, 'to_hash' => $redirect->to_hash]);

        // A re-pointed row whose target became its own source would loop: it is obsolete.
        Redirect::query()->where('is_regex', false)->whereColumn('from_hash', 'to_hash')->toBase()->delete();
    }

    /**
     * @param  array{path: string, query: string|null}  $local
     */
    private function join(array $local): string
    {
        return $local['path'].($local['query'] === null ? '' : '?'.$local['query']);
    }

    private function appUrl(): string
    {
        return (string) $this->config->get('app.url', '');
    }
}

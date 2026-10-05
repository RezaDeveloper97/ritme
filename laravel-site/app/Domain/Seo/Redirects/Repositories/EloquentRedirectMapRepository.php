<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Repositories;

use App\Domain\Seo\Redirects\Contracts\RedirectMapRepository;
use App\Domain\Seo\Redirects\Data\RedirectMap;
use App\Domain\Seo\Redirects\Enums\RedirectCode;
use App\Domain\Seo\Redirects\Models\Redirect;
use App\Domain\Seo\Redirects\Support\RedirectPath;

final class EloquentRedirectMapRepository implements RedirectMapRepository
{
    /** Regex rows are tried one by one on a 404: keep the list short. */
    public const MAX_REGEX = 200;

    public function map(): RedirectMap
    {
        $exact = [];
        $regex = [];

        $rows = Redirect::query()->orderBy('id')->get(['id', 'from_path', 'to_url', 'code', 'is_regex']);
        foreach ($rows as $row) {
            $code = $row->code->value;
            $target = $code === RedirectCode::Gone->value ? null : $row->to_url;

            if ($row->is_regex) {
                if (count($regex) < self::MAX_REGEX && RedirectPath::isValidRegex($row->from_path)) {
                    $regex[] = [$row->id, RedirectPath::compile($row->from_path), $target, $code];
                }

                continue;
            }

            $exact[RedirectPath::key($row->from_path)] = [$row->id, $target, $code];
        }

        return new RedirectMap($exact, $regex);
    }
}

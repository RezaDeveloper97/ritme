<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Actions;

use App\Domain\Seo\Redirects\InvalidRedirect;
use App\Domain\Seo\Redirects\Models\NotFoundLog;
use App\Domain\Seo\Redirects\Models\Redirect;
use Illuminate\Database\Eloquent\Model;

/**
 * "Create redirect" from the 404 monitor: saves a redirect for the logged path (SaveRedirect rules apply) and removes
 * the 404 row, which is resolved.
 */
final class CreateRedirectFromNotFound
{
    public function __construct(private readonly SaveRedirect $save) {}

    /**
     * @throws InvalidRedirect
     */
    public function handle(NotFoundLog $log, string $to, int $code, ?Model $causer = null): Redirect
    {
        $redirect = $this->save->handle(new Redirect, [
            'from_path' => $log->path,
            'to_url' => $to,
            'code' => $code,
            'note' => 'از پایش ۴۰۴',
        ], $causer);

        $log->delete();

        return $redirect;
    }
}

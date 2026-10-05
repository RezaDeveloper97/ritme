<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Actions;

use App\Domain\Seo\Redirects\Models\Redirect;
use Illuminate\Database\Eloquent\Model;

/**
 * Deletes redirects one by one (RedirectObserver bumps the map) and records each in the activity log (`seo`).
 */
final class DeleteRedirects
{
    /**
     * @param  list<int>  $ids
     */
    public function handle(array $ids, ?Model $causer = null): int
    {
        $deleted = 0;
        foreach (Redirect::query()->whereKey($ids)->get() as $redirect) {
            $redirect->delete();
            $deleted++;

            activity(SaveRedirect::LOG)
                ->causedBy($causer)
                ->performedOn($redirect)
                ->event('deleted')
                ->withProperties(['old' => ['from_path' => $redirect->from_path, 'to_url' => $redirect->to_url, 'code' => $redirect->code->value]])
                ->log('ریدایرکت حذف شد');
        }

        return $deleted;
    }
}

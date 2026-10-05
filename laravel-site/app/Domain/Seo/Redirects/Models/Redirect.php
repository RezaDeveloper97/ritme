<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Models;

use App\Domain\Seo\Redirects\Enums\RedirectCode;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Carbon;

/**
 * One admin- or auto-managed redirect (L7-03). Write through SaveRedirect (normalisation, chain collapse, loop check);
 * RedirectObserver bumps the `seo` namespace (cached redirect map) and `pages`.
 *
 * @property int $id
 * @property string $from_path
 * @property string $from_hash
 * @property string|null $to_url
 * @property string|null $to_hash
 * @property RedirectCode $code
 * @property bool $is_regex
 * @property bool $is_auto
 * @property int $hits
 * @property Carbon|null $last_hit_at
 * @property string|null $note
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 */
final class Redirect extends Model
{
    protected $table = 'redirects';

    protected $fillable = ['from_path', 'from_hash', 'to_url', 'to_hash', 'code', 'is_regex', 'is_auto', 'note'];

    protected $attributes = ['code' => 301, 'is_regex' => false, 'is_auto' => false, 'hits' => 0];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'code' => RedirectCode::class,
            'is_regex' => 'boolean',
            'is_auto' => 'boolean',
            'hits' => 'integer',
            'last_hit_at' => 'datetime',
        ];
    }
}

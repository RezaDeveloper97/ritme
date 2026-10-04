<?php

declare(strict_types=1);

namespace App\Domain\Settings\Models;

use Illuminate\Database\Eloquent\Model;

/**
 * One admin-editable value: (group, key) => JSON value. Read through SettingsRepository, never directly.
 *
 * @property int $id
 * @property string $group
 * @property string $key
 * @property mixed $value
 */
final class Setting extends Model
{
    protected $table = 'settings';

    protected $fillable = ['group', 'key', 'value'];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return ['value' => 'json:unicode'];
    }
}

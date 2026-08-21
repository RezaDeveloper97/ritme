<?php

namespace App\Models;

use App\Enums\TextDirection;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

/**
 * @OA\Schema(
 *     schema="Language",
 *     type="object",
 *
 *     @OA\Property(property="code", type="string", example="fa"),
 *     @OA\Property(property="name", type="string", example="فارسی", description="Endonym — shown in the language picker"),
 *     @OA\Property(property="english_name", type="string", example="Persian"),
 *     @OA\Property(property="direction", type="string", enum={"rtl","ltr"}, example="rtl"),
 *     @OA\Property(property="is_default", type="boolean", example=true)
 * )
 *
 * One locale the product ships. Read through {@see App\Services\Language\LanguageRegistry}
 * rather than querying this model directly — the registry caches and guarantees
 * a default even on a fresh database.
 */
class Language extends Model
{
    protected $fillable = [
        'code',
        'name',
        'english_name',
        'direction',
        'is_active',
        'is_default',
        'sort_order',
    ];

    protected function casts(): array
    {
        return [
            'direction' => TextDirection::class,
            'is_active' => 'boolean',
            'is_default' => 'boolean',
            'sort_order' => 'integer',
        ];
    }

    /** Locales visible to end users. */
    public function scopeActive(Builder $query): Builder
    {
        return $query->where('is_active', true);
    }

    public function scopeOrdered(Builder $query): Builder
    {
        return $query->orderBy('sort_order')->orderBy('id');
    }

    /**
     * Normalize codes the same way everywhere: lowercase, dash-separated.
     * "PT_br" and "pt-BR" must not become two different locales.
     */
    public static function normalizeCode(string $code): string
    {
        return strtolower(str_replace('_', '-', trim($code)));
    }
}

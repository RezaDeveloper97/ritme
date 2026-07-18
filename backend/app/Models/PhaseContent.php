<?php

namespace App\Models;

use App\Enums\CycleSubphase;
use Illuminate\Database\Eloquent\Model;

/**
 * @OA\Schema(
 *     schema="PhaseContent",
 *     type="object",
 *
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="phase", type="string", example="high_fertility"),
 *     @OA\Property(property="symptom_prediction", type="object", nullable=true,
 *         @OA\Property(property="fa", type="string"),
 *         @OA\Property(property="en", type="string")
 *     ),
 *     @OA\Property(property="vaginal_discharge", type="object", nullable=true,
 *         @OA\Property(property="fa", type="string"),
 *         @OA\Property(property="en", type="string")
 *     ),
 *     @OA\Property(property="fertility", type="object", nullable=true,
 *         @OA\Property(property="fa", type="string"),
 *         @OA\Property(property="en", type="string")
 *     ),
 *     @OA\Property(property="hormonal_changes", type="object", nullable=true,
 *         @OA\Property(property="fa", type="string"),
 *         @OA\Property(property="en", type="string")
 *     ),
 *     @OA\Property(property="sex_tips", type="object", nullable=true,
 *         @OA\Property(property="fa", type="string"),
 *         @OA\Property(property="en", type="string")
 *     ),
 *     @OA\Property(property="nutrition", type="object", nullable=true,
 *         @OA\Property(property="fa", type="string"),
 *         @OA\Property(property="en", type="string")
 *     ),
 *     @OA\Property(property="exercise", type="object", nullable=true,
 *         @OA\Property(property="fa", type="string"),
 *         @OA\Property(property="en", type="string")
 *     ),
 *     @OA\Property(property="skin_care", type="object", nullable=true,
 *         @OA\Property(property="fa", type="string"),
 *         @OA\Property(property="en", type="string")
 *     ),
 *     @OA\Property(property="sleep", type="object", nullable=true,
 *         @OA\Property(property="fa", type="string"),
 *         @OA\Property(property="en", type="string")
 *     ),
 *     @OA\Property(property="created_at", type="string", format="date-time"),
 *     @OA\Property(property="updated_at", type="string", format="date-time")
 * )
 */
class PhaseContent extends Model
{
    /**
     * The nine content sections stored per phase, in display order. The value
     * is the localizable field; new sections can be added here without
     * touching the read/serialize logic.
     */
    public const SECTIONS = [
        'symptom_prediction',
        'vaginal_discharge',
        'fertility',
        'hormonal_changes',
        'sex_tips',
        'nutrition',
        'exercise',
        'skin_care',
        'sleep',
    ];

    protected $fillable = [
        'phase',
        ...self::SECTIONS,
    ];

    protected function casts(): array
    {
        return array_fill_keys(self::SECTIONS, 'array');
    }

    /**
     * Fetch the content row for a given sub-phase value (CycleSubphase).
     */
    public static function getByPhase(string $phase): ?self
    {
        return self::where('phase', $phase)->first();
    }

    /**
     * Localized value for a section, falling back to the raw value when the
     * requested locale is missing so a half-translated row still renders.
     */
    public function getLocalizedContent(string $field, string $locale = 'fa'): mixed
    {
        $content = $this->{$field};

        if (is_array($content)) {
            return $content[$locale] ?? $content['fa'] ?? $content['en'] ?? null;
        }

        return $content;
    }

    /**
     * The valid sub-phase keys a row may use.
     *
     * @return array<int, string>
     */
    public static function phases(): array
    {
        return CycleSubphase::values();
    }
}

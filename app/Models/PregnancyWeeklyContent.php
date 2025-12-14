<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

/**
 * @OA\Schema(
 *     schema="PregnancyWeeklyContent",
 *     type="object",
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="week_number", type="integer", example=12),
 *     @OA\Property(property="fetal_development", type="object", nullable=true,
 *         @OA\Property(property="en", type="string"),
 *         @OA\Property(property="fa", type="string")
 *     ),
 *     @OA\Property(property="mother_body_changes", type="object", nullable=true,
 *         @OA\Property(property="en", type="string"),
 *         @OA\Property(property="fa", type="string")
 *     ),
 *     @OA\Property(property="dos_and_donts", type="object", nullable=true,
 *         @OA\Property(property="en", type="string"),
 *         @OA\Property(property="fa", type="string")
 *     ),
 *     @OA\Property(property="care_plan", type="object", nullable=true,
 *         @OA\Property(property="en", type="string"),
 *         @OA\Property(property="fa", type="string")
 *     ),
 *     @OA\Property(property="body_adaptation", type="object", nullable=true,
 *         @OA\Property(property="en", type="string"),
 *         @OA\Property(property="fa", type="string")
 *     ),
 *     @OA\Property(property="emotional_status", type="object", nullable=true,
 *         @OA\Property(property="en", type="string"),
 *         @OA\Property(property="fa", type="string")
 *     ),
 *     @OA\Property(property="key_nutrition", type="object", nullable=true,
 *         @OA\Property(property="en", type="string"),
 *         @OA\Property(property="fa", type="string")
 *     ),
 *     @OA\Property(property="physical_activity", type="object", nullable=true,
 *         @OA\Property(property="en", type="string"),
 *         @OA\Property(property="fa", type="string")
 *     ),
 *     @OA\Property(property="tests_and_checkups", type="object", nullable=true,
 *         @OA\Property(property="en", type="string"),
 *         @OA\Property(property="fa", type="string")
 *     ),
 *     @OA\Property(property="faq", type="object", nullable=true,
 *         @OA\Property(property="en", type="array", @OA\Items(type="object")),
 *         @OA\Property(property="fa", type="array", @OA\Items(type="object"))
 *     ),
 *     @OA\Property(property="created_at", type="string", format="date-time"),
 *     @OA\Property(property="updated_at", type="string", format="date-time")
 * )
 */
class PregnancyWeeklyContent extends Model
{
    protected $table = 'pregnancy_weekly_content';

    protected $fillable = [
        'week_number',
        'fetal_development',
        'mother_body_changes',
        'dos_and_donts',
        'care_plan',
        'body_adaptation',
        'emotional_status',
        'key_nutrition',
        'physical_activity',
        'tests_and_checkups',
        'faq',
    ];

    protected function casts(): array
    {
        return [
            'fetal_development' => 'array',
            'mother_body_changes' => 'array',
            'dos_and_donts' => 'array',
            'care_plan' => 'array',
            'body_adaptation' => 'array',
            'emotional_status' => 'array',
            'key_nutrition' => 'array',
            'physical_activity' => 'array',
            'tests_and_checkups' => 'array',
            'faq' => 'array',
        ];
    }

    /**
     * Get content by week number
     */
    public static function getByWeek(int $week): ?self
    {
        return self::where('week_number', $week)->first();
    }

    /**
     * Get localized content for a specific field
     */
    public function getLocalizedContent(string $field, string $locale = 'en'): mixed
    {
        $content = $this->{$field};

        if (is_array($content) && isset($content[$locale])) {
            return $content[$locale];
        }

        return $content;
    }
}

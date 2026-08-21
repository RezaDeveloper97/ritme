<?php

namespace App\Models;

use App\Models\Concerns\HasLocalizedContent;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

/**
 * One box on an in-app text screen: a bilingual heading + body, plus an
 * optional call-to-action link.
 *
 * `group` says which screen the box belongs to — privacy, terms, about, or
 * help. All four render identically in the app, so they share one table and one
 * admin screen.
 *
 * @OA\Schema(
 *     schema="InfoSection",
 *     type="object",
 *
 *     @OA\Property(property="id", type="integer", example=1),
 *     @OA\Property(property="heading", type="string", example="پشتیبانی"),
 *     @OA\Property(property="body", type="string", example="اگه سؤالی داری برامون بنویس."),
 *     @OA\Property(property="link_label", type="string", nullable=true, example="ارسال ایمیل"),
 *     @OA\Property(property="link_url", type="string", nullable=true, example="mailto:support@ritmesalamat.com")
 * )
 */
class InfoSection extends Model
{
    use HasLocalizedContent;

    public const GROUP_PRIVACY = 'privacy';

    public const GROUP_TERMS = 'terms';

    public const GROUP_ABOUT = 'about';

    public const GROUP_HELP = 'help';

    /** Every screen this table can feed, in the order the admin panel lists them. */
    public const GROUPS = [
        self::GROUP_HELP,
        self::GROUP_PRIVACY,
        self::GROUP_TERMS,
        self::GROUP_ABOUT,
    ];

    /** Persian screen names for the admin panel. */
    public const GROUP_LABELS = [
        self::GROUP_HELP => 'راهنما و پشتیبانی',
        self::GROUP_PRIVACY => 'حریم خصوصی',
        self::GROUP_TERMS => 'قوانین و مقررات',
        self::GROUP_ABOUT => 'درباره ریتمی',
    ];

    protected $fillable = [
        'group',
        'key',
        'heading',
        'body',
        'link_label',
        'link_url',
        'is_active',
        'sort_order',
    ];

    protected function casts(): array
    {
        return [
            'heading' => 'array',
            'body' => 'array',
            'link_label' => 'array',
            'link_url' => 'string',
            'is_active' => 'boolean',
            'sort_order' => 'integer',
        ];
    }

    public static function isGroup(?string $group): bool
    {
        return $group !== null && in_array($group, self::GROUPS, true);
    }

    public static function groupLabel(string $group): string
    {
        return self::GROUP_LABELS[$group] ?? $group;
    }

    /** Boxes of a single screen. */
    public function scopeInGroup(Builder $query, string $group): Builder
    {
        return $query->where('group', $group);
    }

    /** Display order: `sort_order` first, then insertion order for ties. */
    public function scopeOrdered(Builder $query): Builder
    {
        return $query->orderBy('sort_order')->orderBy('id');
    }

    public function scopeActive(Builder $query): Builder
    {
        return $query->where('is_active', true);
    }

    /**
     * The box as the API hands it to a client: one locale, and the link only
     * when it actually points somewhere.
     *
     * @return array<string, mixed>
     */
    public function toLocalizedArray(string $locale): array
    {
        $label = $this->localized('link_label', $locale);

        return [
            'id' => $this->id,
            'heading' => $this->localized('heading', $locale),
            'body' => $this->localized('body', $locale),
            // A URL with no caption would render an unlabelled button, so both
            // halves have to be present for the client to show anything.
            'link_label' => $this->link_url ? ($label ?: null) : null,
            'link_url' => $label ? $this->link_url : null,
        ];
    }
}

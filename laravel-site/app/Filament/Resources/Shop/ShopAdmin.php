<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop;

use App\Filament\Auth\AdminAccess;
use App\Filament\Auth\AdminRole;
use App\Support\Money\Money;
use DateTimeInterface;
use Filament\Facades\Filament;
use Filament\Forms\Components\TextInput;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Facades\Gate;

/**
 * Shared bits of the shop admin (L6-06): navigation group, who may edit what, Jalali dates, money in tomans (inputs
 * in tomans ⇄ Money in integer rials) and masking of personal data in lists.
 */
final class ShopAdmin
{
    public const NAV_GROUP = 'فروشگاه';

    /** Highest toman amount accepted by a price input (≈ the 4 billion toman cap of the shop settings). */
    public const MAX_TOMAN = 4_000_000_000;

    public static function user(): ?Model
    {
        $user = Filament::auth()->user();

        return $user instanceof Model ? $user : null;
    }

    public static function can(string $ability, mixed $subject): bool
    {
        return Gate::forUser(Filament::auth()->user())->allows($ability, $subject);
    }

    /**
     * Catalog content (everything but the SEO tab of products and categories): shop managers and super-admins.
     */
    public static function canEditContent(mixed $user = null): bool
    {
        return AdminAccess::allows($user ?? Filament::auth()->user(), [AdminRole::ShopManager]);
    }

    /**
     * The SEO tab of products and categories: shop managers, SEO managers and super-admins.
     */
    public static function canEditSeo(mixed $user = null): bool
    {
        return AdminAccess::allows($user ?? Filament::auth()->user(), [AdminRole::ShopManager, AdminRole::SeoManager]);
    }

    public static function date(mixed $state, string $format = 'Y/m/d H:i'): ?string
    {
        return $state instanceof DateTimeInterface ? jdate($state, $format) : null;
    }

    public static function money(mixed $state): ?string
    {
        return $state instanceof Money ? $state->format() : null;
    }

    /** Money → the toman integer shown in an input (null stays null). */
    public static function toToman(mixed $state): ?int
    {
        return $state instanceof Money ? $state->toToman() : null;
    }

    /** A toman input value → Money in rials (empty → null). */
    public static function fromToman(mixed $value): ?Money
    {
        return is_numeric($value) && (int) $value >= 0 ? Money::fromToman((int) $value) : null;
    }

    public static function tomanInput(string $name, string $label): TextInput
    {
        return TextInput::make($name)
            ->label($label)
            ->integer()
            ->minValue(0)
            ->maxValue(self::MAX_TOMAN)
            ->suffix('تومان')
            ->extraInputAttributes(['dir' => 'ltr']);
    }

    /**
     * A name as lists show it: first letter of each word + dots («س… ر…»); the full name is only on the order page.
     */
    public static function maskName(string $name): string
    {
        $parts = preg_split('/\s+/u', trim($name), -1, PREG_SPLIT_NO_EMPTY) ?: [];

        return implode(' ', array_map(static fn (string $part): string => mb_substr($part, 0, 1).'…', $parts));
    }
}

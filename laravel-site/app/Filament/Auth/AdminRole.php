<?php

declare(strict_types=1);

namespace App\Filament\Auth;

/**
 * Admin roles (spatie/laravel-permission, guard "web"). Any role grants access to the panel; what a role may do is
 * decided per resource by its policy (deny by default, see App\Filament\Policies\AdminPolicy). Super-admin passes
 * every policy check except the ones a policy hard-denies (e.g. editing the activity log).
 */
enum AdminRole: string
{
    case SuperAdmin = 'super-admin';
    case Editor = 'editor';
    case SeoManager = 'seo-manager';
    case ShopManager = 'shop-manager';
    case DirectoryManager = 'directory-manager';
    case Support = 'support';

    /**
     * @return list<string>
     */
    public static function values(): array
    {
        return array_map(static fn (self $role): string => $role->value, self::cases());
    }

    public function label(): string
    {
        return match ($this) {
            self::SuperAdmin => 'مدیر کل',
            self::Editor => 'ویراستار محتوا',
            self::SeoManager => 'مدیر سئو',
            self::ShopManager => 'مدیر فروشگاه',
            self::DirectoryManager => 'مدیر راهنمای مادر و کودک',
            self::Support => 'پشتیبانی',
        };
    }

    /**
     * @return array<string, string> value => Persian label, for selects
     */
    public static function options(): array
    {
        $options = [];
        foreach (self::cases() as $role) {
            $options[$role->value] = $role->label();
        }

        return $options;
    }
}

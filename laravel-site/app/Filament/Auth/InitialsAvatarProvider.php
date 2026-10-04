<?php

declare(strict_types=1);

namespace App\Filament\Auth;

use Filament\AvatarProviders\Contracts\AvatarProvider;
use Filament\Facades\Filament;
use Illuminate\Database\Eloquent\Model;

/**
 * Replaces Filament's default ui-avatars.com provider (an external request) with an inline SVG data URI showing the
 * user's first letter on the brand colour.
 */
final class InitialsAvatarProvider implements AvatarProvider
{
    public function get(Model $record): string
    {
        $name = trim(Filament::getNameForDefaultAvatar($record));
        $initial = $name === '' ? '?' : mb_strtoupper(mb_substr($name, 0, 1));
        $color = config('filament.admin.brand_color', '#6e54f0');
        $fill = is_string($color) && preg_match('/^#[0-9a-f]{6}$/i', $color) === 1 ? $color : '#6e54f0';

        $svg = '<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64">'
            .'<rect width="64" height="64" fill="'.$fill.'"/>'
            .'<text x="50%" y="50%" dy=".35em" text-anchor="middle" font-family="Vazirmatn, sans-serif" font-size="30" fill="#fff">'
            .htmlspecialchars($initial, ENT_XML1 | ENT_QUOTES, 'UTF-8')
            .'</text></svg>';

        return 'data:image/svg+xml;base64,'.base64_encode($svg);
    }
}

<?php

declare(strict_types=1);

namespace App\Domain\Settings\View;

use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\LayoutSettings;
use Illuminate\Contracts\View\View;

/**
 * Shares `$siteSettings` (LayoutSettings: general, contact, social, app links, legal) with the layout views.
 */
final class LayoutSettingsComposer
{
    public function __construct(private readonly SettingsRepository $settings) {}

    public function compose(View $view): void
    {
        $view->with('siteSettings', LayoutSettings::from($this->settings->all()));
    }
}

<?php

declare(strict_types=1);

namespace App\Providers\Domain;

use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Models\Setting;
use App\Domain\Settings\Observers\SettingObserver;
use App\Domain\Settings\Repositories\CachedSettingsRepository;
use App\Domain\Settings\Repositories\EloquentSettingsRepository;
use App\Domain\Settings\View\LayoutSettingsComposer;
use App\Providers\DomainServiceProvider;

final class SettingsServiceProvider extends DomainServiceProvider
{
    protected array $repositories = [
        SettingsRepository::class => [EloquentSettingsRepository::class, CachedSettingsRepository::class],
    ];

    protected array $observers = [
        Setting::class => SettingObserver::class,
    ];

    protected array $composers = [
        'layouts.*' => LayoutSettingsComposer::class,
    ];
}

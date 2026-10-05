<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing\Actions;

use App\Domain\Seo\Indexing\IndexNow\IndexNowKey;
use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Enums\SettingGroup;

/** A fresh IndexNow key (e.g. after it leaked); the old /{key}.txt stops answering immediately. Returns the new key. */
final class RegenerateIndexNowKey
{
    public function __construct(private readonly UpdateSettings $update) {}

    public function handle(): string
    {
        $key = IndexNowKey::generate();
        $this->update->handle(SettingGroup::Seo, ['indexnow_key' => $key]);

        return $key;
    }
}

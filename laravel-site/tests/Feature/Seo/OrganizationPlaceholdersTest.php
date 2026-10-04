<?php

declare(strict_types=1);

use App\Domain\Seo\Schema\Nodes\OrganizationNode;
use App\Domain\Settings\Contracts\SettingsRepository;
use Database\Seeders\SettingsSeeder;

it('keeps seeded [placeholder] contact values out of the Organization node', function (): void {
    $this->seed(SettingsSeeder::class);

    $node = OrganizationNode::make(app(SettingsRepository::class)->all(), 'https://ritme.test');

    expect(json_encode($node, JSON_UNESCAPED_UNICODE))->not->toContain('[');
});

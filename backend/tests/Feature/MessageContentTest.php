<?php

namespace Tests\Feature;

use App\Models\Admin;
use App\Models\MessageContent;
use App\Services\MessageSystem\Support\MessageContentRepository;
use Database\Seeders\MessageContentSeeder;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class MessageContentTest extends TestCase
{
    use RefreshDatabase;

    public function test_seeder_populates_every_provider_group(): void
    {
        $this->seed(MessageContentSeeder::class);

        $groups = MessageContent::query()->distinct()->pluck('group');

        foreach ([
            'cycle_base_non_ttc', 'cycle_base_ttc', 'cycle_override',
            'pregnancy_trimester', 'pregnancy_week', 'pregnancy_override',
            'nutrition_cycle', 'sleep_cycle', 'exercise_cycle',
            'correlation_cycle', 'pattern',
        ] as $group) {
            $this->assertContains($group, $groups, "Missing seeded group: {$group}");
        }
    }

    public function test_repository_returns_db_payload_when_approved(): void
    {
        MessageContent::create([
            'group' => 'cycle_base_non_ttc',
            'item_key' => 'menstruation',
            'locale' => 'fa',
            'payload' => ['short' => 'متن سفارشی'],
            'is_active' => true,
            'is_approved' => true,
        ]);

        $repo = app(MessageContentRepository::class);
        $resolved = $repo->resolve('cycle_base_non_ttc', 'menstruation', 'fa', ['short' => 'پیش‌فرض']);

        $this->assertSame('متن سفارشی', $resolved['short']);
    }

    public function test_repository_falls_back_when_unapproved_or_missing(): void
    {
        MessageContent::create([
            'group' => 'cycle_base_non_ttc',
            'item_key' => 'menstruation',
            'locale' => 'fa',
            'payload' => ['short' => 'در انتظار تأیید'],
            'is_active' => true,
            'is_approved' => false, // not yet approved
        ]);

        $repo = app(MessageContentRepository::class);

        // Unapproved row is ignored -> fallback.
        $this->assertSame('پیش‌فرض', $repo->resolve('cycle_base_non_ttc', 'menstruation', 'fa', ['short' => 'پیش‌فرض'])['short']);
        // Missing row -> fallback.
        $this->assertSame('پیش‌فرض', $repo->resolve('cycle_base_non_ttc', 'ovulation', 'fa', ['short' => 'پیش‌فرض'])['short']);
    }

    public function test_admin_edit_updates_payload_preserving_list_structure(): void
    {
        $admin = Admin::create([
            'name' => 'A', 'email' => 'a@ritme.test', 'password' => 'secret123',
            'role' => Admin::ROLE_SUPER, 'is_active' => true,
        ]);

        $row = MessageContent::create([
            'group' => 'cycle_base_non_ttc',
            'item_key' => 'menstruation',
            'locale' => 'fa',
            'payload' => ['short' => 'قدیمی', 'dos' => ['الف', 'ب']],
            'is_active' => true,
            'is_approved' => true,
        ]);

        $this->actingAs($admin, 'admin')->put("/admin/messages/{$row->id}", [
            'payload' => [
                'short' => 'جدید',
                'dos' => "یک\nدو\nسه",
            ],
        ])->assertRedirect();

        $fresh = $row->fresh();
        $this->assertSame('جدید', $fresh->payload['short']);
        $this->assertSame(['یک', 'دو', 'سه'], $fresh->payload['dos']); // newline-split list
    }

    public function test_approve_toggle_flips_flag(): void
    {
        $admin = Admin::create([
            'name' => 'A', 'email' => 'a@ritme.test', 'password' => 'secret123',
            'role' => Admin::ROLE_SUPER, 'is_active' => true,
        ]);

        $row = MessageContent::create([
            'group' => 'pattern', 'item_key' => 'severe_pain', 'locale' => 'fa',
            'payload' => ['message' => 'x'], 'is_active' => true, 'is_approved' => false,
        ]);

        $this->actingAs($admin, 'admin')->post("/admin/messages/{$row->id}/approve");

        $this->assertTrue($row->fresh()->is_approved);
    }
}

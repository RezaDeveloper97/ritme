<?php

declare(strict_types=1);

namespace Database\Seeders;

use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Settings\Models\Setting;
use Illuminate\Database\Seeder;

/**
 * Initial site settings: copy from the design export (design/html) and its bracketed placeholders
 * («[ایمیل پشتیبانی]» …) where the real value is not known yet. Idempotent and non-destructive:
 * only missing keys are created, so admin edits survive a re-seed.
 */
final class SettingsSeeder extends Seeder
{
    public function run(): void
    {
        foreach ($this->values() as $group => $values) {
            $group = SettingGroup::fromName($group);
            $normalized = array_intersect_key($group->data($values)->toArray(), $values);

            foreach ($normalized as $key => $value) {
                $group->assertKey($key);
                Setting::query()->firstOrCreate(['group' => $group->value, 'key' => $key], ['value' => $value]);
            }
        }
    }

    /**
     * @return array<string, array<string, mixed>>
     */
    public function values(): array
    {
        return [
            'general' => [
                'site_name' => 'ریتمی',
                'alternate_name' => 'Ritme',
                'tagline' => 'همراه سلامت زنان، از اولین پریود تا یائسگی',
                'footer_note' => 'جایگزین پزشک نیستیم؛ کنارت هستیم.',
                'emergency_number' => '115',
            ],
            'contact' => [
                'support_email' => '[ایمیل پشتیبانی]',
                'partnership_email' => '[ایمیل همکاری]',
                'phone' => '[شماره تماس]',
                'working_hours' => '[روزها و ساعت کاری]',
                'response_time' => '[ساعت پاسخگویی]',
                'address' => '[نشانی دفتر]',
                'address_note' => '[نیاز به هماهنگی قبلی؟]',
            ],
            // Footer links are href="#" in the design: unknown until the admin fills them in.
            'social' => [
                'instagram' => null,
                'telegram' => null,
                'linkedin' => null,
            ],
            'app_links' => [
                'bazaar' => null,
                'myket' => null,
                'google_play' => null,
                'app_store' => null,
                'web_app' => null,
            ],
            'seo' => [
                'title_template' => '%s — ریتمی',
                'separator' => '—',
                'default_title' => 'ریتمی — همراه سلامت زنان، از اولین پریود تا یائسگی',
                'default_description' => 'ریتمی اپ پیگیری چرخه، بارداری، مادری و یائسگی است؛ با زبان احتمالی، بدون ترساندن، و با داده‌ای که پیش خودت می‌ماند.',
                'default_og_media_id' => null,
                'twitter_handle' => null,
                'verification' => [],
            ],
            'organization' => [
                'legal_name' => 'ریتمی',
                'logo_media_id' => null,
                'founding_date' => null,
                'same_as' => [],
                'contact_point' => ['contact_type' => 'customer support', 'telephone' => null, 'email' => null],
            ],
            'legal' => [
                'enamad_code' => '[اینماد]',
                'enamad_html' => null,
                'enamad_allowed_tags' => ['a', 'img'],
                'data_protection_email' => '[ایمیل مسئول داده]',
            ],
            'pwa' => [
                'name' => 'ریتمی',
                'short_name' => 'ریتمی',
                'description' => 'همراه سلامت زنان، از اولین پریود تا یائسگی',
                'theme_color' => '#17112B',
                'background_color' => '#FFFFFF',
            ],
        ];
    }
}

<?php

namespace Database\Seeders;

use App\Models\Language;
use App\Services\Language\LanguageRegistry;
use Illuminate\Database\Seeder;

/**
 * Seeds the two locales the product shipped with, so an existing install keeps
 * behaving identically once the registry takes over from the hardcoded fa/en.
 *
 * Idempotent: rows are matched on their code and only inserted, so re-running
 * never resets a name, direction or the admin's choice of default.
 */
class LanguageSeeder extends Seeder
{
    public function run(): void
    {
        foreach (LanguageRegistry::BOOTSTRAP as $index => $row) {
            Language::firstOrCreate(
                ['code' => $row['code']],
                [
                    'name' => $row['name'],
                    'english_name' => $row['english_name'],
                    'direction' => $row['direction'],
                    'is_active' => true,
                    'is_default' => $row['is_default'],
                    'sort_order' => $index,
                ]
            );
        }

        app(LanguageRegistry::class)->flush();
    }
}

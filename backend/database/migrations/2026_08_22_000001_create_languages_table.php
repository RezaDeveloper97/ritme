<?php

use App\Services\Language\LanguageRegistry;
use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * The registry of locales the product ships.
     *
     * This table is the single source of truth for "which languages exist":
     * the admin panel renders one content field per active row, the API
     * resolves Accept-Language against it, and the frontend builds its locale
     * switcher and URL prefixes from it. Nothing may hardcode fa/en again.
     *
     * `fa` and `en` are seeded (LanguageSeeder) so an existing install keeps
     * behaving exactly as before; admins add further rows from
     * /admin/languages, which also generates the locale's translation files.
     */
    public function up(): void
    {
        Schema::create('languages', function (Blueprint $table) {
            $table->id();
            // BCP-47-ish code used in URLs, JSON content keys and Accept-Language
            // (e.g. "fa", "en", "ar", "pt-BR"). Lowercased on write.
            $table->string('code', 12)->unique();
            $table->string('name', 60);                  // endonym, shown to users ("فارسی")
            $table->string('english_name', 60);          // for the admin list ("Persian")
            $table->string('direction', 3)->default('ltr'); // App\Enums\TextDirection
            $table->boolean('is_active')->default(true);    // false = hidden from the app
            $table->boolean('is_default')->default(false);  // exactly one row is true
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
        });

        // Seeded here rather than only in LanguageSeeder: an empty registry
        // would leave the first language an admin adds as the *only* language,
        // silently dropping fa/en from an existing install. Every deploy runs
        // migrations, so this guarantees the table is never empty.
        $now = now();

        DB::table('languages')->insert(array_map(
            fn (array $row, int $index): array => [
                'code' => $row['code'],
                'name' => $row['name'],
                'english_name' => $row['english_name'],
                'direction' => $row['direction'],
                'is_active' => true,
                'is_default' => $row['is_default'],
                'sort_order' => $index,
                'created_at' => $now,
                'updated_at' => $now,
            ],
            LanguageRegistry::BOOTSTRAP,
            array_keys(LanguageRegistry::BOOTSTRAP),
        ));
    }

    public function down(): void
    {
        Schema::dropIfExists('languages');
    }
};

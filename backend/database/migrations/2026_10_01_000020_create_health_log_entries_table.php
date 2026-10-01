<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00020_health_log_entries.sql
     * (docs/go-migration/migrations.md): the log taxonomy v2 storage
     * (B-N3-01) — one row per logged value (user, day, category, param,
     * item) — written by the Go-only /api/v1/logs endpoints, plus the same
     * backfill of every daily_health_logs row (idempotent INSERT IGNORE).
     * The backfill uses MariaDB JSON functions, so it only runs on
     * mysql/mariadb (the sqlite test database has no legacy rows anyway).
     * The statements below are backend-go's taxonomy.BackfillSQL()
     * verbatim (TestBackfillSQL_InMigrations). No model or routes here.
     */
    public function up(): void
    {
        $mariadb = in_array(DB::getDriverName(), ['mysql', 'mariadb'], true);

        Schema::create('health_log_entries', function (Blueprint $table) use ($mariadb) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->date('log_date');
            $table->string('category', 32);
            $table->string('param', 32);
            $item = $table->string('item', 191)->default('');
            if ($mariadb) {                                       // codes compare exactly (sqlite: no such collation)
                $item->charset('utf8mb4')->collation('utf8mb4_bin');
            }
            $table->string('value_code')->nullable();
            $table->decimal('value_num', 8, 2)->nullable();
            $table->text('value_text')->nullable();
            $table->string('source', 16)->default('manual');   // manual|legacy|voice
            $table->timestamps();

            $table->unique(['user_id', 'log_date', 'category', 'param', 'item'], 'health_log_entries_slot_unique');
            $table->index(['user_id', 'category', 'param', 'log_date'], 'health_log_entries_user_param_date_index');
        });

        if (! $mariadb) {
            return;
        }
        foreach (array_filter(array_map('trim', explode(";\n", rtrim(self::BACKFILL, ";\n")))) as $statement) {
            DB::statement($statement);
        }
    }

    /**
     * Reverse the migrations. daily_health_logs was never modified, so
     * dropping the table loses nothing.
     */
    public function down(): void
    {
        Schema::dropIfExists('health_log_entries');
    }

    private const BACKFILL = <<<'SQL'
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'bleeding', 'flow', '', CASE l.`bleeding_intensity` WHEN 'high' THEN 'heavy' WHEN 'low' THEN 'light' WHEN 'very_high' THEN 'very_heavy' ELSE l.`bleeding_intensity` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`bleeding_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'bleeding', 'color', '', l.`blood_color`, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`blood_color` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'bleeding', 'clots', '', IF(l.`has_clots`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`has_clots` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'bleeding', 'clot_size', '', CASE l.`clots_amount` WHEN 'high' THEN 'large' WHEN 'low' THEN 'small' ELSE l.`clots_amount` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`clots_amount` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'bleeding', 'spotting', '', IF(l.`spotting`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`spotting` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'bleeding', 'odor', '', l.`bleeding_smell`, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`bleeding_smell` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'pain', 'location', 'head', CASE l.`headache_intensity` WHEN 'high' THEN 'severe' WHEN 'low' THEN 'mild' WHEN 'medium' THEN 'moderate' ELSE l.`headache_intensity` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`headache_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'pain', 'location', 'abdomen', CASE l.`stomach_ache_intensity` WHEN 'high' THEN 'severe' WHEN 'low' THEN 'mild' WHEN 'medium' THEN 'moderate' ELSE l.`stomach_ache_intensity` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`stomach_ache_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'pain', 'location', 'pelvis', CASE l.`pelvic_pain_intensity` WHEN 'high' THEN 'severe' WHEN 'low' THEN 'mild' WHEN 'medium' THEN 'moderate' ELSE l.`pelvic_pain_intensity` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`pelvic_pain_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'pain', 'location', 'breast', CASE l.`breast_pain_intensity` WHEN 'high' THEN 'severe' WHEN 'low' THEN 'mild' WHEN 'medium' THEN 'moderate' ELSE l.`breast_pain_intensity` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`breast_pain_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'pain', 'location', 'back', CASE l.`back_pain_intensity` WHEN 'high' THEN 'severe' WHEN 'low' THEN 'mild' WHEN 'medium' THEN 'moderate' ELSE l.`back_pain_intensity` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`back_pain_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'pain', 'location', 'ovary', CASE l.`ovarian_pain_intensity` WHEN 'high' THEN 'severe' WHEN 'low' THEN 'mild' WHEN 'medium' THEN 'moderate' ELSE l.`ovarian_pain_intensity` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`ovarian_pain_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'symptoms', 'digestive', 'nausea', CASE l.`nausea_intensity` WHEN 'high' THEN 'severe' WHEN 'low' THEN 'mild' WHEN 'medium' THEN 'moderate' ELSE l.`nausea_intensity` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`nausea_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'symptoms', 'digestive', 'bloating', CASE l.`bloating_intensity` WHEN 'high' THEN 'severe' WHEN 'low' THEN 'mild' WHEN 'medium' THEN 'moderate' ELSE l.`bloating_intensity` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`bloating_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'symptoms', 'digestive', 'diarrhea', IF(l.`diarrhea`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`diarrhea` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'symptoms', 'digestive', 'constipation', IF(l.`constipation`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`constipation` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'appetite_energy', 'appetite', '', CASE l.`appetite_change` WHEN 'gain' THEN 'increased' WHEN 'loss' THEN 'decreased' ELSE l.`appetite_change` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`appetite_change` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'appetite_energy', 'cravings', 'any', IF(l.`food_craving`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`food_craving` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'symptoms', 'general', 'breast_tenderness', CASE l.`breast_sensitivity_intensity` WHEN 'high' THEN 'severe' WHEN 'low' THEN 'mild' WHEN 'medium' THEN 'moderate' ELSE l.`breast_sensitivity_intensity` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`breast_sensitivity_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'urogenital', 'symptoms', 'vaginal_dryness', IF(l.`vaginal_dryness`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`vaginal_dryness` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'urogenital', 'symptoms', 'vaginal_burning', CASE l.`vaginal_burning_intensity` WHEN 'high' THEN 'severe' WHEN 'low' THEN 'mild' WHEN 'medium' THEN 'moderate' ELSE l.`vaginal_burning_intensity` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`vaginal_burning_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'urogenital', 'symptoms', 'vaginal_burning', IF(l.`vaginal_burning`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`vaginal_burning` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'urogenital', 'symptoms', 'vaginal_itching', CASE l.`vaginal_itching_intensity` WHEN 'high' THEN 'severe' WHEN 'low' THEN 'mild' WHEN 'medium' THEN 'moderate' ELSE l.`vaginal_itching_intensity` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`vaginal_itching_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'urogenital', 'symptoms', 'vaginal_itching', IF(l.`vaginal_itching`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`vaginal_itching` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'urogenital', 'symptoms', 'odor_change', IF(l.`vaginal_smell_change`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`vaginal_smell_change` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'urogenital', 'urination', '', CASE l.`urination_change` WHEN 'decrease' THEN 'decreased' WHEN 'increase' THEN 'increased' ELSE l.`urination_change` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`urination_change` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'urogenital', 'symptoms', 'urination_burning', CASE l.`urination_burning_intensity` WHEN 'high' THEN 'severe' WHEN 'low' THEN 'mild' WHEN 'medium' THEN 'moderate' ELSE l.`urination_burning_intensity` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`urination_burning_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'skin_hair', 'symptoms', 'acne', IF(l.`acne`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`acne` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'skin_hair', 'symptoms', 'oily_skin', IF(l.`oily_skin`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`oily_skin` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'skin_hair', 'symptoms', 'hair_loss', IF(l.`hair_loss`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`hair_loss` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'symptoms', 'general', 'swelling', IF(l.`swelling`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`swelling` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'symptoms', 'general', 'fatigue', IF(l.`fatigue`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`fatigue` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'symptoms', 'general', 'dizziness', IF(l.`dizziness`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`dizziness` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'symptoms', 'general', 'hot_flashes', IF(l.`hot_flashes`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`hot_flashes` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'symptoms', 'general', 'chills', IF(l.`chills`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`chills` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'mood', 'moods', LEFT(COALESCE(t.`v`, t.`j`), 191), 'yes', NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l,
  JSON_TABLE(l.`moods`, '$[*]' COLUMNS (`v` LONGTEXT PATH '$' NULL ON ERROR, `j` JSON PATH '$')) t
WHERE JSON_TYPE(l.`moods`) = 'ARRAY' AND JSON_TYPE(t.`j`) <> 'NULL';
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'mood', 'moods', LEFT(COALESCE(t.`v`, t.`j`), 191), 'yes', NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l,
  JSON_TABLE(l.`moods`, '$.*' COLUMNS (`v` LONGTEXT PATH '$' NULL ON ERROR, `j` JSON PATH '$')) t
WHERE JSON_TYPE(l.`moods`) = 'OBJECT' AND JSON_TYPE(t.`j`) <> 'NULL';
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'mood', 'moods', LEFT(JSON_UNQUOTE(l.`moods`), 191), 'yes', NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE JSON_TYPE(l.`moods`) NOT IN ('ARRAY', 'OBJECT', 'NULL');
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'sleep', 'duration', '', l.`sleep_duration`, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`sleep_duration` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'sleep', 'quality', '', CASE l.`sleep_quality` WHEN 'bad' THEN 'poor' WHEN 'medium' THEN 'fair' ELSE l.`sleep_quality` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`sleep_quality` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'activity', 'types', LEFT(COALESCE(t.`v`, t.`j`), 191), 'yes', NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l,
  JSON_TABLE(l.`exercise_type`, '$[*]' COLUMNS (`v` LONGTEXT PATH '$' NULL ON ERROR, `j` JSON PATH '$')) t
WHERE JSON_TYPE(l.`exercise_type`) = 'ARRAY' AND JSON_TYPE(t.`j`) <> 'NULL';
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'activity', 'types', LEFT(COALESCE(t.`v`, t.`j`), 191), 'yes', NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l,
  JSON_TABLE(l.`exercise_type`, '$.*' COLUMNS (`v` LONGTEXT PATH '$' NULL ON ERROR, `j` JSON PATH '$')) t
WHERE JSON_TYPE(l.`exercise_type`) = 'OBJECT' AND JSON_TYPE(t.`j`) <> 'NULL';
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'activity', 'types', LEFT(JSON_UNQUOTE(l.`exercise_type`), 191), 'yes', NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE JSON_TYPE(l.`exercise_type`) NOT IN ('ARRAY', 'OBJECT', 'NULL');
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'activity', 'duration', '', NULL, l.`exercise_duration`, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`exercise_duration` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'activity', 'intensity', '', l.`exercise_intensity`, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`exercise_intensity` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'sex', 'symptoms', LEFT(COALESCE(t.`v`, t.`j`), 191), 'yes', NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l,
  JSON_TABLE(l.`sexual_activities`, '$[*]' COLUMNS (`v` LONGTEXT PATH '$' NULL ON ERROR, `j` JSON PATH '$')) t
WHERE JSON_TYPE(l.`sexual_activities`) = 'ARRAY' AND JSON_TYPE(t.`j`) <> 'NULL';
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'sex', 'symptoms', LEFT(COALESCE(t.`v`, t.`j`), 191), 'yes', NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l,
  JSON_TABLE(l.`sexual_activities`, '$.*' COLUMNS (`v` LONGTEXT PATH '$' NULL ON ERROR, `j` JSON PATH '$')) t
WHERE JSON_TYPE(l.`sexual_activities`) = 'OBJECT' AND JSON_TYPE(t.`j`) <> 'NULL';
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'sex', 'symptoms', LEFT(JSON_UNQUOTE(l.`sexual_activities`), 191), 'yes', NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE JSON_TYPE(l.`sexual_activities`) NOT IN ('ARRAY', 'OBJECT', 'NULL');
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'sex', 'desire', '', l.`sexual_desire`, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`sexual_desire` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'sex', 'intercourse', '', l.`intercourse_type`, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`intercourse_type` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'measurements', 'weight', '', NULL, l.`weight`, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`weight` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'measurements', 'bbt', '', NULL, l.`basal_body_temperature`, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`basal_body_temperature` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'measurements', 'heart_rate', '', NULL, l.`heart_rate`, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`heart_rate` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'measurements', 'bp_systolic', '', NULL, l.`systolic_pressure`, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`systolic_pressure` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'measurements', 'bp_diastolic', '', NULL, l.`diastolic_pressure`, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`diastolic_pressure` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'measurements', 'blood_sugar', '', NULL, l.`blood_sugar`, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`blood_sugar` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'appetite_energy', 'energy', '', l.`energy_level`, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`energy_level` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'discharge', 'color', '', l.`discharge_color`, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`discharge_color` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'discharge', 'consistency', '', CASE l.`discharge_texture` WHEN 'thick' THEN 'sticky' ELSE l.`discharge_texture` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`discharge_texture` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'discharge', 'amount', '', CASE l.`discharge_amount` WHEN 'high' THEN 'heavy' WHEN 'low' THEN 'light' ELSE l.`discharge_amount` END, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`discharge_amount` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'discharge', 'odor', '', l.`discharge_smell`, NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`discharge_smell` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'discharge', 'symptoms', 'itching', IF(l.`discharge_itching`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`discharge_itching` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'discharge', 'symptoms', 'burning', IF(l.`discharge_burning`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`discharge_burning` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'urogenital', 'symptoms', 'frequent_urination', IF(l.`frequent_urination`, 'yes', 'no'), NULL, NULL, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`frequent_urination` IS NOT NULL;
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'meds', 'other', LEFT(k.`k`, 191), NULL, NULL, COALESCE(t.`v`, t.`j`), 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l,
  JSON_TABLE(JSON_KEYS(l.`medications`), '$[*]' COLUMNS (`n` FOR ORDINALITY, `k` LONGTEXT PATH '$')) k,
  JSON_TABLE(l.`medications`, '$.*' COLUMNS (`n` FOR ORDINALITY, `v` LONGTEXT PATH '$' NULL ON ERROR, `j` JSON PATH '$')) t
WHERE JSON_TYPE(l.`medications`) = 'OBJECT' AND k.`n` = t.`n` AND JSON_TYPE(t.`j`) <> 'NULL';
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'meds', 'other', '_raw', NULL, NULL, l.`medications`, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE JSON_TYPE(l.`medications`) NOT IN ('OBJECT', 'NULL');
INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, `value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)
SELECT l.`user_id`, l.`log_date`, 'note', 'text', '', NULL, NULL, l.`notes`, 'legacy', l.`created_at`, l.`updated_at`
FROM `daily_health_logs` l
WHERE l.`notes` IS NOT NULL;
SQL;
};

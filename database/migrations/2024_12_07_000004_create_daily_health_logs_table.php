<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('daily_health_logs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->date('log_date');

            // =========================================
            // 1. قاعدگی و خون‌ریزی (Menstruation & Bleeding)
            // =========================================
            $table->string('bleeding_intensity')->nullable(); // low, medium, high, very_high
            $table->string('blood_color')->nullable(); // bright_red, red, dark_red, brown
            $table->boolean('has_clots')->nullable(); // وجود لخته
            $table->boolean('spotting')->nullable(); // لکه بینی
            $table->string('bleeding_smell')->nullable(); // normal, slightly_unusual, strong_unpleasant

            // =========================================
            // 2. علائم (Symptoms)
            // =========================================

            // الف) دردها (Pains)
            $table->string('headache_intensity')->nullable(); // low, medium, high
            $table->string('stomach_ache_intensity')->nullable();
            $table->string('pelvic_pain_intensity')->nullable();
            $table->string('breast_pain_intensity')->nullable();
            $table->string('back_pain_intensity')->nullable();
            $table->string('ovarian_pain_intensity')->nullable();

            // ب) گوارشی (Digestive)
            $table->string('nausea_intensity')->nullable(); // low, medium, high
            $table->string('bloating_intensity')->nullable();
            $table->boolean('diarrhea')->nullable();
            $table->boolean('constipation')->nullable();
            $table->string('appetite_change')->nullable(); // loss, gain, normal
            $table->boolean('food_craving')->nullable(); // هوس غذایی

            // پ) سینه و تناسلی (Breast & Genital)
            $table->string('breast_sensitivity_intensity')->nullable();
            $table->boolean('vaginal_dryness')->nullable();
            $table->boolean('vaginal_burning')->nullable();
            $table->boolean('vaginal_itching')->nullable();
            $table->boolean('vaginal_smell_change')->nullable();
            $table->string('urination_change')->nullable(); // increase, decrease, normal
            $table->string('urination_burning_intensity')->nullable();

            // ت) پوست و مو (Skin & Hair)
            $table->boolean('acne')->nullable();
            $table->boolean('oily_skin')->nullable();
            $table->boolean('hair_loss')->nullable();
            $table->boolean('swelling')->nullable(); // ورم / پف

            // ث) انرژی و عمومی (Energy & General)
            $table->boolean('fatigue')->nullable();
            $table->boolean('dizziness')->nullable();
            $table->boolean('hot_flashes')->nullable(); // گرگرفتگی
            $table->boolean('chills')->nullable(); // لرز

            // =========================================
            // 3. مود و احساسات (Mood & Emotions)
            // =========================================
            $table->json('moods')->nullable(); // array of: happy, calm, angry, anxious, sad, frustrated, sensitive, bored

            // =========================================
            // 4. خواب (Sleep)
            // =========================================
            $table->string('sleep_duration')->nullable(); // 0_3, 3_6, 6_9, 9_plus
            $table->string('sleep_quality')->nullable(); // good, medium, bad

            // =========================================
            // 5. فعالیت جنسی (Sexual Activity)
            // =========================================
            $table->string('sexual_activity_level')->nullable(); // low, medium, high, very_high

            // =========================================
            // 6. علائم حیاتی و سنجش‌ها (Vital Signs)
            // =========================================
            $table->decimal('weight', 5, 2)->nullable(); // وزن (kg)
            $table->decimal('basal_body_temperature', 4, 2)->nullable(); // دمای پایه بدن (BBT)

            // =========================================
            // 7. ترشحات واژن (Vaginal Discharge)
            // =========================================
            $table->string('discharge_color')->nullable();
            $table->string('discharge_texture')->nullable(); // watery, creamy, egg_white, thick
            $table->string('discharge_amount')->nullable(); // low, medium, high
            $table->string('discharge_smell')->nullable(); // normal, slightly_unusual, strong_unpleasant
            $table->boolean('discharge_itching')->nullable();
            $table->boolean('discharge_burning')->nullable();

            // =========================================
            // 8. گوارش و ادراری (Digestive & Urinary)
            // =========================================
            // Note: Most fields already covered in symptoms section
            $table->boolean('frequent_urination')->nullable(); // تکرر ادرار

            // =========================================
            // 9. دارو و مکمل‌ها (Medications & Supplements)
            // =========================================
            $table->json('medications')->nullable(); // array of medication objects

            // =========================================
            // 10. یادداشت‌ها (Notes)
            // =========================================
            $table->text('notes')->nullable();

            $table->timestamps();

            // یک لاگ در روز برای هر کاربر
            $table->unique(['user_id', 'log_date']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('daily_health_logs');
    }
};

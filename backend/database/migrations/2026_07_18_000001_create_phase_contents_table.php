<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/**
 * DB-driven educational content shown on the "Phase Details" screen the user
 * opens from the daily cycle card. One row per fine-grained cycle sub-phase
 * (App\Enums\CycleSubphase — the value on cycle_view.subphase), with nine
 * bilingual {fa,en} content sections. Editable from the admin panel so the
 * copy can change without a code deploy. Mirrors pregnancy_weekly_content.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('phase_contents', function (Blueprint $table) {
            $table->id();
            // One of App\Enums\CycleSubphase::values() (e.g. menstruation, high_fertility …)
            $table->string('phase')->unique();

            // Content sections (JSON with fa/en translations)
            $table->json('symptom_prediction')->nullable(); // پیش‌بینی علائم
            $table->json('vaginal_discharge')->nullable();   // ترشحات واژن
            $table->json('fertility')->nullable();           // باروری
            $table->json('hormonal_changes')->nullable();    // تغییرات هورمونی
            $table->json('sex_tips')->nullable();            // نکات رابطه جنسی
            $table->json('nutrition')->nullable();           // تغذیه
            $table->json('exercise')->nullable();            // ورزش
            $table->json('skin_care')->nullable();           // مراقبت از پوست
            $table->json('sleep')->nullable();               // خواب

            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('phase_contents');
    }
};

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
        Schema::create('pregnancy_weekly_content', function (Blueprint $table) {
            $table->id();
            $table->integer('week_number')->unique(); // 1-40

            // Content modules (stored as JSON with en/fa translations)
            $table->json('fetal_development')->nullable(); // وضعیت رشد جنین
            $table->json('mother_body_changes')->nullable(); // وضعیت و تغییرات بدن مادر
            $table->json('dos_and_donts')->nullable(); // بایدها و نبایدها
            $table->json('care_plan')->nullable(); // برنامه مراقبتی این هفته
            $table->json('body_adaptation')->nullable(); // تطابق بدن مادر
            $table->json('emotional_status')->nullable(); // وضعیت احساسی و روانی مادر
            $table->json('key_nutrition')->nullable(); // تغذیه کلیدی این هفته
            $table->json('physical_activity')->nullable(); // تحرک و فعالیت بدنی مناسب
            $table->json('tests_and_checkups')->nullable(); // آزمایش‌ها و چکاپ‌های رایج
            $table->json('faq')->nullable(); // سوالات پرتکرار این هفته

            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('pregnancy_weekly_content');
    }
};

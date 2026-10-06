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
     * Twin of backend-go/db/migrations/00035_vitals.sql
     * (docs/go-migration/migrations.md): timed blood pressure, glucose and
     * heart-rate readings and the weekly measurement plan (bloom B-N6-01),
     * used only by the Go internal/vitals package, plus the vitals_alert
     * bp_crisis / glucose_low copy rows (fa + en, needs clinical review) so
     * `make schema-diff` row counts match. No model or routes here.
     */
    public function up(): void
    {
        Schema::create('vital_readings', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('type', 8);                       // bp|glucose|hr
            $table->dateTime('measured_at');
            $table->unsignedSmallInteger('systolic')->nullable();
            $table->unsignedSmallInteger('diastolic')->nullable();
            $table->unsignedSmallInteger('pulse')->nullable();
            $table->string('arm', 8)->nullable();             // left|right
            $table->string('position', 12)->nullable();       // sitting|standing|lying
            $table->decimal('glucose_mg_dl', 5, 1)->nullable();
            $table->string('glucose_unit', 8)->nullable();    // mg_dl|mmol_l (as typed)
            $table->string('context', 16)->nullable();
            $table->string('method', 12)->nullable();         // glucometer|lab|sensor
            $table->string('note', 500)->nullable();
            $table->timestamps();

            $table->index(['user_id', 'type', 'measured_at']);
            $table->index(['user_id', 'measured_at']);
        });

        Schema::create('vital_plan_items', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('type', 8);
            $table->string('slot', 16);
            $table->unsignedTinyInteger('days')->default(127); // bit 0 = Saturday
            $table->string('remind_at', 5)->nullable();          // HH:MM
            $table->unsignedTinyInteger('sort_order')->default(0);
            $table->timestamps();

            $table->unique(['user_id', 'type', 'slot']);
        });

        $now = now();
        DB::table('message_contents')->insertOrIgnore([
            [
                'group' => 'vitals_alert',
                'item_key' => 'bp_crisis',
                'locale' => 'fa',
                'label' => 'vitals_alert / bp_crisis',
                'payload' => '{"level":"urgent","title":"فشار خونت در محدودهٔ بحرانی است","what_we_saw":"فشار {value} ثبت کردی؛ عدد بالاتر از ۱۸۰/۱۲۰ اورژانسی حساب می‌شود.","advice":"۵ دقیقه آرام بنشین و دوباره اندازه بگیر. اگر باز هم بالاتر از ۱۸۰/۱۲۰ بود، یا سردرد شدید، درد قفسهٔ سینه، تنگی نفس، تاری دید، ضعف یا بی‌حسی یک طرف بدن یا اشکال در حرف زدن داری، منتظر نمان.","actions":[{"key":"call","label":"تماس با اورژانس ۱۱۵","phone":"115"},{"key":"ack","label":"دوباره اندازه می‌گیرم"}],"contact":"ریتمی تشخیص پزشکی نمی‌دهد؛ در شرایط اورژانسی همین حالا با ۱۱۵ تماس بگیر."}',
                'is_active' => true,
                'is_approved' => true,
                'sort_order' => 0,
                'created_at' => $now,
                'updated_at' => $now,
            ],
            [
                'group' => 'vitals_alert',
                'item_key' => 'bp_crisis',
                'locale' => 'en',
                'label' => 'vitals_alert / bp_crisis',
                'payload' => '{"level":"urgent","title":"Your blood pressure is in the crisis range","what_we_saw":"You logged {value}; a reading above 180/120 counts as an emergency.","advice":"Sit quietly for 5 minutes and measure again. If it is still above 180/120, or you have a severe headache, chest pain, shortness of breath, blurred vision, weakness or numbness on one side or trouble speaking, do not wait.","actions":[{"key":"call","label":"Call emergency services (115)","phone":"115"},{"key":"ack","label":"I will measure again"}],"contact":"Ritme does not give a medical diagnosis; in an emergency call 115 now."}',
                'is_active' => true,
                'is_approved' => true,
                'sort_order' => 0,
                'created_at' => $now,
                'updated_at' => $now,
            ],
            [
                'group' => 'vitals_alert',
                'item_key' => 'glucose_low',
                'locale' => 'fa',
                'label' => 'vitals_alert / glucose_low',
                'payload' => '{"level":"urgent","title":"قند خونت خیلی پایین است","what_we_saw":"قند {value} ثبت کردی؛ کمتر از ۵۴ mg/dL (۳٫۰ mmol/L) افت قند شدید حساب می‌شود.","advice":"همین حالا ۱۵ گرم قند زودجذب بخور (مثلاً نصف لیوان آب‌میوه یا ۳ تا ۴ حبه قند) و ۱۵ دقیقه بعد دوباره اندازه بگیر. اگر گیج یا خواب‌آلودی، نمی‌توانی چیزی بخوری یا قند بالا نمی‌رود، از اطرافیان کمک بخواه.","actions":[{"key":"call","label":"تماس با اورژانس ۱۱۵","phone":"115"},{"key":"ack","label":"قند خوردم، دوباره اندازه می‌گیرم"}],"contact":"ریتمی تشخیص پزشکی نمی‌دهد؛ در شرایط اورژانسی همین حالا با ۱۱۵ تماس بگیر."}',
                'is_active' => true,
                'is_approved' => true,
                'sort_order' => 0,
                'created_at' => $now,
                'updated_at' => $now,
            ],
            [
                'group' => 'vitals_alert',
                'item_key' => 'glucose_low',
                'locale' => 'en',
                'label' => 'vitals_alert / glucose_low',
                'payload' => '{"level":"urgent","title":"Your blood sugar is very low","what_we_saw":"You logged {value}; below 54 mg/dL (3.0 mmol/L) counts as severe low blood sugar.","advice":"Take 15 g of fast-acting sugar now (for example half a glass of juice or 3 to 4 sugar cubes) and measure again after 15 minutes. If you feel confused or drowsy, cannot eat or it does not go up, ask someone near you for help.","actions":[{"key":"call","label":"Call emergency services (115)","phone":"115"},{"key":"ack","label":"I had sugar, measuring again"}],"contact":"Ritme does not give a medical diagnosis; in an emergency call 115 now."}',
                'is_active' => true,
                'is_approved' => true,
                'sort_order' => 0,
                'created_at' => $now,
                'updated_at' => $now,
            ],
        ]);
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        DB::table('message_contents')
            ->where('group', 'vitals_alert')->whereIn('item_key', ['bp_crisis', 'glucose_low'])
            ->whereIn('locale', ['fa', 'en'])
            ->whereIn('label', ['vitals_alert / bp_crisis', 'vitals_alert / glucose_low'])
            ->whereColumn('updated_at', 'created_at')
            ->delete();
        Schema::dropIfExists('vital_plan_items');
        Schema::dropIfExists('vital_readings');
    }
};

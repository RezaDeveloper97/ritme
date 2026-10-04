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
     * Twin of backend-go/db/migrations/00033_baby_logs.sql
     * (docs/go-migration/migrations.md): baby feeding / sleep sessions and
     * diapers per child, kick-count and contraction sessions per pregnant user
     * (bloom B-N5-03), used only by the Go internal/babylog and
     * internal/pregnancy/tools packages, plus the pregnancy_alert /
     * contractions_511 rule rows (fa + en, needs clinical review) so
     * `make schema-diff` row counts match. No model or routes here.
     * active_lock = 1 while a session runs, NULL once it ended: the unique
     * index makes "one running session" a database rule.
     */
    public function up(): void
    {
        Schema::create('baby_feeds', function (Blueprint $table) {
            $table->id();
            $table->foreignId('child_id')->constrained('children')->cascadeOnDelete();
            $table->string('type', 8);                          // breast|bottle|pump
            $table->dateTime('started_at');
            $table->dateTime('ended_at')->nullable();           // NULL = running
            $table->string('active_side', 8)->nullable();       // left|right (running segment)
            $table->dateTime('side_started_at')->nullable();
            $table->string('last_side', 8)->nullable();
            $table->unsignedInteger('left_seconds')->default(0);
            $table->unsignedInteger('right_seconds')->default(0);
            $table->unsignedInteger('duration_seconds')->nullable();
            $table->unsignedSmallInteger('amount_ml')->nullable();
            $table->string('note', 500)->nullable();
            $table->unsignedTinyInteger('active_lock')->nullable();
            $table->timestamps();

            $table->unique(['child_id', 'active_lock']);
            $table->index(['child_id', 'started_at']);
        });

        Schema::create('baby_sleeps', function (Blueprint $table) {
            $table->id();
            $table->foreignId('child_id')->constrained('children')->cascadeOnDelete();
            $table->dateTime('started_at');
            $table->dateTime('ended_at')->nullable();
            $table->string('note', 500)->nullable();
            $table->unsignedTinyInteger('active_lock')->nullable();
            $table->timestamps();

            $table->unique(['child_id', 'active_lock']);
            $table->index(['child_id', 'started_at']);
        });

        Schema::create('baby_diapers', function (Blueprint $table) {
            $table->id();
            $table->foreignId('child_id')->constrained('children')->cascadeOnDelete();
            $table->dateTime('changed_at');
            $table->string('kind', 8);                          // wet|dirty|both
            $table->string('note', 500)->nullable();
            $table->timestamps();

            $table->index(['child_id', 'changed_at']);
        });

        Schema::create('pregnancy_kick_sessions', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained('users')->cascadeOnDelete();
            $table->dateTime('started_at');
            $table->dateTime('ended_at')->nullable();
            $table->unsignedSmallInteger('kicks')->default(0);
            $table->dateTime('tenth_kick_at')->nullable();
            $table->dateTime('last_kick_at')->nullable();
            $table->unsignedTinyInteger('pregnancy_week')->nullable();
            $table->unsignedTinyInteger('active_lock')->nullable();
            $table->timestamps();

            $table->unique(['user_id', 'active_lock']);
            $table->index(['user_id', 'started_at']);
        });

        Schema::create('pregnancy_contraction_sessions', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained('users')->cascadeOnDelete();
            $table->dateTime('started_at');
            $table->dateTime('ended_at')->nullable();
            $table->dateTime('alert_at')->nullable();           // first time the run met 5-1-1
            $table->unsignedTinyInteger('active_lock')->nullable();
            $table->timestamps();

            $table->unique(['user_id', 'active_lock']);
            $table->index(['user_id', 'started_at']);
        });

        Schema::create('pregnancy_contractions', function (Blueprint $table) {
            $table->id();
            $table->foreignId('session_id')->constrained('pregnancy_contraction_sessions')->cascadeOnDelete();
            $table->foreignId('user_id')->constrained('users')->cascadeOnDelete();
            $table->dateTime('started_at');
            $table->dateTime('ended_at')->nullable();           // NULL = in progress
            $table->unsignedTinyInteger('active_lock')->nullable();
            $table->timestamps();

            $table->unique(['session_id', 'active_lock']);
            $table->index(['user_id', 'started_at']);
        });

        $now = now();
        DB::table('message_contents')->insertOrIgnore([
            [
                'group' => 'pregnancy_alert',
                'item_key' => 'contractions_511',
                'locale' => 'fa',
                'label' => 'pregnancy_alert / contractions_511',
                'payload' => '{"enabled":true,"level":"urgent","window_days":1,"params":{"interval_max_minutes":5,"duration_min_seconds":45,"run_minutes":60},"title":"انقباض‌ها به الگوی ۵-۱-۱ رسیده","what_we_saw":"در {minutes} دقیقهٔ گذشته {count} انقباض ثبت کردی؛ فاصلهٔ میانگین {interval} و مدت میانگین {duration}","how_sure":"این محاسبه از زمان‌هایی است که خودت ثبت کردی؛ فقط معاینه می‌تواند شروع زایمان را تأیید کند","advice":"وقت تماس با زایشگاه یا مامای خودت است؛ وسایل بیمارستان را آماده کن. اگر کیسهٔ آب پاره شد، خون‌ریزی داری یا حرکات جنین کم شده، منتظر نمان.","actions":[{"key":"call","label":"تماس با زایشگاه"},{"key":"ack","label":"دیدم، ممنون"}],"contact":"اگر خون‌ریزی شدید، درد مداوم یا کم‌شدن حرکات جنین داری، همین حالا با اورژانس (۱۱۵) تماس بگیر."}',
                'is_active' => true,
                'is_approved' => true,
                'sort_order' => 0,
                'created_at' => $now,
                'updated_at' => $now,
            ],
            [
                'group' => 'pregnancy_alert',
                'item_key' => 'contractions_511',
                'locale' => 'en',
                'label' => 'pregnancy_alert / contractions_511',
                'payload' => '{"enabled":true,"level":"urgent","window_days":1,"params":{"interval_max_minutes":5,"duration_min_seconds":45,"run_minutes":60},"title":"Your contractions reached the 5-1-1 pattern","what_we_saw":"You timed {count} contractions in the last {minutes} minutes, on average {interval} apart and {duration} long","how_sure":"This is worked out from the times you logged; only an exam can confirm that labour has started","advice":"It is time to call your maternity unit or midwife and get your hospital bag ready. If your waters break, you are bleeding or the baby moves less, do not wait.","actions":[{"key":"call","label":"Call the maternity unit"},{"key":"ack","label":"Got it, thanks"}],"contact":"If you have heavy bleeding, constant pain or the baby moves less, call emergency services (115) now."}',
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
            ->where('group', 'pregnancy_alert')->where('item_key', 'contractions_511')
            ->whereIn('locale', ['fa', 'en'])
            ->where('label', 'pregnancy_alert / contractions_511')
            ->whereColumn('updated_at', 'created_at')
            ->delete();
        Schema::dropIfExists('pregnancy_contractions');
        Schema::dropIfExists('pregnancy_contraction_sessions');
        Schema::dropIfExists('pregnancy_kick_sessions');
        Schema::dropIfExists('baby_diapers');
        Schema::dropIfExists('baby_sleeps');
        Schema::dropIfExists('baby_feeds');
    }
};

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
     * Twin of backend-go/db/migrations/00012_privacy_support.sql
     * (docs/go-migration/migrations.md): consents and support reports written
     * by the Go-only /api/v1/profile/consents and /api/v1/support/reports
     * endpoints (B-N1-12), plus the info_sections rows the new privacy /
     * legal / support screens read by key. Schema + seed rows only — no
     * models or routes here. insertOrIgnore never overwrites an existing row.
     */
    public function up(): void
    {
        Schema::create('user_consents', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('consent', 64);
            $table->boolean('granted')->default(false);
            $table->timestamp('granted_at')->nullable();
            $table->timestamp('revoked_at')->nullable();
            $table->timestamps();

            $table->unique(['user_id', 'consent']);
        });

        Schema::create('support_reports', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->text('message');
            $table->string('screenshot_path')->nullable();
            $table->string('app_version', 32)->nullable();
            $table->string('user_agent')->nullable();
            $table->string('status', 16)->default('open');
            $table->timestamps();

            $table->index(['status', 'created_at']);
        });

        $now = now();
        $json = fn (array $v) => json_encode($v, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $row = fn (string $group, string $key, array $heading, array $body, ?array $label, ?string $url, int $sort) => [
            'group' => $group,
            'key' => $key,
            'heading' => $json($heading),
            'body' => $json($body),
            'link_label' => $label === null ? null : $json($label),
            'link_url' => $url,
            'is_active' => 1,
            'sort_order' => $sort,
            'created_at' => $now,
            'updated_at' => $now,
        ];

        DB::table('info_sections')->insertOrIgnore([
            $row('privacy', 'summary', ['fa' => 'خلاصه در ۳ خط', 'en' => 'In 3 lines'], [
                'fa' => "داده‌های سلامتت را نمی‌فروشیم و برای تبلیغات استفاده نمی‌کنیم\nاشتراک با پزشک، همراه یا هوش مصنوعی فقط با رضایت صریح توست\nهر زمان می‌توانی داده‌هایت را دریافت یا حذف کنی",
                'en' => "We never sell your health data or use it for advertising\nSharing with a doctor, a companion or AI happens only with your explicit consent\nYou can download or delete your data at any time",
            ], null, null, 0),
            $row('terms', 'summary', ['fa' => 'خلاصه در ۳ خط', 'en' => 'In 3 lines'], [
                'fa' => "ریتمی ابزار آگاهی سلامت است و جایگزین تشخیص یا درمان پزشکی نیست\nپیش‌بینی‌های ریتمی روش پیشگیری از بارداری نیستند\nکد ورود را به کسی نده؛ امنیت حساب و گوشی با خودت است",
                'en' => "Ritme is a health-awareness tool, not a substitute for medical diagnosis or treatment\nRitme's predictions are not a method of contraception\nNever share your sign-in code; keeping your account and phone secure is up to you",
            ], null, null, 0),
            $row('about', 'disclaimer', ['fa' => 'یادآوری مهم', 'en' => 'Please note'], [
                'fa' => 'ریتمی ابزار آگاهی سلامت است و جایگزین تشخیص یا درمان پزشکی نیست. پیش‌بینی‌ها روش پیشگیری از بارداری نیستند.',
                'en' => 'Ritme is a health-awareness tool, not a substitute for medical diagnosis or treatment. Predictions are not a method of contraception.',
            ], null, null, 1000),
            $row('support', 'email', ['fa' => 'ایمیل پشتیبانی', 'en' => 'Support email'], [
                'fa' => 'برایمان بنویس؛ هرچه زودتر جواب می‌دهیم',
                'en' => "Write to us and we'll reply as soon as we can",
            ], ['fa' => 'support@ritmesalamat.com', 'en' => 'support@ritmesalamat.com'], 'mailto:support@ritmesalamat.com', 10),
        ]);
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        DB::table('info_sections')
            ->where(fn ($q) => $q->where(fn ($q) => $q->where('group', 'privacy')->where('key', 'summary'))
                ->orWhere(fn ($q) => $q->where('group', 'terms')->where('key', 'summary'))
                ->orWhere(fn ($q) => $q->where('group', 'about')->where('key', 'disclaimer'))
                ->orWhere(fn ($q) => $q->where('group', 'support')->where('key', 'email')))
            ->whereColumn('updated_at', 'created_at')
            ->delete();
        Schema::dropIfExists('support_reports');
        Schema::dropIfExists('user_consents');
    }
};

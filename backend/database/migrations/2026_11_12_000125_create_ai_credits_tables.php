<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

// Twin of backend-go/db/migrations/00125_ai_credits.sql: AI credits (plan / trial credits that end with the period,
// extra credits that never expire, the model catalogue, the feature slots and the credit packs). Schema + the same
// seed rows; the rules live in the Go migration's header.
return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasColumn('plus_plans', 'ai_credits')) {
            Schema::table('plus_plans', function (Blueprint $table) {
                $table->unsignedInteger('ai_credits')->default(0)->after('monthly_display_rials');
            });
        }

        if (! Schema::hasTable('ai_models')) {
            Schema::create('ai_models', function (Blueprint $table) {
                $table->id();
                $table->string('code', 64)->unique();
                $table->string('provider', 16);
                $table->string('upstream_model', 128);
                $table->string('title', 128);
                $table->json('capabilities');
                $table->unsignedInteger('input_price_milli')->default(0);
                $table->unsignedInteger('output_price_milli')->default(0);
                $table->unsignedInteger('audio_price_milli')->default(0);
                $table->boolean('is_active')->default(true);
                $table->timestamps();
                $table->unique(['provider', 'upstream_model']);
            });
        }

        if (! Schema::hasTable('ai_features')) {
            Schema::create('ai_features', function (Blueprint $table) {
                $table->id();
                $table->string('key', 64)->unique();
                $table->json('title');
                $table->foreignId('model_id')->nullable()->constrained('ai_models')->nullOnDelete();
                $table->foreignId('fallback_model_id')->nullable()->constrained('ai_models')->nullOnDelete();
                $table->unsignedInteger('credit_cost')->default(1);
                $table->boolean('is_active')->default(true);
                $table->unsignedInteger('sort_order')->default(0);
                $table->timestamps();
            });
        }

        if (! Schema::hasTable('ai_credit_packs')) {
            Schema::create('ai_credit_packs', function (Blueprint $table) {
                $table->id();
                $table->string('code', 64)->unique();
                $table->json('title');
                $table->json('badge')->nullable();
                $table->unsignedInteger('credits');
                $table->unsignedBigInteger('price_rials');
                $table->boolean('is_highlighted')->default(false);
                $table->boolean('is_active')->default(true);
                $table->unsignedInteger('sort_order')->default(0);
                $table->timestamps();
            });
        }

        if (! Schema::hasColumn('plus_invoices', 'kind')) {
            Schema::table('plus_invoices', function (Blueprint $table) {
                $table->string('kind', 16)->default('plan')->after('plan_id');
                $table->foreignId('credit_pack_id')->nullable()->after('kind')->constrained('ai_credit_packs')->nullOnDelete();
                $table->unsignedInteger('credits')->nullable()->after('credit_pack_id');
            });
        }

        if (! Schema::hasTable('ai_credit_grants')) {
            Schema::create('ai_credit_grants', function (Blueprint $table) {
                $table->id();
                $table->foreignId('user_id')->constrained()->cascadeOnDelete();
                $table->string('kind', 16);
                $table->foreignId('subscription_id')->nullable()->unique()->constrained('plus_subscriptions')->cascadeOnDelete();
                $table->foreignId('trial_id')->nullable()->unique()->constrained('plus_trials')->cascadeOnDelete();
                $table->foreignId('invoice_id')->nullable()->unique()->constrained('plus_invoices')->nullOnDelete();
                $table->unsignedInteger('amount');
                $table->unsignedInteger('remaining');
                $table->string('note', 255)->nullable();
                $table->timestamps();
                $table->index(['user_id', 'kind']);
            });
        }

        if (! Schema::hasTable('ai_credit_spends')) {
            Schema::create('ai_credit_spends', function (Blueprint $table) {
                $table->id();
                $table->foreignId('user_id')->constrained()->cascadeOnDelete();
                $table->string('feature', 64);
                $table->unsignedInteger('credits');
                $table->unsignedInteger('plan_credits')->default(0);
                $table->unsignedInteger('extra_credits')->default(0);
                $table->json('allocations');
                $table->string('status', 16)->default('spent');
                $table->timestamps();
                $table->index(['user_id', 'created_at']);
                $table->index(['feature', 'created_at']);
            });
        }

        $now = DB::raw("CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')");
        foreach (['plus_1m' => 100, 'plus_3m' => 300, 'plus_6m' => 600, 'mtest' => 10] as $code => $credits) {
            DB::table('plus_plans')->where('code', $code)->where('ai_credits', 0)->update(['ai_credits' => $credits]);
        }
        DB::table('plus_settings')->insertOrIgnore([
            'key' => 'trial_ai_credits', 'value' => '20', 'created_at' => $now, 'updated_at' => $now,
        ]);
        DB::table('ai_models')->insertOrIgnore([
            'code' => 'gemini-flash', 'provider' => 'gemini', 'upstream_model' => 'gemini-flash-latest',
            'title' => 'Gemini Flash', 'capabilities' => '["chat","vision","audio","json"]',
            'input_price_milli' => 300, 'output_price_milli' => 2500, 'audio_price_milli' => 1000, 'is_active' => 1,
            'created_at' => $now, 'updated_at' => $now,
        ]);
        $modelId = DB::table('ai_models')->where('code', 'gemini-flash')->value('id');
        $features = [
            ['assistant', '{"fa":"دستیار سلامت","en":"Health assistant"}', 1, 1],
            ['voice_log', '{"fa":"ثبت صوتی","en":"Voice logging"}', 1, 2],
            ['lab_analysis', '{"fa":"تحلیل آزمایش","en":"Lab analysis"}', 5, 3],
            ['doc_extract', '{"fa":"خواندن مدارک","en":"Document reading"}', 3, 4],
        ];
        foreach ($features as [$key, $title, $cost, $sort]) {
            DB::table('ai_features')->insertOrIgnore([
                'key' => $key, 'title' => $title, 'model_id' => $modelId, 'fallback_model_id' => null,
                'credit_cost' => $cost, 'is_active' => 1, 'sort_order' => $sort, 'created_at' => $now, 'updated_at' => $now,
            ]);
        }
        $packs = [
            ['credits_50', '{"fa":"۵۰ اعتبار","en":"50 credits"}', null, 50, 490000, 0, 1],
            ['credits_150', '{"fa":"۱۵۰ اعتبار","en":"150 credits"}', '{"fa":"محبوب‌ترین","en":"Most popular"}', 150, 1290000, 1, 2],
            ['credits_400', '{"fa":"۴۰۰ اعتبار","en":"400 credits"}', null, 400, 2990000, 0, 3],
        ];
        foreach ($packs as [$code, $title, $badge, $credits, $price, $highlighted, $sort]) {
            DB::table('ai_credit_packs')->insertOrIgnore([
                'code' => $code, 'title' => $title, 'badge' => $badge, 'credits' => $credits, 'price_rials' => $price,
                'is_highlighted' => $highlighted, 'is_active' => 1, 'sort_order' => $sort,
                'created_at' => $now, 'updated_at' => $now,
            ]);
        }
    }

    public function down(): void
    {
        Schema::dropIfExists('ai_credit_spends');
        Schema::dropIfExists('ai_credit_grants');
        if (Schema::hasColumn('plus_invoices', 'kind')) {
            Schema::table('plus_invoices', function (Blueprint $table) {
                $table->dropConstrainedForeignId('credit_pack_id');
                $table->dropColumn(['credits', 'kind']);
            });
        }
        Schema::dropIfExists('ai_credit_packs');
        Schema::dropIfExists('ai_features');
        Schema::dropIfExists('ai_models');
        DB::table('plus_settings')->where('key', 'trial_ai_credits')->delete();
        if (Schema::hasColumn('plus_plans', 'ai_credits')) {
            Schema::table('plus_plans', fn (Blueprint $table) => $table->dropColumn('ai_credits'));
        }
    }
};

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
     * Twin of backend-go/db/migrations/00015_plus_subscriptions.sql
     * (docs/go-migration/migrations.md): the Ritme Plus subscription domain
     * (plans, trials, discount codes, invoices, receipts, subscriptions,
     * usage counters) served by the Go-only /api/v1/plus endpoints
     * (B-N2-04) plus the same three seed plans, so `make schema-diff` row
     * counts match. Amounts are integer rials. Schema only — no models or
     * routes here. insertOrIgnore never overwrites an admin edit.
     */
    public function up(): void
    {
        Schema::create('plus_plans', function (Blueprint $table) {
            $table->id();
            $table->string('code', 64)->unique();
            $table->json('title');
            $table->json('badge')->nullable();
            $table->unsignedSmallInteger('duration_months');
            $table->unsignedBigInteger('price_rials');
            $table->unsignedBigInteger('monthly_display_rials')->nullable();
            $table->boolean('is_highlighted')->default(false);
            $table->boolean('is_active')->default(true);
            $table->unsignedInteger('sort_order')->default(0);
            $table->timestamps();
        });

        Schema::create('plus_trials', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained()->cascadeOnDelete();
            $table->dateTime('started_at');
            $table->dateTime('ends_at');
            $table->timestamps();
        });

        Schema::create('plus_discount_codes', function (Blueprint $table) {
            $table->id();
            $table->string('code', 64)->unique();
            $table->string('kind', 16);                          // percent | amount
            $table->unsignedBigInteger('value');                 // percent 1–100 or rials
            $table->unsignedInteger('max_redemptions')->nullable();
            $table->unsignedInteger('per_user_limit')->nullable()->default(1);
            $table->json('plan_ids')->nullable();
            $table->dateTime('starts_at')->nullable();
            $table->dateTime('expires_at')->nullable();
            $table->boolean('is_active')->default(true);
            $table->timestamps();
        });

        Schema::create('plus_invoices', function (Blueprint $table) {
            $table->id();
            $table->string('reference', 32)->unique();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('plan_id')->nullable()->constrained('plus_plans')->nullOnDelete();
            $table->unsignedSmallInteger('duration_months');
            $table->string('status', 16)->default('pending');   // pending | paid | failed | expired | refunded
            $table->string('currency', 3)->default('IRR');
            $table->unsignedBigInteger('subtotal_rials');
            $table->unsignedBigInteger('discount_rials')->default(0);
            $table->unsignedInteger('vat_rate_bps');
            $table->unsignedBigInteger('vat_rials');
            $table->unsignedBigInteger('total_rials');
            $table->foreignId('discount_code_id')->nullable()->constrained('plus_discount_codes')->nullOnDelete();
            $table->string('discount_code', 64)->nullable();
            $table->string('gateway', 32)->nullable();
            $table->string('authority', 191)->nullable()->unique();
            $table->dateTime('expires_at');
            $table->dateTime('paid_at')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'status']);
            $table->index(['discount_code_id', 'status']);
        });

        Schema::create('plus_receipts', function (Blueprint $table) {
            $table->id();
            $table->foreignId('invoice_id')->unique()->constrained('plus_invoices')->cascadeOnDelete();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('gateway', 32);
            $table->string('ref_id', 191);
            $table->string('card_pan', 32)->nullable();
            $table->unsignedBigInteger('amount_rials');
            $table->dateTime('paid_at');
            $table->timestamps();

            $table->unique(['gateway', 'ref_id']);
        });

        Schema::create('plus_subscriptions', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('plan_id')->nullable()->constrained('plus_plans')->nullOnDelete();
            $table->foreignId('invoice_id')->nullable()->unique()->constrained('plus_invoices')->nullOnDelete();
            $table->string('status', 16)->default('active');    // active | canceled | refunded
            $table->string('source', 16)->default('purchase');  // purchase | admin
            $table->dateTime('starts_at');
            $table->dateTime('ends_at');
            $table->boolean('auto_renew')->default(true);
            $table->dateTime('canceled_at')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'ends_at']);
        });

        Schema::create('plus_usage_counters', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('feature', 64);
            $table->date('period_start');
            $table->unsignedInteger('used')->default(0);
            $table->timestamps();

            $table->unique(['user_id', 'feature', 'period_start']);
        });

        $now = now();
        $json = fn (?array $v) => $v === null ? null : json_encode($v, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $plan = fn (string $code, array $title, ?array $badge, int $months, int $price, bool $highlighted, int $sort) => [
            'code' => $code,
            'title' => $json($title),
            'badge' => $json($badge),
            'duration_months' => $months,
            'price_rials' => $price,
            'monthly_display_rials' => null,
            'is_highlighted' => $highlighted ? 1 : 0,
            'is_active' => 1,
            'sort_order' => $sort,
            'created_at' => $now,
            'updated_at' => $now,
        ];

        DB::table('plus_plans')->insertOrIgnore([
            $plan('plus_1m', ['fa' => '۱ ماهه', 'en' => '1 month'], null, 1, 990000, false, 1),
            $plan('plus_3m', ['fa' => '۳ ماهه', 'en' => '3 months'], ['fa' => 'محبوب‌ترین', 'en' => 'Most popular'], 3, 2370000, true, 2),
            $plan('plus_6m', ['fa' => '۶ ماهه', 'en' => '6 months'], null, 6, 3900000, false, 3),
        ]);
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('plus_usage_counters');
        Schema::dropIfExists('plus_subscriptions');
        Schema::dropIfExists('plus_receipts');
        Schema::dropIfExists('plus_invoices');
        Schema::dropIfExists('plus_discount_codes');
        Schema::dropIfExists('plus_trials');
        Schema::dropIfExists('plus_plans');
    }
};

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
     * Twin of backend-go/db/migrations/00050_telemed_bookings.sql
     * (docs/go-migration/migrations.md): visit booking with payment,
     * cancellation rules and scoped data-share consent (bloom B-N7-03), used
     * only by the Go internal/telemed package, plus the telemed_visit_reasons
     * catalog rows (fa + en) so `make schema-diff` row counts match. No model
     * or routes here.
     */
    public function up(): void
    {
        Schema::create('telemed_bookings', function (Blueprint $table) {
            $table->id();
            $table->string('reference', 24)->unique();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('doctor_id')->constrained('telemed_doctors')->cascadeOnDelete();
            $table->string('mode', 12);                          // video|phone|in_person
            $table->unsignedSmallInteger('duration_minutes');
            $table->dateTime('starts_at');
            $table->dateTime('ends_at');
            $table->string('status', 12);                        // held|confirmed|completed|cancelled|expired|failed
            $table->string('slot_key', 48)->nullable()->unique(); // "doctor:start" while held/confirmed
            $table->dateTime('hold_expires_at')->nullable();
            $table->string('for_whom', 8);                       // self|child|other
            $table->foreignId('child_id')->nullable()->constrained('children')->nullOnDelete();
            $table->string('patient_name', 64)->nullable();
            $table->string('reason', 64)->nullable();            // catalog telemed_visit_reasons code
            $table->string('note', 1000)->nullable();
            $table->unsignedBigInteger('price_rials');
            $table->unsignedBigInteger('discount_rials')->default(0);
            $table->string('discount_source', 16)->nullable();
            $table->unsignedBigInteger('total_rials');
            $table->string('payment_status', 16)->default('none');
            $table->string('gateway', 32)->nullable();
            $table->string('authority', 191)->nullable();
            $table->string('ref_id', 191)->nullable();
            $table->string('card_pan', 32)->nullable();
            $table->dateTime('paid_at')->nullable();
            $table->string('refund_id', 191)->nullable();
            $table->unsignedBigInteger('refunded_rials')->default(0);
            $table->dateTime('refunded_at')->nullable();
            $table->foreignId('appointment_id')->nullable()->constrained('reminders')->nullOnDelete();
            $table->unsignedTinyInteger('reschedules')->default(0);
            $table->dateTime('cancelled_at')->nullable();
            $table->dateTime('completed_at')->nullable();
            $table->timestamps();

            $table->unique(['gateway', 'ref_id']);
            $table->index(['user_id', 'starts_at']);
            $table->index(['doctor_id', 'starts_at']);
            $table->index(['status', 'hold_expires_at']);
        });

        Schema::create('telemed_booking_consents', function (Blueprint $table) {
            $table->id();
            $table->foreignId('booking_id')->constrained('telemed_bookings')->cascadeOnDelete();
            $table->string('scope', 32);                         // cycle_summary|bbt_lh|assistant_summaries
            $table->dateTime('granted_at');
            $table->dateTime('revoked_at')->nullable();
            $table->timestamps();

            $table->unique(['booking_id', 'scope']);
        });

        $now = now();
        $row = fn (string $code, int $sort, array $title) => [
            'group' => 'telemed_visit_reasons',
            'code' => $code,
            'sort_order' => $sort,
            'is_active' => 1,
            'audiences' => null,
            'title' => json_encode($title, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES),
            'body' => null,
            'meta' => null,
            'needs_review' => 1,
            'created_at' => $now,
            'updated_at' => $now,
        ];
        DB::table('catalog_items')->insertOrIgnore([
            $row('ttc', 1, ['fa' => 'اقدام به بارداری', 'en' => 'Trying to conceive']),
            $row('irregular_period', 2, ['fa' => 'پریود نامنظم', 'en' => 'Irregular periods']),
            $row('pelvic_pain', 3, ['fa' => 'درد لگن', 'en' => 'Pelvic pain']),
            $row('lab_followup', 4, ['fa' => 'پیگیری آزمایش', 'en' => 'Lab result follow-up']),
            $row('other', 5, ['fa' => 'سایر', 'en' => 'Other']),
        ]);
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        DB::table('catalog_items')->where('group', 'telemed_visit_reasons')
            ->whereIn('code', ['ttc', 'irregular_period', 'pelvic_pain', 'lab_followup', 'other'])
            ->whereColumn('updated_at', 'created_at')
            ->delete();
        Schema::dropIfExists('telemed_booking_consents');
        Schema::dropIfExists('telemed_bookings');
    }
};

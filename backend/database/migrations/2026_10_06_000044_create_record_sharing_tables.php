<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00044_record_sharing.sql
     * (docs/go-migration/migrations.md): record sharing (canvas-build
     * CB-REC-03) — the 24h doctor code kind of health_share_links, the share
     * access log and the emergency card. Used only by the Go
     * internal/sharelinks and internal/emergency packages. No model or routes.
     */
    public function up(): void
    {
        Schema::table('health_share_links', function (Blueprint $table) {
            $table->string('kind', 16)->default('report')->after('user_id');   // report | summary
            $table->string('label', 60)->nullable()->after('kind');             // the owner's own note
            $table->char('code_hash', 64)->nullable()->unique()->after('token_hash'); // HMAC of the short code
            $table->text('code_payload')->nullable()->after('code_hash');       // token sealed under the code

            $table->index(['user_id', 'kind']);
        });

        Schema::create('health_share_link_views', function (Blueprint $table) {
            $table->id();
            $table->foreignId('share_link_id')->constrained('health_share_links')->cascadeOnDelete();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('via', 8);       // link | code
            $table->string('device', 16);   // mobile | tablet | desktop | unknown
            $table->string('browser', 16);  // coarse family only
            $table->timestamp('viewed_at')->nullable();

            $table->index(['user_id', 'viewed_at']);
            $table->index(['share_link_id', 'viewed_at']);
        });

        Schema::create('emergency_cards', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained()->cascadeOnDelete();
            $table->boolean('show_on_lock_screen')->default(false);
            $table->boolean('show_pregnancy')->default(false);
            $table->string('contact_name', 60)->nullable();
            $table->string('contact_relation', 30)->nullable();
            $table->string('contact_phone', 20)->nullable();
            $table->string('insurance_label', 60)->nullable();
            $table->char('insurance_last4', 4)->nullable();
            $table->char('public_token_hash', 64)->nullable()->unique();
            $table->timestamp('public_enabled_at')->nullable();
            $table->unsignedInteger('public_view_count')->default(0);
            $table->timestamp('public_last_viewed_at')->nullable();
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('emergency_cards');
        Schema::dropIfExists('health_share_link_views');
        Schema::table('health_share_links', function (Blueprint $table) {
            $table->dropIndex(['user_id', 'kind']);
            $table->dropUnique(['code_hash']);
            $table->dropColumn(['code_payload', 'code_hash', 'label', 'kind']);
        });
    }
};

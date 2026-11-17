<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

// Twin of backend-go/db/migrations/00127_voice_stt_default_model.sql: the default speech-to-text model of voice
// logging (GPT-4o Transcribe on the ArvanCloud AI gateway) and its `voice_stt` slot. Data only, the same rows; the
// reasons live in the Go migration's header. The Laravel API never reads these tables.
return new class extends Migration
{
    public function up(): void
    {
        $now = now();
        DB::table('ai_models')->insertOrIgnore([
            'code' => 'gpt-4o-transcribe', 'provider' => 'arvan', 'upstream_model' => 'gpt-4o-transcribe',
            'title' => 'GPT-4o Transcribe', 'capabilities' => '["audio"]',
            'input_price_milli' => 2500, 'output_price_milli' => 10000, 'audio_price_milli' => 6000, 'is_active' => 1,
            'created_at' => $now, 'updated_at' => $now,
        ]);
        $modelId = DB::table('ai_models')->where('code', 'gpt-4o-transcribe')->value('id');
        $fallbackId = DB::table('ai_models')->where('code', 'gemini-flash')->value('id');
        DB::table('ai_features')->insertOrIgnore([
            'key' => 'voice_stt', 'title' => '{"fa":"تبدیل صدا به متن","en":"Speech to text"}',
            'model_id' => $modelId, 'fallback_model_id' => $fallbackId,
            'credit_cost' => 1, 'is_active' => 1, 'sort_order' => 5, 'created_at' => $now, 'updated_at' => $now,
        ]);
        $slot = DB::table('ai_features')->where('key', 'voice_stt')->whereNull('model_id')->first();
        if ($slot) {
            DB::table('ai_features')->where('id', $slot->id)->update([
                'model_id' => $modelId, 'fallback_model_id' => $slot->fallback_model_id ?? $fallbackId,
            ]);
        }
    }

    public function down(): void
    {
        $modelId = DB::table('ai_models')->where('code', 'gpt-4o-transcribe')->value('id');
        if ($modelId) {
            DB::table('ai_features')->where('key', 'voice_stt')->where('model_id', $modelId)->update(['model_id' => null]);
            DB::table('ai_models')->where('id', $modelId)->delete();
        }
    }
};

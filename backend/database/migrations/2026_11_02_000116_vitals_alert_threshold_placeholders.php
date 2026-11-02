<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    /**
     * The seeded threshold numbers of the vitals_alert copy and their placeholders, per item_key and locale.
     *
     * @var array<string, array<string, array{0: string, 1: string}>>
     */
    private array $swaps = [
        'bp_crisis' => [
            'fa' => ['۱۸۰/۱۲۰', '{systolic}/{diastolic}'],
            'en' => ['180/120', '{systolic}/{diastolic}'],
        ],
        'glucose_low' => [
            'fa' => ['۵۴ mg/dL (۳٫۰ mmol/L)', '{mgdl} mg/dL ({mmol_l} mmol/L)'],
            'en' => ['54 mg/dL (3.0 mmol/L)', '{mgdl} mg/dL ({mmol_l} mmol/L)'],
        ],
    ];

    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00116_vitals_alert_threshold_placeholders.sql (docs/go-migration/migrations.md,
     * T-M10-19b, D-170): the vitals_alert copy rows seeded by 2026_10_06_000035 spell the thresholds; they take the
     * {systolic}/{diastolic}/{mgdl}/{mmol_l} placeholders the Go API fills from the clinical config. Only rows still
     * holding the seeded numbers change. Data only, used only by the Go backend.
     */
    public function up(): void
    {
        $this->swap(false);
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        $this->swap(true);
    }

    private function swap(bool $reverse): void
    {
        foreach ($this->swaps as $item => $locales) {
            foreach ($locales as $locale => [$seed, $placeholder]) {
                [$from, $to] = $reverse ? [$placeholder, $seed] : [$seed, $placeholder];
                DB::table('message_contents')
                    ->where('group', 'vitals_alert')->where('item_key', $item)->where('locale', $locale)
                    ->update(['payload' => DB::raw('REPLACE(`payload`, '.DB::getPdo()->quote($from).', '.DB::getPdo()->quote($to).')')]);
            }
        }
    }
};

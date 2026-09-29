<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    /**
     * Data-only twin of step 3 of backend-go/db/migrations/00008_pregnancy_v2_copy.sql
     * (T-M7-20): the admin-editable `pregnancy_setup/calendar_note` rows, so
     * `make schema-diff` row counts match. Steps 1 and 2 of 00008 are guarded
     * updates with no row-count effect and are left to the cutover (T-M2-27).
     * insertOrIgnore never overwrites an admin-created row.
     */
    public function up(): void
    {
        $now = now();
        $row = fn (string $locale, array $payload) => [
            'group' => 'pregnancy_setup',
            'item_key' => 'calendar_note',
            'locale' => $locale,
            'label' => 'pregnancy_setup / calendar_note',
            'payload' => json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES),
            'is_active' => 1,
            'is_approved' => 1,
            'sort_order' => 0,
            'created_at' => $now,
            'updated_at' => $now,
        ];

        DB::table('message_contents')->insertOrIgnore([
            $row('fa', [
                'plan_note' => 'زمان‌ها بر اساس برنامهٔ رایج مراقبت‌های بارداری‌اند و ممکنه پزشکت برنامهٔ متفاوتی بده.',
                'basis_lmp' => 'تاریخ‌ها بر اساس اولین روز آخرین قاعدگی محاسبه شده‌اند.',
                'basis_ultrasound' => 'تاریخ‌ها بر اساس سونوگرافی محاسبه شده‌اند.',
                'basis_manual' => 'تاریخ‌ها بر اساس سن بارداری واردشده محاسبه شده‌اند.',
            ]),
            $row('en', [
                'plan_note' => 'Timings follow the usual pregnancy care schedule; your doctor may give you a different plan.',
                'basis_lmp' => 'Dates are based on the first day of your last period.',
                'basis_ultrasound' => 'Dates are based on your ultrasound.',
                'basis_manual' => 'Dates are based on the pregnancy age you entered.',
            ]),
        ]);
    }

    public function down(): void
    {
        DB::table('message_contents')
            ->where('group', 'pregnancy_setup')
            ->where('item_key', 'calendar_note')
            ->whereColumn('updated_at', 'created_at')
            ->delete();
    }
};

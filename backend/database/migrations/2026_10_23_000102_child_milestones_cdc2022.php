<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00102_child_milestones_cdc2022.sql (docs/go-migration/migrations.md): the CDC
     * 2022 15- and 30-month milestone bands (catalog_items child_milestones, child_milestone_activities,
     * child_milestone_notes; needs clinical review) and the 3-month band marked as Ritme's own (meta.band_source =
     * ritme, DECISIONS.md H-15), so `make schema-diff` row counts match. Data only; read by the Go internal/children
     * package (T-M10-22, D-173). insertOrIgnore never overwrites an admin edit.
     */
    private const MILESTONES = ['m15_shows_object', 'm15_tries_words', 'm15_points_for_help', 'm15_stacks_two', 'm15_takes_steps', 'm30_look_at_me', 'm30_says_50_words', 'm30_action_phrase', 'm30_pretend_play', 'm30_jumps_both_feet'];

    private const ACTIVITIES = ['m15_build_knock', 'm30_jump_line'];

    private const NOTES = ['m15', 'm30'];

    public function up(): void
    {
        $now = now();
        $json = fn (?array $v) => $v === null ? null : json_encode($v, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $row = fn (string $group, string $code, int $sort, array $title, ?array $body, array $meta) => [
            'group' => $group,
            'code' => $code,
            'sort_order' => $sort,
            'is_active' => 1,
            'audiences' => null,
            'title' => $json($title),
            'body' => $json($body),
            'meta' => $json($meta),
            'needs_review' => 1,
            'created_at' => $now,
            'updated_at' => $now,
        ];

        DB::table('catalog_items')->insertOrIgnore([
            $row('child_milestones', 'm15_shows_object', 58, ['fa' => 'چیزی را که دوست دارد به تو نشان می‌دهد', 'en' => 'Shows you an object they like'], null, ['age_months' => 15, 'domain' => 'social', 'band_source' => 'cdc_2022']),
            $row('child_milestones', 'm15_tries_words', 59, ['fa' => 'غیر از ماما و بابا، یکی دو کلمه دیگر هم امتحان می‌کند', 'en' => 'Tries to say one or two words besides mama or dada'], null, ['age_months' => 15, 'domain' => 'language', 'band_source' => 'cdc_2022']),
            $row('child_milestones', 'm15_points_for_help', 60, ['fa' => 'برای خواستن چیزی یا کمک گرفتن اشاره می‌کند', 'en' => 'Points to ask for something or to get help'], null, ['age_months' => 15, 'domain' => 'language', 'band_source' => 'cdc_2022']),
            $row('child_milestones', 'm15_stacks_two', 61, ['fa' => 'دست‌کم دو چیز کوچک، مثل مکعب، را روی هم می‌گذارد', 'en' => 'Stacks at least two small objects, like blocks'], null, ['age_months' => 15, 'domain' => 'cognitive', 'band_source' => 'cdc_2022']),
            $row('child_milestones', 'm15_takes_steps', 62, ['fa' => 'خودش چند قدم برمی‌دارد', 'en' => 'Takes a few steps on their own'], null, ['age_months' => 15, 'domain' => 'motor', 'band_source' => 'cdc_2022']),
            $row('child_milestones', 'm30_look_at_me', 63, ['fa' => 'با گفتن «ببین!» کاری را که بلد است نشانت می‌دهد', 'en' => 'Shows you what they can do by saying “Look at me!”'], null, ['age_months' => 30, 'domain' => 'social', 'band_source' => 'cdc_2022']),
            $row('child_milestones', 'm30_says_50_words', 64, ['fa' => 'حدود ۵۰ کلمه می‌گوید', 'en' => 'Says about 50 words'], null, ['age_months' => 30, 'domain' => 'language', 'band_source' => 'cdc_2022']),
            $row('child_milestones', 'm30_action_phrase', 65, ['fa' => 'دو کلمه یا بیشتر را همراه یک فعل می‌گوید، مثل «هاپو دوید»', 'en' => 'Says two or more words together, with one action word, like “doggie run”'], null, ['age_months' => 30, 'domain' => 'language', 'band_source' => 'cdc_2022']),
            $row('child_milestones', 'm30_pretend_play', 66, ['fa' => 'با چیزها بازی وانمودی می‌کند، مثلاً به عروسک مکعب می‌دهد که بخورد', 'en' => 'Uses things to pretend, like feeding a block to a doll'], null, ['age_months' => 30, 'domain' => 'cognitive', 'band_source' => 'cdc_2022']),
            $row('child_milestones', 'm30_jumps_both_feet', 67, ['fa' => 'با هر دو پا از زمین می‌پرد', 'en' => 'Jumps off the ground with both feet'], null, ['age_months' => 30, 'domain' => 'motor', 'band_source' => 'cdc_2022']),
            $row('child_milestone_activities', 'm15_build_knock', 18, ['fa' => 'بساز و بریز', 'en' => 'Build and topple'], ['fa' => 'دو سه مکعب یا لیوان را روی هم بگذار و بگذار او خرابشان کند. بعد نوبت خودش است که بسازد.', 'en' => 'Stack two or three blocks or cups and let them knock them over. Then it’s their turn to build.'], ['age_months' => 15, 'band_source' => 'cdc_2022']),
            $row('child_milestone_activities', 'm30_jump_line', 19, ['fa' => 'پرش از روی خط', 'en' => 'Jump the line'], ['fa' => 'با نوار چسب روی زمین خطی بکش و با هم از رویش بپرید. هر پرش را بلند بشمار.', 'en' => 'Put a strip of tape on the floor and jump over it together, counting each jump out loud.'], ['age_months' => 30, 'band_source' => 'cdc_2022']),
            $row('child_milestone_notes', 'm15', 12, ['fa' => 'مراحل رشد ۱۵ ماهگی', 'en' => '15-month milestones'], ['fa' => 'اگر تا ۱۵ ماهگی هیچ کلمه‌ای نمی‌گوید، برای نشان دادن چیزی اشاره نمی‌کند یا مهارتی را که داشت از دست داده، در مراجعه بعدی با پزشک کودک در میان بگذار.', 'en' => 'If by 15 months they say no words, don’t point to show you things, or lose a skill they had, mention it to the paediatrician at the next visit.'], ['age_months' => 15, 'band_source' => 'cdc_2022']),
            $row('child_milestone_notes', 'm30', 13, ['fa' => 'مراحل رشد ۳۰ ماهگی', 'en' => '30-month milestones'], ['fa' => 'اگر تا ۳۰ ماهگی دو کلمه را با هم نمی‌گوید، کارهای ساده را تقلید نمی‌کند یا مهارتی را که داشت از دست داده، با پزشک کودک صحبت کن.', 'en' => 'If by 30 months they don’t put two words together, don’t copy simple actions, or lose a skill they had, talk to the paediatrician.'], ['age_months' => 30, 'band_source' => 'cdc_2022']),
        ]);

        $this->threeMonthBand()->update(['meta' => DB::raw("JSON_SET(COALESCE(meta, '{}'), '$.band_source', 'ritme')")]);
    }

    public function down(): void
    {
        $this->threeMonthBand()->update(['meta' => DB::raw("JSON_REMOVE(meta, '$.band_source')")]);

        DB::table('child_milestone_checks')->whereIn('milestone_code', self::MILESTONES)->delete();
        DB::table('catalog_items')->where('group', 'child_milestones')->whereIn('code', self::MILESTONES)->delete();
        DB::table('catalog_items')->where('group', 'child_milestone_activities')->whereIn('code', self::ACTIVITIES)->delete();
        DB::table('catalog_items')->where('group', 'child_milestone_notes')->whereIn('code', self::NOTES)->delete();
    }

    /** The 3-month band's milestones, play ideas and doctor note (seeded by 000031). */
    private function threeMonthBand()
    {
        return DB::table('catalog_items')->where(function ($q) {
            $q->where(fn ($q) => $q->where('group', 'child_milestones')->whereIn('code', ['m3_social_smile', 'm3_head_up_tummy', 'm3_reacts_to_sounds', 'm3_hands_to_mouth', 'm3_coos_vowels', 'm3_tracks_objects', 'm3_hands_together']))
                ->orWhere(fn ($q) => $q->where('group', 'child_milestone_activities')->whereIn('code', ['m3_tummy_time', 'm3_sound_talk']))
                ->orWhere(fn ($q) => $q->where('group', 'child_milestone_notes')->where('code', 'm3'));
        });
    }
};

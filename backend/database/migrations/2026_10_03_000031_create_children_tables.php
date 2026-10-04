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
     * Twin of backend-go/db/migrations/00031_children.sql
     * (docs/go-migration/migrations.md): the children tables (bloom B-N5-02)
     * used only by the Go internal/children package, the
     * family_children → children FK announced in 00026, and the same
     * `catalog_items` seed rows (child_* groups, needs clinical review), so
     * `make schema-diff` row counts match. No model or routes here.
     * insertOrIgnore never overwrites an admin edit.
     */
    public function up(): void
    {
        Schema::create('children', function (Blueprint $table) {
            $table->id();
            $table->foreignId('owner_id')->constrained('users')->cascadeOnDelete();
            $table->string('name', 64);
            $table->date('birth_date');
            $table->string('sex', 8)->nullable();                 // girl|boy
            $table->decimal('birth_weight_kg', 5, 3)->nullable();
            $table->decimal('birth_length_cm', 4, 1)->nullable();
            $table->decimal('birth_head_cm', 4, 1)->nullable();
            $table->string('delivery_type', 16)->nullable();      // vaginal|cesarean
            $table->timestamps();

            $table->index(['owner_id', 'birth_date']);
        });

        Schema::create('child_measurements', function (Blueprint $table) {
            $table->id();
            $table->foreignId('child_id')->constrained('children')->cascadeOnDelete();
            $table->date('measured_on');
            $table->decimal('weight_kg', 5, 3)->nullable();
            $table->decimal('length_cm', 4, 1)->nullable();
            $table->decimal('head_cm', 4, 1)->nullable();
            $table->timestamps();

            $table->unique(['child_id', 'measured_on']);
        });

        Schema::create('child_vaccine_doses', function (Blueprint $table) {
            $table->id();
            $table->foreignId('child_id')->constrained('children')->cascadeOnDelete();
            $table->string('dose_code', 64);                      // catalog_items child_vaccines code
            $table->date('given_on');
            $table->string('note', 500)->nullable();
            $table->timestamps();

            $table->unique(['child_id', 'dose_code']);
        });

        Schema::create('child_milestone_checks', function (Blueprint $table) {
            $table->id();
            $table->foreignId('child_id')->constrained('children')->cascadeOnDelete();
            $table->string('milestone_code', 64);                 // catalog_items child_milestones code
            $table->date('checked_on');
            $table->timestamps();

            $table->unique(['child_id', 'milestone_code']);
        });

        DB::table('family_children')->whereNotIn('child_id', DB::table('children')->select('id'))->delete();
        Schema::table('family_children', function (Blueprint $table) {
            $table->foreign('child_id')->references('id')->on('children')->cascadeOnDelete();
        });

        $now = now();
        $json = fn (?array $v) => $v === null ? null : json_encode($v, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $row = fn (string $group, string $code, int $sort, array $title, ?array $body, ?array $meta) => [
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
            $row('child_vaccines', 'bcg', 1, ['fa' => 'ب.ث.ژ (BCG)', 'en' => 'BCG'], ['fa' => 'سل', 'en' => 'Tuberculosis'], ['visit' => 'birth', 'age_months' => 0]),
            $row('child_vaccines', 'hepb_1', 2, ['fa' => 'هپاتیت B نوبت اول', 'en' => 'Hepatitis B, dose 1'], ['fa' => 'هپاتیت B', 'en' => 'Hepatitis B'], ['visit' => 'birth', 'age_months' => 0]),
            $row('child_vaccines', 'opv_0', 3, ['fa' => 'فلج اطفال خوراکی', 'en' => 'Oral polio (OPV)'], ['fa' => 'فلج اطفال', 'en' => 'Polio'], ['visit' => 'birth', 'age_months' => 0]),
            $row('child_vaccines', 'penta_1', 4, ['fa' => 'پنتاوالان نوبت اول', 'en' => 'Pentavalent, dose 1'], ['fa' => 'دیفتری، کزاز، سیاه‌سرفه، هپاتیت B و هموفیلوس آنفلوانزای نوع b', 'en' => 'Diphtheria, tetanus, whooping cough, hepatitis B and Hib'], ['visit' => 'm2', 'age_months' => 2]),
            $row('child_vaccines', 'polio_m2', 5, ['fa' => 'فلج اطفال (خوراکی + تزریقی)', 'en' => 'Polio (oral + injectable)'], ['fa' => 'فلج اطفال', 'en' => 'Polio'], ['visit' => 'm2', 'age_months' => 2]),
            $row('child_vaccines', 'pcv_1', 6, ['fa' => 'پنوموکوک نوبت اول', 'en' => 'Pneumococcal, dose 1'], ['fa' => 'عفونت‌های پنوموکوکی (ذات‌الریه، مننژیت، عفونت گوش)', 'en' => 'Pneumococcal disease (pneumonia, meningitis, ear infections)'], ['visit' => 'm2', 'age_months' => 2]),
            $row('child_vaccines', 'rota_1', 7, ['fa' => 'روتاویروس نوبت اول', 'en' => 'Rotavirus, dose 1'], ['fa' => 'اسهال شدید روتاویروسی', 'en' => 'Severe rotavirus diarrhoea'], ['visit' => 'm2', 'age_months' => 2]),
            $row('child_vaccines', 'penta_2', 8, ['fa' => 'پنتاوالان نوبت دوم', 'en' => 'Pentavalent, dose 2'], ['fa' => 'دیفتری، کزاز، سیاه‌سرفه، هپاتیت B و هموفیلوس آنفلوانزای نوع b', 'en' => 'Diphtheria, tetanus, whooping cough, hepatitis B and Hib'], ['visit' => 'm4', 'age_months' => 4]),
            $row('child_vaccines', 'opv_m4', 9, ['fa' => 'فلج اطفال خوراکی', 'en' => 'Oral polio (OPV)'], ['fa' => 'فلج اطفال', 'en' => 'Polio'], ['visit' => 'm4', 'age_months' => 4]),
            $row('child_vaccines', 'pcv_2', 10, ['fa' => 'پنوموکوک نوبت دوم', 'en' => 'Pneumococcal, dose 2'], ['fa' => 'عفونت‌های پنوموکوکی (ذات‌الریه، مننژیت، عفونت گوش)', 'en' => 'Pneumococcal disease (pneumonia, meningitis, ear infections)'], ['visit' => 'm4', 'age_months' => 4]),
            $row('child_vaccines', 'rota_2', 11, ['fa' => 'روتاویروس نوبت دوم', 'en' => 'Rotavirus, dose 2'], ['fa' => 'اسهال شدید روتاویروسی', 'en' => 'Severe rotavirus diarrhoea'], ['visit' => 'm4', 'age_months' => 4]),
            $row('child_vaccines', 'penta_3', 12, ['fa' => 'پنتاوالان نوبت سوم', 'en' => 'Pentavalent, dose 3'], ['fa' => 'دیفتری، کزاز، سیاه‌سرفه، هپاتیت B و هموفیلوس آنفلوانزای نوع b', 'en' => 'Diphtheria, tetanus, whooping cough, hepatitis B and Hib'], ['visit' => 'm6', 'age_months' => 6]),
            $row('child_vaccines', 'polio_m6', 13, ['fa' => 'فلج اطفال (خوراکی + تزریقی)', 'en' => 'Polio (oral + injectable)'], ['fa' => 'فلج اطفال', 'en' => 'Polio'], ['visit' => 'm6', 'age_months' => 6]),
            $row('child_vaccines', 'pcv_3', 14, ['fa' => 'پنوموکوک نوبت سوم', 'en' => 'Pneumococcal, dose 3'], ['fa' => 'عفونت‌های پنوموکوکی (ذات‌الریه، مننژیت، عفونت گوش)', 'en' => 'Pneumococcal disease (pneumonia, meningitis, ear infections)'], ['visit' => 'm6', 'age_months' => 6]),
            $row('child_vaccines', 'mmr_1', 15, ['fa' => 'MMR نوبت اول', 'en' => 'MMR, dose 1'], ['fa' => 'سرخک، اوریون و سرخجه', 'en' => 'Measles, mumps and rubella'], ['visit' => 'm12', 'age_months' => 12]),
            $row('child_vaccines', 'pcv_booster', 16, ['fa' => 'پنوموکوک یادآور', 'en' => 'Pneumococcal booster'], ['fa' => 'عفونت‌های پنوموکوکی (ذات‌الریه، مننژیت، عفونت گوش)', 'en' => 'Pneumococcal disease (pneumonia, meningitis, ear infections)'], ['visit' => 'm12', 'age_months' => 12]),
            $row('child_vaccines', 'dtp_booster_1', 17, ['fa' => 'سه‌گانه یادآور', 'en' => 'DTP booster'], ['fa' => 'دیفتری، کزاز و سیاه‌سرفه', 'en' => 'Diphtheria, tetanus and whooping cough'], ['visit' => 'm18', 'age_months' => 18]),
            $row('child_vaccines', 'opv_booster_1', 18, ['fa' => 'فلج اطفال یادآور', 'en' => 'Polio booster'], ['fa' => 'فلج اطفال', 'en' => 'Polio'], ['visit' => 'm18', 'age_months' => 18]),
            $row('child_vaccines', 'mmr_2', 19, ['fa' => 'MMR نوبت دوم', 'en' => 'MMR, dose 2'], ['fa' => 'سرخک، اوریون و سرخجه', 'en' => 'Measles, mumps and rubella'], ['visit' => 'm18', 'age_months' => 18]),
            $row('child_vaccines', 'dtp_booster_2', 20, ['fa' => 'سه‌گانه یادآور دوم', 'en' => 'DTP second booster'], ['fa' => 'دیفتری، کزاز و سیاه‌سرفه', 'en' => 'Diphtheria, tetanus and whooping cough'], ['visit' => 'y6', 'age_months' => 72]),
            $row('child_vaccines', 'opv_booster_2', 21, ['fa' => 'فلج اطفال یادآور دوم', 'en' => 'Polio second booster'], ['fa' => 'فلج اطفال', 'en' => 'Polio'], ['visit' => 'y6', 'age_months' => 72]),
            $row('child_milestones', 'm2_smiles_back', 1, ['fa' => 'وقتی با او حرف می‌زنی لبخند می‌زند', 'en' => 'Smiles when you talk to or smile at them'], null, ['age_months' => 2, 'domain' => 'social']),
            $row('child_milestones', 'm2_looks_at_face', 2, ['fa' => 'به صورتت نگاه می‌کند', 'en' => 'Looks at your face'], null, ['age_months' => 2, 'domain' => 'social']),
            $row('child_milestones', 'm2_coos', 3, ['fa' => 'صداهایی غیر از گریه درمی‌آورد', 'en' => 'Makes sounds other than crying'], null, ['age_months' => 2, 'domain' => 'language']),
            $row('child_milestones', 'm2_lifts_head', 4, ['fa' => 'روی شکم، سرش را کمی بالا می‌آورد', 'en' => 'Lifts head a little during tummy time'], null, ['age_months' => 2, 'domain' => 'motor']),
            $row('child_milestones', 'm2_moves_limbs', 5, ['fa' => 'هر دو دست و هر دو پا را حرکت می‌دهد', 'en' => 'Moves both arms and both legs'], null, ['age_months' => 2, 'domain' => 'motor']),
            $row('child_milestones', 'm3_social_smile', 6, ['fa' => 'لبخند اجتماعی می‌زند', 'en' => 'Gives a social smile'], null, ['age_months' => 3, 'domain' => 'social']),
            $row('child_milestones', 'm3_head_up_tummy', 7, ['fa' => 'سر را موقع دمر خوابیدن بالا می‌گیرد', 'en' => 'Holds head up during tummy time'], null, ['age_months' => 3, 'domain' => 'motor']),
            $row('child_milestones', 'm3_reacts_to_sounds', 8, ['fa' => 'به صداها واکنش نشان می‌دهد', 'en' => 'Reacts to sounds'], null, ['age_months' => 3, 'domain' => 'language']),
            $row('child_milestones', 'm3_hands_to_mouth', 9, ['fa' => 'دست‌ها را به دهان می‌برد', 'en' => 'Brings hands to mouth'], null, ['age_months' => 3, 'domain' => 'motor']),
            $row('child_milestones', 'm3_coos_vowels', 10, ['fa' => 'صداهای «آ» و «او» درمی‌آورد', 'en' => 'Makes “ah” and “oo” sounds'], null, ['age_months' => 3, 'domain' => 'language']),
            $row('child_milestones', 'm3_tracks_objects', 11, ['fa' => 'اشیا را با چشم دنبال می‌کند', 'en' => 'Follows things with the eyes'], null, ['age_months' => 3, 'domain' => 'cognitive']),
            $row('child_milestones', 'm3_hands_together', 12, ['fa' => 'دست‌ها را جلوی صورت به هم می‌رساند', 'en' => 'Brings hands together in front of the face'], null, ['age_months' => 3, 'domain' => 'motor']),
            $row('child_milestones', 'm4_laughs', 13, ['fa' => 'بلند می‌خندد', 'en' => 'Laughs out loud'], null, ['age_months' => 4, 'domain' => 'social']),
            $row('child_milestones', 'm4_holds_head_steady', 14, ['fa' => 'سرش را بدون کمک ثابت نگه می‌دارد', 'en' => 'Holds head steady without support'], null, ['age_months' => 4, 'domain' => 'motor']),
            $row('child_milestones', 'm4_holds_toy', 15, ['fa' => 'اسباب‌بازی را در دست نگه می‌دارد', 'en' => 'Holds a toy put in the hand'], null, ['age_months' => 4, 'domain' => 'motor']),
            $row('child_milestones', 'm4_pushes_up_elbows', 16, ['fa' => 'روی شکم، روی آرنج‌ها خودش را بالا می‌کشد', 'en' => 'Pushes up onto elbows on the tummy'], null, ['age_months' => 4, 'domain' => 'motor']),
            $row('child_milestones', 'm4_turns_to_voice', 17, ['fa' => 'به سمت صدای تو برمی‌گردد', 'en' => 'Turns toward your voice'], null, ['age_months' => 4, 'domain' => 'language']),
            $row('child_milestones', 'm6_knows_familiar', 18, ['fa' => 'آدم‌های آشنا را می‌شناسد', 'en' => 'Knows familiar people'], null, ['age_months' => 6, 'domain' => 'social']),
            $row('child_milestones', 'm6_takes_turns_sounds', 19, ['fa' => 'با تو نوبتی صدا درمی‌آورد', 'en' => 'Takes turns making sounds with you'], null, ['age_months' => 6, 'domain' => 'language']),
            $row('child_milestones', 'm6_rolls_over', 20, ['fa' => 'از شکم به پشت غلت می‌زند', 'en' => 'Rolls from tummy to back'], null, ['age_months' => 6, 'domain' => 'motor']),
            $row('child_milestones', 'm6_reaches_toy', 21, ['fa' => 'برای برداشتن اسباب‌بازی دست دراز می‌کند', 'en' => 'Reaches for a toy'], null, ['age_months' => 6, 'domain' => 'motor']),
            $row('child_milestones', 'm6_sits_with_support', 22, ['fa' => 'با تکیه به دست‌ها می‌نشیند', 'en' => 'Leans on the hands to sit'], null, ['age_months' => 6, 'domain' => 'motor']),
            $row('child_milestones', 'm9_stranger_aware', 23, ['fa' => 'با غریبه‌ها کمی خجالتی یا نگران است', 'en' => 'Is shy or wary with strangers'], null, ['age_months' => 9, 'domain' => 'social']),
            $row('child_milestones', 'm9_responds_to_name', 24, ['fa' => 'به اسمش واکنش نشان می‌دهد', 'en' => 'Responds to their name'], null, ['age_months' => 9, 'domain' => 'language']),
            $row('child_milestones', 'm9_sits_alone', 25, ['fa' => 'بدون کمک می‌نشیند', 'en' => 'Sits without support'], null, ['age_months' => 9, 'domain' => 'motor']),
            $row('child_milestones', 'm9_passes_hands', 26, ['fa' => 'اشیا را از یک دست به دست دیگر می‌دهد', 'en' => 'Moves things from one hand to the other'], null, ['age_months' => 9, 'domain' => 'motor']),
            $row('child_milestones', 'm9_babbles_syllables', 27, ['fa' => 'صداهایی مثل «ماماما» و «بابابا» درمی‌آورد', 'en' => 'Babbles “mamama” and “bababa”'], null, ['age_months' => 9, 'domain' => 'language']),
            $row('child_milestones', 'm12_waves_bye', 28, ['fa' => 'دست تکان می‌دهد (بای‌بای)', 'en' => 'Waves bye-bye'], null, ['age_months' => 12, 'domain' => 'social']),
            $row('child_milestones', 'm12_says_mama_dada', 29, ['fa' => '«ماما» یا «بابا» را با معنی می‌گوید', 'en' => 'Says “mama” or “dada” for a parent'], null, ['age_months' => 12, 'domain' => 'language']),
            $row('child_milestones', 'm12_pulls_to_stand', 30, ['fa' => 'خودش را بالا می‌کشد و می‌ایستد', 'en' => 'Pulls up to stand'], null, ['age_months' => 12, 'domain' => 'motor']),
            $row('child_milestones', 'm12_cruises', 31, ['fa' => 'با گرفتن مبل راه می‌رود', 'en' => 'Walks holding on to furniture'], null, ['age_months' => 12, 'domain' => 'motor']),
            $row('child_milestones', 'm12_pincer_grasp', 32, ['fa' => 'چیزهای کوچک را با شست و انگشت اشاره برمی‌دارد', 'en' => 'Picks things up between thumb and pointer finger'], null, ['age_months' => 12, 'domain' => 'motor']),
            $row('child_milestones', 'm18_walks_alone', 33, ['fa' => 'بدون کمک چند قدم راه می‌رود', 'en' => 'Walks a few steps alone'], null, ['age_months' => 18, 'domain' => 'motor']),
            $row('child_milestones', 'm18_points_to_show', 34, ['fa' => 'برای نشان دادن چیزی به آن اشاره می‌کند', 'en' => 'Points to show you something'], null, ['age_months' => 18, 'domain' => 'social']),
            $row('child_milestones', 'm18_says_words', 35, ['fa' => 'چند کلمه غیر از ماما و بابا می‌گوید', 'en' => 'Says a few words besides mama and dada'], null, ['age_months' => 18, 'domain' => 'language']),
            $row('child_milestones', 'm18_uses_spoon', 36, ['fa' => 'سعی می‌کند با قاشق غذا بخورد', 'en' => 'Tries to eat with a spoon'], null, ['age_months' => 18, 'domain' => 'motor']),
            $row('child_milestones', 'm18_scribbles', 37, ['fa' => 'خط‌خطی می‌کند', 'en' => 'Scribbles'], null, ['age_months' => 18, 'domain' => 'motor']),
            $row('child_milestones', 'm24_two_word_phrases', 38, ['fa' => 'دو کلمه را با هم می‌گوید', 'en' => 'Puts two words together'], null, ['age_months' => 24, 'domain' => 'language']),
            $row('child_milestones', 'm24_kicks_ball', 39, ['fa' => 'به توپ ضربه می‌زند', 'en' => 'Kicks a ball'], null, ['age_months' => 24, 'domain' => 'motor']),
            $row('child_milestones', 'm24_runs', 40, ['fa' => 'می‌دود', 'en' => 'Runs'], null, ['age_months' => 24, 'domain' => 'motor']),
            $row('child_milestones', 'm24_points_in_book', 41, ['fa' => 'در کتاب به تصاویر اشاره می‌کند', 'en' => 'Points to pictures in a book'], null, ['age_months' => 24, 'domain' => 'cognitive']),
            $row('child_milestones', 'm24_notices_feelings', 42, ['fa' => 'متوجه ناراحتی دیگران می‌شود', 'en' => 'Notices when others are upset'], null, ['age_months' => 24, 'domain' => 'social']),
            $row('child_milestones', 'm36_converses', 43, ['fa' => 'در گفت‌وگو چند جمله رد و بدل می‌کند', 'en' => 'Has a back-and-forth conversation'], null, ['age_months' => 36, 'domain' => 'language']),
            $row('child_milestones', 'm36_says_own_name', 44, ['fa' => 'اسم کوچکش را می‌گوید', 'en' => 'Says their first name'], null, ['age_months' => 36, 'domain' => 'language']),
            $row('child_milestones', 'm36_draws_circle', 45, ['fa' => 'با دیدن نمونه دایره می‌کشد', 'en' => 'Draws a circle after you show how'], null, ['age_months' => 36, 'domain' => 'motor']),
            $row('child_milestones', 'm36_dresses_partly', 46, ['fa' => 'بعضی لباس‌ها را خودش می‌پوشد', 'en' => 'Puts on some clothes alone'], null, ['age_months' => 36, 'domain' => 'motor']),
            $row('child_milestones', 'm36_plays_with_others', 47, ['fa' => 'با بچه‌های دیگر بازی می‌کند', 'en' => 'Plays with other children'], null, ['age_months' => 36, 'domain' => 'social']),
            $row('child_milestones', 'm48_tells_story', 48, ['fa' => 'اتفاقی را که افتاده تعریف می‌کند', 'en' => 'Tells what happened in a story or their day'], null, ['age_months' => 48, 'domain' => 'language']),
            $row('child_milestones', 'm48_names_colours', 49, ['fa' => 'چند رنگ را نام می‌برد', 'en' => 'Names a few colours'], null, ['age_months' => 48, 'domain' => 'cognitive']),
            $row('child_milestones', 'm48_pretend_play', 50, ['fa' => 'بازی نقش‌آفرینی می‌کند', 'en' => 'Plays pretend'], null, ['age_months' => 48, 'domain' => 'social']),
            $row('child_milestones', 'm48_catches_ball', 51, ['fa' => 'توپ بزرگ را می‌گیرد', 'en' => 'Catches a large ball'], null, ['age_months' => 48, 'domain' => 'motor']),
            $row('child_milestones', 'm48_comforts_others', 52, ['fa' => 'دیگران را دلداری می‌دهد', 'en' => 'Comforts others who are hurt or sad'], null, ['age_months' => 48, 'domain' => 'social']),
            $row('child_milestones', 'm60_counts_to_ten', 53, ['fa' => 'تا ۱۰ می‌شمارد', 'en' => 'Counts to 10'], null, ['age_months' => 60, 'domain' => 'cognitive']),
            $row('child_milestones', 'm60_hops_one_foot', 54, ['fa' => 'روی یک پا لی‌لی می‌کند', 'en' => 'Hops on one foot'], null, ['age_months' => 60, 'domain' => 'motor']),
            $row('child_milestones', 'm60_writes_letters', 55, ['fa' => 'چند حرف از اسمش را می‌نویسد', 'en' => 'Writes some letters of their name'], null, ['age_months' => 60, 'domain' => 'motor']),
            $row('child_milestones', 'm60_follows_rules', 56, ['fa' => 'در بازی‌ها قانون را رعایت می‌کند', 'en' => 'Follows rules in simple games'], null, ['age_months' => 60, 'domain' => 'social']),
            $row('child_milestones', 'm60_buttons', 57, ['fa' => 'دکمه‌ها را باز و بسته می‌کند', 'en' => 'Does and undoes buttons'], null, ['age_months' => 60, 'domain' => 'motor']),
            $row('child_milestone_activities', 'm2_face_time', 1, ['fa' => 'صورت به صورت', 'en' => 'Face to face'], ['fa' => 'در فاصله ۲۰ تا ۳۰ سانتی صورتش با او حرف بزن و لبخند بزن.', 'en' => 'Talk and smile 20–30 cm from their face.'], ['age_months' => 2]),
            $row('child_milestone_activities', 'm2_tummy_time', 2, ['fa' => 'وقت دمر (Tummy time)', 'en' => 'Tummy time'], ['fa' => 'روزی چند بار، هر بار یکی دو دقیقه وقتی بیدار است روی شکم بگذارش.', 'en' => 'A few times a day, a minute or two on the tummy while awake.'], ['age_months' => 2]),
            $row('child_milestone_activities', 'm3_tummy_time', 3, ['fa' => 'وقت دمر (Tummy time)', 'en' => 'Tummy time'], ['fa' => 'روزی چند بار، هر بار ۳ تا ۵ دقیقه روی شکم؛ گردن و شانه قوی می‌شود.', 'en' => 'A few times a day, 3 to 5 minutes on the tummy; it strengthens neck and shoulders.'], ['age_months' => 3]),
            $row('child_milestone_activities', 'm3_sound_talk', 4, ['fa' => 'گفت‌وگوی صدا', 'en' => 'Sound conversations'], ['fa' => 'صداهایش را تکرار کن و منتظر جوابش بمان؛ پایه زبان است.', 'en' => 'Repeat their sounds and wait for a reply; it is the root of language.'], ['age_months' => 3]),
            $row('child_milestone_activities', 'm4_rattle_reach', 5, ['fa' => 'جغجغه در دسترس', 'en' => 'Reach for the rattle'], ['fa' => 'اسباب‌بازی را کمی دورتر بگیر تا برای گرفتنش تلاش کند.', 'en' => 'Hold a toy just out of reach so they stretch for it.'], ['age_months' => 4]),
            $row('child_milestone_activities', 'm4_mirror_play', 6, ['fa' => 'بازی با آینه', 'en' => 'Mirror play'], ['fa' => 'جلوی آینه با او حرف بزن و چهره‌اش را نشانش بده.', 'en' => 'Talk to them in front of a mirror and show them their face.'], ['age_months' => 4]),
            $row('child_milestone_activities', 'm6_peekaboo', 7, ['fa' => 'دالی‌موشه', 'en' => 'Peekaboo'], ['fa' => 'صورتت را بپوشان و دوباره نشان بده؛ می‌خندد و منتظر می‌ماند.', 'en' => 'Hide your face and show it again; they laugh and wait for it.'], ['age_months' => 6]),
            $row('child_milestone_activities', 'm6_supported_sitting', 8, ['fa' => 'نشستن با تکیه', 'en' => 'Supported sitting'], ['fa' => 'با بالش دورش را امن کن و بگذار نشستن را تمرین کند.', 'en' => 'Surround them with cushions and let them practise sitting.'], ['age_months' => 6]),
            $row('child_milestone_activities', 'm9_name_things', 9, ['fa' => 'اسم چیزها', 'en' => 'Name things'], ['fa' => 'به چیزهایی که نگاه می‌کند اشاره کن و اسمشان را بگو.', 'en' => 'Point to what they look at and say its name.'], ['age_months' => 9]),
            $row('child_milestone_activities', 'm9_crawl_course', 10, ['fa' => 'مسیر چهاردست‌وپا', 'en' => 'Crawling course'], ['fa' => 'اسباب‌بازی را کمی دورتر بگذار تا به سمتش برود.', 'en' => 'Put a toy a little further away so they move toward it.'], ['age_months' => 9]),
            $row('child_milestone_activities', 'm12_stack_cups', 11, ['fa' => 'لیوان‌های تودرتو', 'en' => 'Stacking cups'], ['fa' => 'لیوان‌ها را روی هم بچینید و خراب کنید.', 'en' => 'Stack cups together and knock them down.'], ['age_months' => 12]),
            $row('child_milestone_activities', 'm12_read_together', 12, ['fa' => 'کتاب خواندن با هم', 'en' => 'Read together'], ['fa' => 'کتاب‌های تصویری بخوان و به تصویرها اشاره کن.', 'en' => 'Read picture books and point to the pictures.'], ['age_months' => 12]),
            $row('child_milestone_activities', 'm18_simple_tasks', 13, ['fa' => 'کارهای کوچک', 'en' => 'Little jobs'], ['fa' => 'از او بخواه چیزی را بیاورد یا در سبد بگذارد.', 'en' => 'Ask them to fetch something or put it in a basket.'], ['age_months' => 18]),
            $row('child_milestone_activities', 'm24_pretend_kitchen', 14, ['fa' => 'آشپزی خیالی', 'en' => 'Pretend kitchen'], ['fa' => 'با ظرف‌های پلاستیکی بازی آشپزی کنید و اسم غذاها را بگو.', 'en' => 'Play cooking with plastic dishes and name the foods.'], ['age_months' => 24]),
            $row('child_milestone_activities', 'm36_story_turns', 15, ['fa' => 'قصه نوبتی', 'en' => 'Story turns'], ['fa' => 'یک جمله از قصه را تو بگو و جمله بعدی را او.', 'en' => 'You say one sentence of a story and they say the next.'], ['age_months' => 36]),
            $row('child_milestone_activities', 'm48_colour_hunt', 16, ['fa' => 'رنگ‌یابی', 'en' => 'Colour hunt'], ['fa' => 'در خانه یا پارک دنبال چیزهای یک رنگ بگردید.', 'en' => 'Look for things of one colour at home or in the park.'], ['age_months' => 48]),
            $row('child_milestone_activities', 'm60_board_games', 17, ['fa' => 'بازی‌های نوبتی', 'en' => 'Turn-taking games'], ['fa' => 'بازی‌های ساده با قانون، مثل مار و پله، صبر و شمردن را تمرین می‌دهد.', 'en' => 'Simple rule games like snakes and ladders practise patience and counting.'], ['age_months' => 60]),
            $row('child_milestone_notes', 'm2', 1, ['fa' => 'مراحل رشد ۲ ماهگی', 'en' => '2-month milestones'], ['fa' => 'اگر تا ۲ ماهگی به صداهای بلند واکنش نداد یا وقتی دمر است سرش را اصلاً بالا نیاورد، در مراجعه بعدی با پزشک در میان بگذار.', 'en' => 'If by 2 months they don\'t react to loud sounds or never lift their head on the tummy, mention it at the next visit.'], ['age_months' => 2]),
            $row('child_milestone_notes', 'm3', 2, ['fa' => 'مراحل رشد ۳ ماهگی', 'en' => '3-month milestones'], ['fa' => 'اگر تا ۴ ماهگی به صدا واکنش نداد، سر را بالا نگرفت یا لبخند نزد، در مراجعه بعدی با پزشک در میان بگذار.', 'en' => 'If by 4 months they don\'t react to sounds, hold up their head or smile, mention it at the next visit.'], ['age_months' => 3]),
            $row('child_milestone_notes', 'm4', 3, ['fa' => 'مراحل رشد ۴ ماهگی', 'en' => '4-month milestones'], ['fa' => 'اگر تا ۴ ماهگی سرش را ثابت نگه نمی‌دارد یا اشیا را با چشم دنبال نمی‌کند، با پزشک کودک صحبت کن.', 'en' => 'If by 4 months they can\'t hold their head steady or follow things with the eyes, talk to the paediatrician.'], ['age_months' => 4]),
            $row('child_milestone_notes', 'm6', 4, ['fa' => 'مراحل رشد ۶ ماهگی', 'en' => '6-month milestones'], ['fa' => 'اگر تا ۶ ماهگی برای گرفتن چیزی دست دراز نمی‌کند یا به آدم‌های آشنا واکنش نشان نمی‌دهد، با پزشک صحبت کن.', 'en' => 'If by 6 months they don\'t reach for things or respond to familiar people, talk to the doctor.'], ['age_months' => 6]),
            $row('child_milestone_notes', 'm9', 5, ['fa' => 'مراحل رشد ۹ ماهگی', 'en' => '9-month milestones'], ['fa' => 'اگر تا ۹ ماهگی با تکیه هم نمی‌نشیند یا به اسمش واکنشی ندارد، در مراجعه بعدی مطرح کن.', 'en' => 'If by 9 months they can\'t sit even with support or don\'t respond to their name, raise it at the next visit.'], ['age_months' => 9]),
            $row('child_milestone_notes', 'm12', 6, ['fa' => 'مراحل رشد ۱۲ ماهگی', 'en' => '12-month milestones'], ['fa' => 'اگر تا ۱۲ ماهگی هیچ کلمه یا اشاره‌ای ندارد یا مهارتی را که داشت از دست داده، با پزشک صحبت کن.', 'en' => 'If by 12 months there are no words or gestures, or a skill they had is lost, talk to the doctor.'], ['age_months' => 12]),
            $row('child_milestone_notes', 'm18', 7, ['fa' => 'مراحل رشد ۱۸ ماهگی', 'en' => '18-month milestones'], ['fa' => 'اگر تا ۱۸ ماهگی راه نمی‌رود یا چند کلمه نمی‌گوید، با پزشک کودک در میان بگذار.', 'en' => 'If by 18 months they don\'t walk or say a few words, mention it to the paediatrician.'], ['age_months' => 18]),
            $row('child_milestone_notes', 'm24', 8, ['fa' => 'مراحل رشد ۲۴ ماهگی', 'en' => '24-month milestones'], ['fa' => 'اگر تا ۲ سالگی دو کلمه را با هم نمی‌گوید یا کارهای ساده را تقلید نمی‌کند، با پزشک صحبت کن.', 'en' => 'If by 2 years they don\'t put two words together or copy simple actions, talk to the doctor.'], ['age_months' => 24]),
            $row('child_milestone_notes', 'm36', 9, ['fa' => 'مراحل رشد ۳۶ ماهگی', 'en' => '36-month milestones'], ['fa' => 'اگر تا ۳ سالگی جمله ساده نمی‌گوید یا با بچه‌های دیگر بازی نمی‌کند، با پزشک صحبت کن.', 'en' => 'If by 3 years they don\'t speak in simple sentences or play with other children, talk to the doctor.'], ['age_months' => 36]),
            $row('child_milestone_notes', 'm48', 10, ['fa' => 'مراحل رشد ۴۸ ماهگی', 'en' => '48-month milestones'], ['fa' => 'اگر تا ۴ سالگی حرفش برای غریبه‌ها قابل فهم نیست یا بازی خیالی ندارد، با پزشک صحبت کن.', 'en' => 'If by 4 years strangers can\'t understand their speech or there is no pretend play, talk to the doctor.'], ['age_months' => 48]),
            $row('child_milestone_notes', 'm60', 11, ['fa' => 'مراحل رشد ۶۰ ماهگی', 'en' => '60-month milestones'], ['fa' => 'اگر تا ۵ سالگی نمی‌تواند داستان کوتاهی تعریف کند یا مهارتی را از دست داده، با پزشک صحبت کن.', 'en' => 'If by 5 years they can\'t tell a short story or a skill is lost, talk to the doctor.'], ['age_months' => 60]),
            $row('child_age_notes', 'm0', 1, ['fa' => 'نوزادی', 'en' => 'Newborn'], ['fa' => 'نوزادها بیشتر روز را می‌خوابند و هر ۲ تا ۳ ساعت شیر می‌خورند. نگاه کردن به صورت تو و شنیدن صدایت بهترین بازی این روزهاست.', 'en' => 'Newborns sleep most of the day and feed every 2–3 hours. Looking at your face and hearing your voice is the best play right now.'], ['age_months' => 0]),
            $row('child_age_notes', 'm2', 2, ['fa' => '۲ ماهگی', 'en' => '2 months'], ['fa' => 'در این سن بیشتر نوزادها اولین لبخندهای اجتماعی را می‌زنند و صداهای نرم درمی‌آورند. هر بچه با سرعت خودش پیش می‌رود.', 'en' => 'Around now most babies give their first social smiles and make soft cooing sounds. Every baby goes at their own pace.'], ['age_months' => 2]),
            $row('child_age_notes', 'm3', 3, ['fa' => '۳ ماهگی', 'en' => '3 months'], ['fa' => 'بیشتر نوزادها در این سن سرشان را موقع دمر خوابیدن بالا نگه می‌دارند، به صداها می‌خندند و دست‌ها را به دهان می‌برند. اگر هنوز نه، عجله نکن؛ بازه طبیعی وسیع است.', 'en' => 'Most babies this age hold their head up during tummy time, laugh at sounds and bring their hands to the mouth. If not yet, no rush; the normal range is wide.'], ['age_months' => 3]),
            $row('child_age_notes', 'm4', 4, ['fa' => '۴ ماهگی', 'en' => '4 months'], ['fa' => 'خنده‌های بلند، گرفتن اسباب‌بازی و غلت‌های اول در این ماه‌ها شروع می‌شود. وقت دمر را ادامه بده.', 'en' => 'Big laughs, holding toys and first rolls start around now. Keep up tummy time.'], ['age_months' => 4]),
            $row('child_age_notes', 'm6', 5, ['fa' => '۶ ماهگی', 'en' => '6 months'], ['fa' => 'بسیاری از بچه‌ها در حدود ۶ ماهگی آماده غذای کمکی می‌شوند و نشستن با تکیه را تمرین می‌کنند.', 'en' => 'Many babies are ready for first foods around 6 months and practise sitting with support.'], ['age_months' => 6]),
            $row('child_age_notes', 'm9', 6, ['fa' => '۹ ماهگی', 'en' => '9 months'], ['fa' => 'نشستن بدون کمک، چهاردست‌وپا رفتن و غریبی کردن در این سن رایج است.', 'en' => 'Sitting alone, crawling and being wary of strangers are common at this age.'], ['age_months' => 9]),
            $row('child_age_notes', 'm12', 7, ['fa' => '۱۲ ماهگی', 'en' => '12 months'], ['fa' => 'حدود یک‌سالگی خیلی از بچه‌ها با گرفتن مبل راه می‌روند و اولین کلمه‌ها را می‌گویند.', 'en' => 'Around their first birthday many children cruise along furniture and say first words.'], ['age_months' => 12]),
            $row('child_age_notes', 'm18', 8, ['fa' => '۱۸ ماهگی', 'en' => '18 months'], ['fa' => 'نوپاها در این سن راه رفتن را تمرین می‌کنند، چند کلمه می‌گویند و دوست دارند کارها را خودشان انجام دهند.', 'en' => 'Toddlers this age practise walking, say a handful of words and want to do things themselves.'], ['age_months' => 18]),
            $row('child_age_notes', 'm24', 9, ['fa' => '۲۴ ماهگی', 'en' => '24 months'], ['fa' => 'در دو سالگی جمله‌های دوکلمه‌ای، دویدن و بازی‌های تقلیدی رایج است.', 'en' => 'At two, two-word phrases, running and copycat play are common.'], ['age_months' => 24]),
            $row('child_age_notes', 'm36', 10, ['fa' => '۳۶ ماهگی', 'en' => '36 months'], ['fa' => 'سه‌ساله‌ها جمله می‌سازند، سؤال زیاد می‌پرسند و با بچه‌های دیگر بازی می‌کنند.', 'en' => 'Three-year-olds speak in sentences, ask lots of questions and play with other children.'], ['age_months' => 36]),
            $row('child_age_notes', 'm48', 11, ['fa' => '۴۸ ماهگی', 'en' => '48 months'], ['fa' => 'چهارساله‌ها قصه تعریف می‌کنند، رنگ‌ها را می‌شناسند و بازی‌های خیالی دارند.', 'en' => 'Four-year-olds tell stories, know colours and love pretend play.'], ['age_months' => 48]),
            $row('child_age_notes', 'm60', 12, ['fa' => '۶۰ ماهگی', 'en' => '60 months'], ['fa' => 'پنج‌ساله‌ها می‌شمارند، چند حرف می‌نویسند و قانون بازی‌ها را رعایت می‌کنند.', 'en' => 'Five-year-olds count, write some letters and follow the rules of games.'], ['age_months' => 60]),
            $row('child_learn', 'newborn_safe_sleep', 1, ['fa' => 'خواب امن نوزاد', 'en' => 'Safe sleep for newborns'], ['fa' => 'به پشت، روی سطح سفت و بدون بالش و پتوی شل؛ در اتاق خودت ولی در تخت جدا.', 'en' => 'On the back, on a firm flat surface with no pillows or loose blankets; in your room but in their own cot.'], ['topic' => 'sleep', 'from_months' => 0, 'to_months' => 2, 'minutes' => 4, 'featured' => true]),
            $row('child_learn', 'night_sleep_3m', 2, ['fa' => 'خواب شبانه از ۳ ماهگی', 'en' => 'Night sleep from 3 months'], ['fa' => 'چطور روتین ملایم بسازیم: حمام، شیر، نور کم، همان ترتیب هر شب. بدون روش‌های گریه‌درمانی.', 'en' => 'Building a gentle routine: bath, feed, dim light, the same order every night. No cry-it-out methods.'], ['topic' => 'sleep', 'from_months' => 3, 'to_months' => 6, 'minutes' => 4, 'featured' => true]),
            $row('child_learn', 'toddler_sleep_routine', 3, ['fa' => 'روتین خواب نوپا', 'en' => 'Toddler bedtime routine'], ['fa' => 'ساعت ثابت خواب، قصه کوتاه و صفحه‌نمایش خاموش از یک ساعت قبل.', 'en' => 'A steady bedtime, a short story and screens off an hour before.'], ['topic' => 'sleep', 'from_months' => 12, 'to_months' => 60, 'minutes' => 4, 'featured' => true]),
            $row('child_learn', 'breastfeeding_supply', 4, ['fa' => 'شیردهی و افزایش شیر', 'en' => 'Breastfeeding and milk supply'], ['fa' => 'شیردهی بر اساس تقاضا، وضعیت درست گرفتن سینه و استراحت، مهم‌ترین عامل‌ها هستند.', 'en' => 'Feeding on demand, a good latch and rest matter most.'], ['topic' => 'feeding', 'from_months' => 0, 'to_months' => 6, 'minutes' => 6, 'featured' => false]),
            $row('child_learn', 'starting_solids', 5, ['fa' => 'شروع غذای کمکی', 'en' => 'Starting solid foods'], ['fa' => 'از حدود ۶ ماهگی، با یک غذای ساده و نرم شروع کن و هر چند روز یک غذای تازه اضافه کن.', 'en' => 'From about 6 months, start with one simple soft food and add a new one every few days.'], ['topic' => 'feeding', 'from_months' => 5, 'to_months' => 9, 'minutes' => 5, 'featured' => true]),
            $row('child_learn', 'picky_eating', 6, ['fa' => 'بدغذایی نوپا', 'en' => 'Picky eating in toddlers'], ['fa' => 'غذای تازه را بارها و بدون اصرار پیشنهاد بده؛ سهم را بچه تعیین می‌کند.', 'en' => 'Offer new foods many times without pressure; the child decides how much.'], ['topic' => 'feeding', 'from_months' => 12, 'to_months' => 60, 'minutes' => 5, 'featured' => false]),
            $row('child_learn', 'evening_crying', 7, ['fa' => 'دلیل گریه‌های عصرگاهی', 'en' => 'Why babies cry in the evening'], ['fa' => 'گریه‌های عصر در هفته‌های اول رایج است و معمولاً تا ۳ یا ۴ ماهگی کم می‌شود.', 'en' => 'Evening fussiness is common in the first weeks and usually eases by 3 or 4 months.'], ['topic' => 'health', 'from_months' => 0, 'to_months' => 4, 'minutes' => 5, 'featured' => false]),
            $row('child_learn', 'vaccine_4m', 8, ['fa' => 'واکسن ۴ ماهگی: چه انتظاری داشته باشم', 'en' => 'The 4-month vaccines: what to expect'], ['fa' => 'تب خفیف و بی‌قراری یکی دو روز بعد از واکسن طبیعی است؛ در صورت تب بالا با پزشک تماس بگیر.', 'en' => 'A mild fever and fussiness for a day or two are normal; call the doctor for a high fever.'], ['topic' => 'health', 'from_months' => 3, 'to_months' => 5, 'minutes' => 3, 'featured' => false]),
            $row('child_learn', 'teething', 9, ['fa' => 'دندان درآوردن', 'en' => 'Teething'], ['fa' => 'آب دهان زیاد و جویدن رایج است؛ حلقه دندانی خنک کمک می‌کند. تب بالا ربطی به دندان ندارد.', 'en' => 'Drooling and chewing are common; a cool teething ring helps. A high fever is not from teething.'], ['topic' => 'health', 'from_months' => 5, 'to_months' => 18, 'minutes' => 4, 'featured' => false]),
            $row('child_learn', 'neck_play', 10, ['fa' => 'بازی‌های ساده برای تقویت گردن', 'en' => 'Simple games for neck strength'], ['fa' => 'وقت دمر کوتاه و مکرر، با اسباب‌بازی رنگی جلوی صورتش.', 'en' => 'Short, frequent tummy time with a bright toy in front of their face.'], ['topic' => 'play', 'from_months' => 2, 'to_months' => 5, 'minutes' => 4, 'featured' => false]),
            $row('child_learn', 'crawling_play', 11, ['fa' => 'بازی‌های حرکتی ۶ تا ۱۲ ماهگی', 'en' => 'Movement play from 6 to 12 months'], ['fa' => 'فضای امن روی زمین بهترین اسباب‌بازی برای غلت زدن و چهاردست‌وپا رفتن است.', 'en' => 'A safe space on the floor is the best toy for rolling and crawling.'], ['topic' => 'play', 'from_months' => 6, 'to_months' => 12, 'minutes' => 4, 'featured' => false]),
            $row('child_learn', 'toddler_talking', 12, ['fa' => 'کمک به حرف زدن نوپا', 'en' => 'Helping toddlers talk'], ['fa' => 'با او زیاد حرف بزن، کارهایت را توصیف کن و جمله‌هایش را کامل‌تر تکرار کن.', 'en' => 'Talk a lot, describe what you are doing and repeat their words in fuller sentences.'], ['topic' => 'play', 'from_months' => 12, 'to_months' => 36, 'minutes' => 5, 'featured' => false]),
            $row('child_learn', 'mother_self_care', 13, ['fa' => 'مراقبت از خودت در ماه‌های اول', 'en' => 'Looking after yourself in the first months'], ['fa' => 'خواب کوتاه هر وقت ممکن است، کمک گرفتن از اطرافیان و صحبت درباره حالت، بخشی از مراقبت از بچه است.', 'en' => 'Naps whenever you can, accepting help and talking about how you feel are part of caring for your baby.'], ['topic' => 'mother', 'from_months' => 0, 'to_months' => 12, 'minutes' => 7, 'featured' => false]),
            $row('child_learn', 'mother_return_to_routine', 14, ['fa' => 'برگشت به روال کار و زندگی', 'en' => 'Getting back to work and routine'], ['fa' => 'برنامه‌ریزی تدریجی، تقسیم کارها و زمان کوتاهی برای خودت کمک می‌کند.', 'en' => 'A gradual plan, shared chores and a little time for yourself help.'], ['topic' => 'mother', 'from_months' => 6, 'to_months' => 60, 'minutes' => 5, 'featured' => false]),
        ]);
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        DB::table('catalog_items')->where('group', 'child_vaccines')
            ->whereIn('code', ['bcg', 'hepb_1', 'opv_0', 'penta_1', 'polio_m2', 'pcv_1', 'rota_1', 'penta_2', 'opv_m4', 'pcv_2', 'rota_2', 'penta_3', 'polio_m6', 'pcv_3', 'mmr_1', 'pcv_booster', 'dtp_booster_1', 'opv_booster_1', 'mmr_2', 'dtp_booster_2', 'opv_booster_2'])->delete();
        DB::table('catalog_items')->where('group', 'child_milestones')
            ->whereIn('code', ['m2_smiles_back', 'm2_looks_at_face', 'm2_coos', 'm2_lifts_head', 'm2_moves_limbs', 'm3_social_smile', 'm3_head_up_tummy', 'm3_reacts_to_sounds', 'm3_hands_to_mouth', 'm3_coos_vowels', 'm3_tracks_objects', 'm3_hands_together', 'm4_laughs', 'm4_holds_head_steady', 'm4_holds_toy', 'm4_pushes_up_elbows', 'm4_turns_to_voice', 'm6_knows_familiar', 'm6_takes_turns_sounds', 'm6_rolls_over', 'm6_reaches_toy', 'm6_sits_with_support', 'm9_stranger_aware', 'm9_responds_to_name', 'm9_sits_alone', 'm9_passes_hands', 'm9_babbles_syllables', 'm12_waves_bye', 'm12_says_mama_dada', 'm12_pulls_to_stand', 'm12_cruises', 'm12_pincer_grasp', 'm18_walks_alone', 'm18_points_to_show', 'm18_says_words', 'm18_uses_spoon', 'm18_scribbles', 'm24_two_word_phrases', 'm24_kicks_ball', 'm24_runs', 'm24_points_in_book', 'm24_notices_feelings', 'm36_converses', 'm36_says_own_name', 'm36_draws_circle', 'm36_dresses_partly', 'm36_plays_with_others', 'm48_tells_story', 'm48_names_colours', 'm48_pretend_play', 'm48_catches_ball', 'm48_comforts_others', 'm60_counts_to_ten', 'm60_hops_one_foot', 'm60_writes_letters', 'm60_follows_rules', 'm60_buttons'])->delete();
        DB::table('catalog_items')->where('group', 'child_milestone_activities')
            ->whereIn('code', ['m2_face_time', 'm2_tummy_time', 'm3_tummy_time', 'm3_sound_talk', 'm4_rattle_reach', 'm4_mirror_play', 'm6_peekaboo', 'm6_supported_sitting', 'm9_name_things', 'm9_crawl_course', 'm12_stack_cups', 'm12_read_together', 'm18_simple_tasks', 'm24_pretend_kitchen', 'm36_story_turns', 'm48_colour_hunt', 'm60_board_games'])->delete();
        DB::table('catalog_items')->where('group', 'child_milestone_notes')
            ->whereIn('code', ['m2', 'm3', 'm4', 'm6', 'm9', 'm12', 'm18', 'm24', 'm36', 'm48', 'm60'])->delete();
        DB::table('catalog_items')->where('group', 'child_age_notes')
            ->whereIn('code', ['m0', 'm2', 'm3', 'm4', 'm6', 'm9', 'm12', 'm18', 'm24', 'm36', 'm48', 'm60'])->delete();
        DB::table('catalog_items')->where('group', 'child_learn')
            ->whereIn('code', ['newborn_safe_sleep', 'night_sleep_3m', 'toddler_sleep_routine', 'breastfeeding_supply', 'starting_solids', 'picky_eating', 'evening_crying', 'vaccine_4m', 'teething', 'neck_play', 'crawling_play', 'toddler_talking', 'mother_self_care', 'mother_return_to_routine'])->delete();
        Schema::table('family_children', function (Blueprint $table) {
            $table->dropForeign(['child_id']);
        });
        Schema::dropIfExists('child_milestone_checks');
        Schema::dropIfExists('child_vaccine_doses');
        Schema::dropIfExists('child_measurements');
        Schema::dropIfExists('children');
    }
};

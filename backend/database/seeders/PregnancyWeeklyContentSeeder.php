<?php

namespace Database\Seeders;

use App\Models\PregnancyWeeklyContent;
use Illuminate\Database\Seeder;

class PregnancyWeeklyContentSeeder extends Seeder
{
    /**
     * Seed the pregnancy_weekly_content table for all 40 weeks.
     * Idempotent: uses updateOrCreate keyed on week_number.
     */
    public function run(): void
    {
        foreach ($this->weeks() as $week => $content) {
            PregnancyWeeklyContent::updateOrCreate(
                ['week_number' => $week],
                $content
            );
        }
    }

    /** Bilingual string module helper. */
    private function bi(string $fa, string $en): array
    {
        return ['fa' => $fa, 'en' => $en];
    }

    /**
     * FAQ module helper.
     *
     * @param  array<int, array{0:string,1:string}>  $fa  list of [question, answer]
     * @param  array<int, array{0:string,1:string}>  $en  list of [question, answer]
     */
    private function faq(array $fa, array $en): array
    {
        $map = fn (array $pairs) => array_map(
            fn (array $p) => ['question' => $p[0], 'answer' => $p[1]],
            $pairs
        );

        return ['fa' => $map($fa), 'en' => $map($en)];
    }

    /**
     * All weekly content keyed by week number (1-40).
     *
     * @return array<int, array<string, array>>
     */
    private function weeks(): array
    {
        return $this->weeks01to10()
            + $this->weeks11to20()
            + $this->weeks21to30()
            + $this->weeks31to40();
    }

    /** @return array<int, array<string, array>> */
    private function weeks01to10(): array
    {
        return [
            1 => [
                'fetal_development' => $this->bi(
                    'در هفته اول بارداری هنوز جنینی شکل نگرفته است؛ این هفته بر اساس اولین روز آخرین قاعدگی شمرده می‌شود. بدن شما در حال آماده‌سازی برای تخمک‌گذاری و لانه‌گزینی احتمالی است. در واقع لقاح هنوز اتفاق نیفتاده و اندازه‌ای برای جنین قابل ذکر نیست.',
                    'In week one there is no embryo yet; this week is counted from the first day of your last period. Your body is preparing for ovulation and possible implantation.'
                ),
                'mother_body_changes' => $this->bi(
                    'در این هفته بدن شما در حال دفع پوشش رحم قبلی و آماده‌سازی برای چرخه جدید است. ممکن است علائم معمول قاعدگی را تجربه کنید. تغییر محسوس بارداری هنوز رخ نداده است.',
                    'Your body is shedding the previous uterine lining and preparing a new cycle. You may notice typical period symptoms; no pregnancy signs yet.'
                ),
                'dos_and_donts' => $this->bi(
                    'مصرف اسید فولیک روزانه را از همین حالا شروع کنید و از مصرف الکل و دخانیات پرهیز نمایید. یک رژیم متعادل و خواب کافی داشته باشید. پیش از مصرف هر دارویی با پزشک مشورت کنید.',
                    'Start a daily folic acid supplement now and avoid alcohol and smoking. Eat a balanced diet, sleep well, and check any medication with your doctor.'
                ),
                'care_plan' => $this->bi(
                    'اگر قصد بارداری دارید، این هفته زمان خوبی برای شروع مکمل اسید فولیک و ثبت تاریخ قاعدگی است. یک معاینه پیش از بارداری می‌تواند مفید باشد. سبک زندگی سالم را پایه‌ریزی کنید.',
                    'If planning pregnancy, this is a good week to begin folic acid and track your period date. A pre-conception checkup can help.'
                ),
                'body_adaptation' => $this->bi(
                    'بدن شما هنوز در وضعیت عادی قرار دارد و تطابق بارداری آغاز نشده است. تمرکز بر آمادگی جسمی و روانی برای بارداری احتمالی مفید است.',
                    'Your body is still in its normal state and pregnancy adaptation has not begun. Focus on physical and mental readiness.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است ترکیبی از هیجان و نگرانی برای شروع مسیر بارداری داشته باشید. آرامش و انتظارات واقع‌بینانه به شما کمک می‌کند. صحبت با همسر می‌تواند احساس اطمینان بیشتری بدهد.',
                    'You may feel a mix of excitement and worry about the journey ahead. Staying calm and realistic helps; talking with your partner can reassure you.'
                ),
                'key_nutrition' => $this->bi(
                    'روزانه ۴۰۰ میکروگرم اسید فولیک، سبزیجات برگ‌سبز و غلات کامل را در برنامه بگنجانید. آب کافی بنوشید و مصرف کافئین را محدود کنید. تغذیه سالم پیش از بارداری پایه رشد سالم جنین است.',
                    'Include 400 mcg folic acid daily, leafy greens and whole grains. Drink enough water and limit caffeine.'
                ),
                'physical_activity' => $this->bi(
                    'فعالیت بدنی سبک مانند پیاده‌روی و کشش ملایم را ادامه دهید. ورزش منظم آمادگی بدن را برای بارداری بهتر می‌کند. از فعالیت‌های بسیار شدید بدون مشورت پرهیز کنید.',
                    'Keep up light activity like walking and gentle stretching to prepare your body. Avoid very intense exercise without advice.'
                ),
                'tests_and_checkups' => $this->bi(
                    'اگر قصد بارداری دارید، یک ویزیت پیش از بارداری برای بررسی سلامت عمومی و واکسیناسیون مفید است. آزمایش خاص بارداری در این هفته لازم نیست. سابقه پزشکی خود را با پزشک مرور کنید.',
                    'A pre-conception visit to review general health and vaccinations is helpful. No pregnancy-specific test is needed this week.'
                ),
                'faq' => $this->faq(
                    [
                        ['چرا هفته اول بدون وجود جنین شمرده می‌شود؟', 'محاسبه سن بارداری از اولین روز آخرین قاعدگی آغاز می‌شود، بنابراین دو هفته اول پیش از لقاح در نظر گرفته می‌شود.'],
                        ['چه زمانی باید اسید فولیک را شروع کنم؟', 'بهتر است مصرف اسید فولیک را حتی پیش از بارداری و از همین حالا آغاز کنید تا از رشد سالم سیستم عصبی جنین حمایت شود.'],
                    ],
                    [
                        ['Why is week one counted without an embryo?', 'Gestational age is measured from the first day of your last period, so the first two weeks come before conception.'],
                        ['When should I start folic acid?', 'It is best to begin folic acid even before conception to support healthy neural development.'],
                    ]
                ),
            ],
            2 => [
                'fetal_development' => $this->bi(
                    'در هفته دوم نیز جنینی وجود ندارد و بدن در حال آماده‌سازی برای تخمک‌گذاری است. تخمک بالغ در آستانه آزاد شدن از تخمدان قرار دارد. لقاح معمولاً در پایان این هفته ممکن می‌شود.',
                    'There is still no embryo in week two; your body is preparing for ovulation. A mature egg is about to be released, and conception may occur near the end of this week.'
                ),
                'mother_body_changes' => $this->bi(
                    'پوشش داخلی رحم در حال ضخیم شدن است تا بستری مناسب برای لانه‌گزینی فراهم شود. ممکن است نشانه‌های تخمک‌گذاری مانند تغییر ترشحات را متوجه شوید. سطح هورمون‌ها رو به افزایش است.',
                    'The uterine lining is thickening to prepare for implantation. You may notice ovulation signs such as changes in discharge as hormone levels rise.'
                ),
                'dos_and_donts' => $this->bi(
                    'به مصرف اسید فولیک ادامه دهید و از استرس زیاد پرهیز کنید. تغذیه سالم و خواب منظم را حفظ کنید. از مصرف الکل و سیگار به‌طور کامل دوری کنید.',
                    'Continue folic acid and avoid excess stress. Keep a healthy diet and regular sleep, and fully avoid alcohol and smoking.'
                ),
                'care_plan' => $this->bi(
                    'اگر در تلاش برای بارداری هستید، این هفته احتمال تخمک‌گذاری بالاست و زمان مناسبی برای باروری است. پیگیری علائم تخمک‌گذاری می‌تواند کمک‌کننده باشد. سبک زندگی سالم را ادامه دهید.',
                    'If trying to conceive, ovulation is likely this week, making it a fertile window. Tracking ovulation signs can help.'
                ),
                'body_adaptation' => $this->bi(
                    'بدن شما با ترشح هورمون‌های تخمک‌گذاری خود را برای لقاح احتمالی آماده می‌کند. این تغییرات طبیعی و بخشی از چرخه باروری است.',
                    'Your body prepares for possible conception through ovulation hormones. These changes are a natural part of the fertility cycle.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است در این دوره انتظار، احساس امید یا اضطراب داشته باشید. حفظ آرامش و پرهیز از فشار روانی برای این مسیر مهم است. با خودتان مهربان باشید.',
                    'During this waiting phase you may feel hopeful or anxious. Staying calm and easing pressure on yourself matters; be kind to yourself.'
                ),
                'key_nutrition' => $this->bi(
                    'مصرف اسید فولیک، آهن و پروتئین کافی را ادامه دهید. میوه و سبزیجات تازه و آب فراوان بخش مهم رژیم است. از غذاهای فرآوری‌شده کمتر استفاده کنید.',
                    'Continue folic acid, iron and enough protein. Fresh fruit, vegetables and plenty of water are key; limit processed foods.'
                ),
                'physical_activity' => $this->bi(
                    'ورزش سبک تا متوسط مانند پیاده‌روی و یوگا مفید است و به تعادل هورمونی کمک می‌کند. از فعالیت‌های پرخطر بپرهیزید. به بدن خود گوش دهید.',
                    'Light to moderate exercise like walking and yoga supports hormonal balance. Avoid high-risk activities and listen to your body.'
                ),
                'tests_and_checkups' => $this->bi(
                    'در این هفته آزمایش خاصی لازم نیست، اما کیت تخمک‌گذاری می‌تواند به تشخیص زمان باروری کمک کند. در صورت داشتن بیماری زمینه‌ای با پزشک مشورت کنید.',
                    'No specific test is needed, but an ovulation kit can help identify your fertile window. Consult your doctor about any underlying condition.'
                ),
                'faq' => $this->faq(
                    [
                        ['بهترین زمان برای باروری کدام است؟', 'روزهای نزدیک به تخمک‌گذاری، یعنی اواسط چرخه، بیشترین احتمال باروری را دارند.'],
                        ['آیا تست بارداری در این هفته معتبر است؟', 'خیر، چون لقاح هنوز تثبیت نشده است و تست بارداری در این مرحله نتیجه دقیقی نمی‌دهد.'],
                    ],
                    [
                        ['When is the most fertile time?', 'The days around ovulation, in the middle of your cycle, offer the highest chance of conception.'],
                        ['Is a pregnancy test reliable this week?', 'No, conception has not yet been established, so a test now will not be accurate.'],
                    ]
                ),
            ],
            3 => [
                'fetal_development' => $this->bi(
                    'در هفته سوم لقاح انجام می‌شود و اسپرم و تخمک به هم می‌پیوندند تا یک سلول واحد به نام زیگوت بسازند. این سلول به سرعت تقسیم می‌شود و به سمت رحم حرکت می‌کند. هنوز اندازه قابل مشاهده‌ای وجود ندارد و همه چیز در سطح میکروسکوپی است.',
                    'Fertilization happens in week three as sperm and egg unite to form a single cell, the zygote. It divides rapidly while traveling to the uterus; everything is still microscopic.'
                ),
                'mother_body_changes' => $this->bi(
                    'زیگوت در حال حرکت به سمت رحم برای لانه‌گزینی است و بدن هنوز تغییر محسوسی نشان نمی‌دهد. برخی زنان ممکن است لکه‌بینی خفیف لانه‌گزینی را تجربه کنند. سطح هورمون‌ها به‌تدریج در حال تغییر است.',
                    'The zygote is moving toward the uterus to implant, with few visible changes yet. Some women notice light implantation spotting as hormones gradually shift.'
                ),
                'dos_and_donts' => $this->bi(
                    'مصرف اسید فولیک و تغذیه سالم را جدی بگیرید و از داروهای بدون تجویز پرهیز کنید. از قرار گرفتن در معرض مواد شیمیایی و اشعه دوری کنید. استراحت کافی داشته باشید.',
                    'Take folic acid and healthy nutrition seriously and avoid over-the-counter drugs. Stay away from chemicals and radiation, and rest well.'
                ),
                'care_plan' => $this->bi(
                    'در این هفته حساس، حفظ سبک زندگی سالم اهمیت زیادی دارد چون لانه‌گزینی در حال شکل‌گیری است. از عوامل مضر دوری کنید و مکمل‌ها را منظم مصرف کنید.',
                    'In this sensitive week, a healthy lifestyle matters greatly as implantation begins. Avoid harmful factors and take supplements regularly.'
                ),
                'body_adaptation' => $this->bi(
                    'بدن شما در حال آماده کردن رحم برای پذیرش جنین است و به‌زودی تولید هورمون بارداری آغاز می‌شود. این تطابق اولیه پایه یک بارداری سالم است.',
                    'Your body is preparing the uterus to receive the embryo, and pregnancy hormone production will soon begin. This early adaptation lays the groundwork.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است هنوز از بارداری آگاه نباشید و احساسات معمول روزمره را داشته باشید. اگر در تلاش برای بارداری هستید، امید و انتظار طبیعی است. آرامش خود را حفظ کنید.',
                    'You may not yet know you are pregnant and feel your usual daily emotions. If trying to conceive, hope and anticipation are natural.'
                ),
                'key_nutrition' => $this->bi(
                    'اسید فولیک، امگا-۳ و پروتئین را در اولویت قرار دهید تا از تقسیم سلولی سالم حمایت شود. مصرف غذاهای تازه و آب کافی را ادامه دهید. از کافئین زیاد بپرهیزید.',
                    'Prioritize folic acid, omega-3 and protein to support healthy cell division. Keep eating fresh foods and drinking enough water; limit caffeine.'
                ),
                'physical_activity' => $this->bi(
                    'فعالیت ملایم مانند پیاده‌روی روزانه کافی است و به گردش خون کمک می‌کند. از تمرینات سنگین و ضربه‌ای پرهیز کنید. به علائم بدن توجه کنید.',
                    'Gentle activity such as daily walking is enough and supports circulation. Avoid heavy or high-impact workouts and heed your body.'
                ),
                'tests_and_checkups' => $this->bi(
                    'آزمایش بارداری هنوز نتیجه قابل اعتمادی نمی‌دهد و بهتر است چند روز صبر کنید. در صورت وجود سابقه سقط یا بیماری خاص، با پزشک در میان بگذارید.',
                    'A pregnancy test is not yet reliable, so it is better to wait a few days. Discuss any history of miscarriage or specific condition with your doctor.'
                ),
                'faq' => $this->faq(
                    [
                        ['لکه‌بینی لانه‌گزینی طبیعی است؟', 'بله، لکه‌بینی خفیف هنگام لانه‌گزینی می‌تواند طبیعی باشد، اما خونریزی شدید باید با پزشک بررسی شود.'],
                        ['چه زمانی می‌توانم تست بارداری بدهم؟', 'معمولاً حدود یک هفته پس از لقاح یا بعد از عقب افتادن قاعدگی تست دقیق‌تر خواهد بود.'],
                    ],
                    [
                        ['Is implantation spotting normal?', 'Yes, light spotting at implantation can be normal, but heavy bleeding should be checked by a doctor.'],
                        ['When can I take a pregnancy test?', 'A test is usually more accurate about a week after conception or after a missed period.'],
                    ]
                ),
            ],
            4 => [
                'fetal_development' => $this->bi(
                    'در هفته چهارم جنین به اندازه یک دانه خشخاش و حدود ۲ میلی‌متر است. لانه‌گزینی در دیواره رحم کامل می‌شود و لایه‌های سلولی که اندام‌ها را می‌سازند شکل می‌گیرند. جفت و کیسه آمنیوتیک نیز در حال تشکیل هستند.',
                    'At week four the embryo is about the size of a poppy seed, roughly 2 mm. Implantation completes and the cell layers that form organs begin to develop, along with the placenta and amniotic sac.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است اولین نشانه‌ها مانند خستگی، حساسیت سینه و عقب افتادن قاعدگی را تجربه کنید. بدن شروع به تولید هورمون بارداری (hCG) کرده است. برخی زنان هنوز هیچ علامتی ندارند.',
                    'You may notice first signs such as fatigue, breast tenderness and a missed period. Your body has started producing the pregnancy hormone hCG, though some women feel nothing yet.'
                ),
                'dos_and_donts' => $this->bi(
                    'مصرف اسید فولیک را ادامه دهید و از الکل، سیگار و داروهای غیرمجاز کاملاً دوری کنید. تغذیه سالم و استراحت را در اولویت بگذارید. از بلند کردن اجسام سنگین خودداری کنید.',
                    'Continue folic acid and completely avoid alcohol, smoking and unapproved drugs. Prioritize healthy nutrition and rest, and avoid lifting heavy objects.'
                ),
                'care_plan' => $this->bi(
                    'اگر تست بارداری مثبت شد، اولین ویزیت بارداری را برنامه‌ریزی کنید. مصرف مکمل‌های تجویزشده را آغاز کنید و سبک زندگی سالم را حفظ نمایید. علائم غیرعادی را یادداشت کنید.',
                    'If your test is positive, schedule your first prenatal visit. Begin prescribed supplements and maintain a healthy lifestyle, noting any unusual symptoms.'
                ),
                'body_adaptation' => $this->bi(
                    'بدن شما با افزایش هورمون‌های بارداری در حال تطابق با وضعیت جدید است. ممکن است تغییرات خلقی و جسمی خفیف را حس کنید. این روند کاملاً طبیعی است.',
                    'Your body is adapting to its new state with rising pregnancy hormones. You may sense mild mood and physical changes, which are entirely normal.'
                ),
                'emotional_status' => $this->bi(
                    'آگاهی از بارداری می‌تواند شادی، هیجان یا اضطراب به همراه داشته باشد. نوسانات خلقی به دلیل تغییرات هورمونی طبیعی است. حمایت اطرافیان به آرامش شما کمک می‌کند.',
                    'Learning you are pregnant can bring joy, excitement or anxiety. Mood swings from hormonal changes are normal, and support from loved ones helps.'
                ),
                'key_nutrition' => $this->bi(
                    'اسید فولیک، آهن و کلسیم را در برنامه غذایی حفظ کنید. وعده‌های کوچک و مکرر به کاهش تهوع احتمالی کمک می‌کند. آب کافی بنوشید و از غذاهای خام و نپخته پرهیز کنید.',
                    'Keep folic acid, iron and calcium in your diet. Small frequent meals help ease possible nausea; drink enough water and avoid raw or undercooked foods.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی و حرکات کششی ملایم برای این هفته مناسب است. از ورزش‌های شدید و پرخطر خودداری کنید. اگر خستگی دارید به بدن استراحت بدهید.',
                    'Walking and gentle stretching suit this week. Avoid intense or risky sports, and rest when you feel tired.'
                ),
                'tests_and_checkups' => $this->bi(
                    'تست بارداری خانگی یا آزمایش خون hCG می‌تواند بارداری را تأیید کند. با تأیید بارداری، برای اولین ملاقات پیش از تولد وقت بگیرید. فشار خون و سلامت عمومی بررسی می‌شود.',
                    'A home test or blood hCG test can confirm pregnancy. Once confirmed, book your first prenatal visit where blood pressure and general health are checked.'
                ),
                'faq' => $this->faq(
                    [
                        ['علائم اولیه بارداری چیست؟', 'خستگی، حساسیت سینه، تهوع خفیف و عقب افتادن قاعدگی از علائم شایع اولیه هستند.'],
                        ['اولین ویزیت بارداری چه زمانی است؟', 'معمولاً پزشکان اولین ملاقات را حدود هفته ۶ تا ۸ توصیه می‌کنند، اما با تأیید بارداری می‌توانید وقت بگیرید.'],
                    ],
                    [
                        ['What are early pregnancy symptoms?', 'Fatigue, breast tenderness, mild nausea and a missed period are common early signs.'],
                        ['When is the first prenatal visit?', 'Doctors usually suggest around weeks 6 to 8, but you can book once pregnancy is confirmed.'],
                    ]
                ),
            ],
            5 => [
                'fetal_development' => $this->bi(
                    'در هفته پنجم جنین به اندازه یک دانه کنجد و حدود ۳ میلی‌متر است. لوله عصبی که مغز و نخاع را می‌سازد در حال شکل‌گیری است و قلب کوچک شروع به تپیدن می‌کند. رشد بسیار سریع اندام‌های پایه آغاز شده است.',
                    'At week five the embryo is about the size of a sesame seed, roughly 3 mm. The neural tube forming the brain and spinal cord is developing and the tiny heart begins to beat.'
                ),
                'mother_body_changes' => $this->bi(
                    'علائمی مانند تهوع صبحگاهی، خستگی و تکرر ادرار ممکن است شروع شود. سینه‌ها حساس‌تر و سنگین‌تر می‌شوند. نوسانات خلقی به دلیل هورمون‌ها شایع است.',
                    'Symptoms such as morning sickness, fatigue and frequent urination may begin. Breasts become more tender and heavy, and mood swings from hormones are common.'
                ),
                'dos_and_donts' => $this->bi(
                    'مصرف مکمل‌های بارداری را منظم ادامه دهید و از غذاهای پرخطر مانند گوشت خام و پنیر غیرپاستوریزه بپرهیزید. استراحت کافی داشته باشید و از استرس دوری کنید. داروها را فقط با تجویز پزشک مصرف کنید.',
                    'Continue prenatal supplements and avoid risky foods like raw meat and unpasteurized cheese. Rest well, reduce stress, and take medicines only as prescribed.'
                ),
                'care_plan' => $this->bi(
                    'اگر هنوز به پزشک مراجعه نکرده‌اید، این هفته زمان مناسبی برای اولین ویزیت است. برای مدیریت تهوع، وعده‌های کوچک و مکرر بخورید. علائم هشدار مانند خونریزی را جدی بگیرید.',
                    'If you have not yet seen a doctor, this is a good week for your first visit. Manage nausea with small frequent meals and take warning signs like bleeding seriously.'
                ),
                'body_adaptation' => $this->bi(
                    'حجم خون بدن شما در حال افزایش است تا به جنین در حال رشد خون‌رسانی کند. این تغییر می‌تواند باعث احساس خستگی و سرگیجه شود. بدن به‌تدریج با بارداری هماهنگ می‌شود.',
                    'Your blood volume is increasing to nourish the growing embryo, which can cause fatigue and dizziness. Your body is gradually syncing with pregnancy.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است بین هیجان بارداری و نگرانی درباره سلامت جنین در نوسان باشید. این احساسات کاملاً طبیعی است. صحبت با همسر یا یک دوست می‌تواند آرامش‌بخش باشد.',
                    'You may swing between excitement and worry about the baby’s health. These feelings are entirely normal; talking with your partner or a friend can soothe you.'
                ),
                'key_nutrition' => $this->bi(
                    'اسید فولیک، ویتامین B6 و مایعات کافی می‌تواند به کاهش تهوع کمک کند. زنجبیل و کراکر ساده گزینه‌های خوبی برای صبح هستند. از غذاهای چرب و تند پرهیز کنید.',
                    'Folic acid, vitamin B6 and enough fluids can ease nausea. Ginger and plain crackers are good morning options; avoid greasy and spicy foods.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی سبک و تمرینات تنفسی به بهبود حال عمومی کمک می‌کند. اگر خسته هستید، فعالیت را کاهش دهید و استراحت کنید. از ورزش‌های شدید بپرهیزید.',
                    'Light walking and breathing exercises improve overall wellbeing. If tired, reduce activity and rest; avoid intense workouts.'
                ),
                'tests_and_checkups' => $this->bi(
                    'در اولین ویزیت، آزمایش خون، گروه خونی، Rh و بررسی عفونت‌ها انجام می‌شود. سونوگرافی اولیه ممکن است برای تأیید محل بارداری توصیه شود. فشار خون و وزن ثبت می‌گردد.',
                    'At your first visit, blood tests, blood group, Rh and infection screening are done. An early ultrasound may be advised to confirm the pregnancy location.'
                ),
                'faq' => $this->faq(
                    [
                        ['تهوع صبحگاهی چقدر طول می‌کشد؟', 'معمولاً تهوع از هفته ۵ شروع و تا حدود هفته ۱۲ تا ۱۴ کاهش می‌یابد، اما در هر فرد متفاوت است.'],
                        ['آیا تهوع نشانه سلامت بارداری است؟', 'تهوع شایع است اما نبود آن هم نگران‌کننده نیست؛ در صورت تهوع شدید و استفراغ مکرر با پزشک مشورت کنید.'],
                    ],
                    [
                        ['How long does morning sickness last?', 'Nausea usually starts around week five and eases by weeks 12 to 14, though it varies for each person.'],
                        ['Is nausea a sign of a healthy pregnancy?', 'Nausea is common, but its absence is not worrying; see a doctor if vomiting is severe.'],
                    ]
                ),
            ],
            6 => [
                'fetal_development' => $this->bi(
                    'در هفته ششم جنین به اندازه یک عدس و حدود ۵ میلی‌متر است. ضربان قلب کوچک اکنون قابل تشخیص است و جوانه‌های دست و پا در حال ظاهر شدن هستند. چهره و ساختار مغز نیز به‌تدریج شکل می‌گیرد.',
                    'At week six the embryo is about the size of a lentil, roughly 5 mm. The tiny heartbeat is now detectable and limb buds are appearing, while the face and brain structures gradually take shape.'
                ),
                'mother_body_changes' => $this->bi(
                    'تهوع، خستگی و حساسیت به بوها ممکن است شدیدتر شود. تکرر ادرار و تغییرات خلقی ادامه دارد. برخی زنان افزایش بزاق یا تغییر در ذائقه را تجربه می‌کنند.',
                    'Nausea, fatigue and sensitivity to smells may intensify. Frequent urination and mood changes continue, and some women notice more saliva or altered taste.'
                ),
                'dos_and_donts' => $this->bi(
                    'به مصرف مکمل‌ها ادامه دهید و از خوددرمانی برای تهوع پرهیز کنید. آب کافی بنوشید و از گرسنگی طولانی خودداری کنید. در صورت خونریزی یا درد شدید فوراً با پزشک تماس بگیرید.',
                    'Continue supplements and avoid self-medicating for nausea. Drink enough water and don’t stay hungry long; contact your doctor immediately for bleeding or severe pain.'
                ),
                'care_plan' => $this->bi(
                    'اگر ویزیت اول را انجام نداده‌اید، همین هفته اقدام کنید. برای مدیریت خستگی، استراحت بیشتری در برنامه بگنجانید. علائم بارداری را یادداشت کنید تا با پزشک در میان بگذارید.',
                    'If you haven’t had your first visit, arrange it this week. Build in more rest to manage fatigue and note your symptoms to share with your doctor.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم در حال بزرگ شدن است و ممکن است احساس سنگینی خفیف در ناحیه لگن داشته باشید. سیستم گردش خون برای پشتیبانی از جنین سازگار می‌شود. این تغییرات بخشی از روند طبیعی است.',
                    'The uterus is enlarging and you may feel mild heaviness in the pelvis. Your circulation adapts to support the baby, all part of the natural process.'
                ),
                'emotional_status' => $this->bi(
                    'نوسانات خلقی و حساسیت عاطفی در این هفته شایع است. ممکن است زودتر خسته یا زودرنج شوید. استراحت و گفتگو درباره احساساتتان کمک‌کننده است.',
                    'Mood swings and emotional sensitivity are common this week. You may tire or feel irritable more easily; rest and sharing your feelings help.'
                ),
                'key_nutrition' => $this->bi(
                    'وعده‌های کوچک و مکرر همراه با پروتئین و کربوهیدرات پیچیده به کنترل تهوع کمک می‌کند. زنجبیل، لیمو و مایعات خنک مفید هستند. مصرف آهن و اسید فولیک را ادامه دهید.',
                    'Small frequent meals with protein and complex carbs help control nausea. Ginger, lemon and cool fluids can help; keep taking iron and folic acid.'
                ),
                'physical_activity' => $this->bi(
                    'اگر توان دارید، پیاده‌روی کوتاه و کشش ملایم مفید است. در روزهای پرتهوع، استراحت را در اولویت بگذارید. به سیگنال‌های بدن خود احترام بگذارید.',
                    'If you have energy, short walks and gentle stretching help. On nauseous days, prioritize rest and respect your body’s signals.'
                ),
                'tests_and_checkups' => $this->bi(
                    'سونوگرافی اوایل بارداری ممکن است برای تأیید ضربان قلب جنین و سن بارداری انجام شود. آزمایش‌های خون پایه در صورت انجام نشدن تکمیل می‌گردد. فشار خون و وزن پیگیری می‌شود.',
                    'An early ultrasound may be done to confirm the fetal heartbeat and gestational age. Baseline blood tests are completed if not done yet.'
                ),
                'faq' => $this->faq(
                    [
                        ['چه زمانی ضربان قلب جنین شنیده می‌شود؟', 'ضربان قلب معمولاً از هفته ۶ در سونوگرافی قابل مشاهده است، هرچند گاهی کمی دیرتر دیده می‌شود.'],
                        ['برای تهوع شدید چه کنم؟', 'وعده‌های کوچک، مایعات کافی و زنجبیل کمک می‌کند؛ اگر استفراغ مانع نوشیدن آب شد، حتماً به پزشک مراجعه کنید.'],
                    ],
                    [
                        ['When is the fetal heartbeat detectable?', 'A heartbeat is usually visible on ultrasound from week six, though sometimes slightly later.'],
                        ['What should I do for severe nausea?', 'Small meals, enough fluids and ginger help; if vomiting stops you from drinking, see your doctor.'],
                    ]
                ),
            ],
            7 => [
                'fetal_development' => $this->bi(
                    'در هفته هفتم جنین به اندازه یک بلوبری و حدود ۱ سانتی‌متر است. سر نسبت به بدن بزرگ‌تر است و مغز به سرعت رشد می‌کند. دست‌ها و پاها بیشتر شکل می‌گیرند و کلیه‌ها شروع به کار می‌کنند.',
                    'At week seven the embryo is about the size of a blueberry, roughly 1 cm. The head is large relative to the body as the brain grows rapidly, and arms, legs and kidneys develop further.'
                ),
                'mother_body_changes' => $this->bi(
                    'تهوع و خستگی همچنان ادامه دارد و ممکن است حساسیت سینه بیشتر شود. برخی زنان لکه‌بینی خفیف یا نفخ را تجربه می‌کنند. تغییرات پوستی خفیف نیز ممکن است ظاهر شود.',
                    'Nausea and fatigue continue and breast tenderness may increase. Some women notice light spotting or bloating, and mild skin changes can appear.'
                ),
                'dos_and_donts' => $this->bi(
                    'به تغذیه سالم و مصرف مکمل‌ها پایبند بمانید و از غذاهای خام و کافئین زیاد بپرهیزید. استراحت کافی داشته باشید و از فعالیت سنگین خودداری کنید. علائم غیرعادی را با پزشک مطرح کنید.',
                    'Stick to healthy eating and supplements, and avoid raw foods and excess caffeine. Rest enough, avoid heavy activity, and report unusual symptoms.'
                ),
                'care_plan' => $this->bi(
                    'برنامه ویزیت‌های منظم بارداری را دنبال کنید و سوالات خود را یادداشت کنید. برای مدیریت تهوع و خستگی برنامه‌ریزی داشته باشید. مصرف آب و استراحت را جدی بگیرید.',
                    'Keep up regular prenatal visits and note your questions. Plan to manage nausea and fatigue, and take hydration and rest seriously.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم به اندازه دو برابر حالت عادی بزرگ شده و ممکن است کمی احساس فشار در شکم کنید. بدن در حال تولید بیشتر خون و مایعات است. این سازگاری‌ها طبیعی هستند.',
                    'The uterus has roughly doubled in size and you may feel mild abdominal pressure. Your body is producing more blood and fluids as it adapts.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است احساسات شدید و نوسانات خلقی داشته باشید که ناشی از هورمون‌هاست. کمی نگرانی درباره بارداری طبیعی است. مراقبت از سلامت روان به اندازه جسم مهم است.',
                    'You may feel intense emotions and mood swings driven by hormones. Some worry about the pregnancy is normal; mental care matters as much as physical.'
                ),
                'key_nutrition' => $this->bi(
                    'مصرف پروتئین، غلات کامل و میوه‌ها را در طول روز پخش کنید تا انرژی پایدار داشته باشید. آهن و فولات همچنان مهم هستند. مایعات کافی برای جلوگیری از کم‌آبی بنوشید.',
                    'Spread protein, whole grains and fruit through the day for steady energy. Iron and folate remain important; drink enough fluids to prevent dehydration.'
                ),
                'physical_activity' => $this->bi(
                    'ورزش ملایم مانند پیاده‌روی و یوگای بارداری به کاهش خستگی کمک می‌کند. از حرکات پرفشار و بلند کردن اجسام سنگین بپرهیزید. در صورت سرگیجه فعالیت را متوقف کنید.',
                    'Gentle exercise like walking and prenatal yoga eases fatigue. Avoid high-impact moves and heavy lifting, and stop if you feel dizzy.'
                ),
                'tests_and_checkups' => $this->bi(
                    'در این هفته معمولاً پیگیری آزمایش‌های اولیه و ویزیت پزشک انجام می‌شود. در صورت وجود ریسک، پزشک ممکن است سونوگرافی توصیه کند. علائم هشدار را گزارش دهید.',
                    'This week usually involves follow-up on initial tests and a doctor visit. If there is risk, an ultrasound may be advised; report any warning signs.'
                ),
                'faq' => $this->faq(
                    [
                        ['نفخ و لکه‌بینی خفیف طبیعی است؟', 'نفخ شایع است و لکه‌بینی خیلی خفیف می‌تواند طبیعی باشد، اما خونریزی همراه با درد باید بررسی شود.'],
                        ['چطور با خستگی کنار بیایم؟', 'استراحت کوتاه در طول روز، خواب کافی شبانه و تغذیه متعادل به مدیریت خستگی کمک می‌کند.'],
                    ],
                    [
                        ['Are bloating and light spotting normal?', 'Bloating is common and very light spotting can be normal, but bleeding with pain should be checked.'],
                        ['How can I cope with fatigue?', 'Short daytime rests, enough night sleep and balanced meals help manage tiredness.'],
                    ]
                ),
            ],
            8 => [
                'fetal_development' => $this->bi(
                    'در هفته هشتم جنین به اندازه یک لوبیا قرمز، حدود ۱.۶ سانتی‌متر و کمتر از ۱ گرم است. انگشتان دست و پا در حال شکل‌گیری‌اند و پلک‌ها و لب بالایی نمایان می‌شوند. حرکات کوچک جنین آغاز شده اما هنوز حس نمی‌شود.',
                    'At week eight the fetus is about the size of a kidney bean, roughly 1.6 cm and under 1 g. Fingers and toes are forming, eyelids and the upper lip appear, and tiny movements begin though unfelt.'
                ),
                'mother_body_changes' => $this->bi(
                    'تهوع، خستگی و حساسیت سینه ممکن است به اوج خود برسد. کمر شلوارها ممکن است کمی تنگ‌تر شود، هرچند شکم هنوز برجسته نیست. تغییرات خلقی همچنان ادامه دارد.',
                    'Nausea, fatigue and breast tenderness may peak. Your waistband may feel tighter although the belly isn’t showing yet, and mood changes continue.'
                ),
                'dos_and_donts' => $this->bi(
                    'مصرف مکمل‌ها و تغذیه سالم را حفظ کنید و از الکل، سیگار و کافئین زیاد پرهیز کنید. برای تهوع دارو خودسرانه مصرف نکنید. خواب و استراحت کافی داشته باشید.',
                    'Maintain supplements and healthy eating, and avoid alcohol, smoking and excess caffeine. Don’t self-medicate for nausea, and get enough sleep and rest.'
                ),
                'care_plan' => $this->bi(
                    'اگر هنوز اولین سونوگرافی را انجام نداده‌اید، این دوره زمان مناسبی برای تأیید بارداری و سن جنین است. سوالات خود را برای ویزیت آماده کنید. مصرف مایعات را جدی بگیرید.',
                    'If you haven’t had your first ultrasound, this is a good time to confirm the pregnancy and gestational age. Prepare questions for your visit and stay hydrated.'
                ),
                'body_adaptation' => $this->bi(
                    'حجم خون و ضربان قلب شما افزایش یافته تا نیاز جنین را تأمین کند. ممکن است احساس گرمازدگی یا سرگیجه گاه‌به‌گاه داشته باشید. بدن به‌آرامی با بارداری هماهنگ می‌شود.',
                    'Your blood volume and heart rate rise to meet the baby’s needs. You may occasionally feel warm or dizzy as your body steadily adapts.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است بین شادی و اضطراب نوسان داشته باشید و گاهی احساس آسیب‌پذیری کنید. این احساسات با تغییرات هورمونی مرتبط است. حمایت عاطفی و استراحت کمک‌کننده است.',
                    'You may swing between joy and anxiety and sometimes feel vulnerable. These feelings link to hormonal shifts; emotional support and rest help.'
                ),
                'key_nutrition' => $this->bi(
                    'پروتئین، کلسیم و آهن را در وعده‌ها بگنجانید و از وعده‌های سبک و مکرر استفاده کنید. میوه‌ها و سبزیجات تازه و آب کافی مهم هستند. از غذاهای بسیار چرب پرهیز کنید.',
                    'Include protein, calcium and iron in meals and eat light, frequent portions. Fresh fruit, vegetables and enough water matter; avoid very greasy foods.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی ملایم و تمرینات کف لگن برای این هفته مناسب است. از فعالیت‌های شدید و پرخطر خودداری کنید. اگر احساس ضعف کردید استراحت کنید.',
                    'Gentle walking and pelvic floor exercises suit this week. Avoid intense or risky activities and rest if you feel weak.'
                ),
                'tests_and_checkups' => $this->bi(
                    'سونوگرافی اولیه برای تعیین سن دقیق بارداری و ضربان قلب جنین در این دوره رایج است. آزمایش‌های خون تکمیلی نیز ممکن است انجام شود. فشار خون و وزن ثبت می‌گردد.',
                    'An early dating ultrasound to set gestational age and confirm the heartbeat is common now. Additional blood tests may be done, and vitals are recorded.'
                ),
                'faq' => $this->faq(
                    [
                        ['چرا هنوز شکمم برجسته نشده است؟', 'در هفته هشتم رحم هنوز کوچک است و معمولاً شکم تا هفته‌های بعد از سه‌ماهه اول برجسته نمی‌شود.'],
                        ['آیا مصرف دارو برای سردرد بی‌خطر است؟', 'برخی داروها در بارداری مجاز نیستند؛ برای هر دارویی حتی مسکن ساده با پزشک مشورت کنید.'],
                    ],
                    [
                        ['Why isn’t my belly showing yet?', 'At week eight the uterus is still small, and a visible bump usually appears after the first trimester.'],
                        ['Is it safe to take medicine for a headache?', 'Some drugs are unsafe in pregnancy; check with your doctor before any medicine, even a simple painkiller.'],
                    ]
                ),
            ],
            9 => [
                'fetal_development' => $this->bi(
                    'در هفته نهم جنین به اندازه یک انگور، حدود ۲.۳ سانتی‌متر و نزدیک به ۲ گرم است. اندام‌های اصلی در حال شکل‌گیری‌اند و دم جنینی ناپدید می‌شود. عضلات کوچک شروع به کار می‌کنند و حرکات ظریف بیشتر می‌شود.',
                    'At week nine the fetus is about the size of a grape, roughly 2.3 cm and nearly 2 g. Major organs are forming, the embryonic tail disappears, and small muscles begin to work.'
                ),
                'mother_body_changes' => $this->bi(
                    'علائم سه‌ماهه اول مانند تهوع و خستگی همچنان ادامه دارد. ممکن است تغییرات پوستی و افزایش جزئی وزن را متوجه شوید. حساسیت عاطفی نیز شایع است.',
                    'First-trimester symptoms like nausea and fatigue continue. You may notice skin changes and a slight weight gain, and emotional sensitivity is common.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و از استرس و کم‌خوابی بپرهیزید. از قرار گرفتن در معرض مواد شیمیایی خانگی مضر دوری کنید. علائم نگران‌کننده را با پزشک در میان بگذارید.',
                    'Continue healthy eating and supplements, and avoid stress and sleep loss. Stay away from harmful household chemicals and report worrying symptoms.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌های منظم را دنبال کنید و درباره غربالگری‌های پیش رو با پزشک صحبت کنید. برای مدیریت علائم سه‌ماهه اول برنامه داشته باشید. مصرف آب و استراحت را حفظ کنید.',
                    'Keep regular visits and discuss upcoming screenings with your doctor. Have a plan to manage first-trimester symptoms, and maintain hydration and rest.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم در حال بزرگ شدن است و ممکن است احساس نفخ یا سنگینی داشته باشید. سیستم گوارش کندتر عمل می‌کند که می‌تواند باعث یبوست شود. این تغییرات طبیعی‌اند.',
                    'The uterus is enlarging and you may feel bloated or heavy. Digestion slows, which can cause constipation; these changes are normal.'
                ),
                'emotional_status' => $this->bi(
                    'نوسانات خلقی و گاهی احساس فرسودگی می‌تواند وجود داشته باشد. ممکن است نسبت به آینده هم هیجان‌زده و هم نگران باشید. گفتگو و استراحت به تعادل روانی کمک می‌کند.',
                    'Mood swings and occasional exhaustion may occur. You might feel both excited and anxious about the future; talking and rest support balance.'
                ),
                'key_nutrition' => $this->bi(
                    'فیبر کافی از میوه، سبزیجات و غلات کامل به پیشگیری از یبوست کمک می‌کند. پروتئین و آهن را ادامه دهید و آب فراوان بنوشید. از غذاهای فرآوری‌شده بکاهید.',
                    'Enough fiber from fruit, vegetables and whole grains helps prevent constipation. Keep up protein and iron, drink plenty of water, and cut processed foods.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی روزانه و حرکات کششی ملایم به گوارش و خلق‌وخو کمک می‌کند. از تمرینات پرفشار پرهیز کنید. فعالیت را با توجه به انرژی خود تنظیم کنید.',
                    'Daily walking and gentle stretching aid digestion and mood. Avoid high-impact workouts and adjust activity to your energy level.'
                ),
                'tests_and_checkups' => $this->bi(
                    'در این هفته پیگیری ویزیت‌های منظم ادامه دارد و پزشک ممکن است درباره غربالگری سه‌ماهه اول توضیح دهد. آزمایش‌های روتین در صورت نیاز انجام می‌شود. علائم خود را گزارش دهید.',
                    'Routine visits continue and your doctor may explain first-trimester screening. Routine tests are done as needed; report your symptoms.'
                ),
                'faq' => $this->faq(
                    [
                        ['برای یبوست بارداری چه کنم؟', 'مصرف فیبر، آب کافی و فعالیت بدنی ملایم کمک می‌کند؛ در صورت تداوم با پزشک مشورت کنید.'],
                        ['غربالگری سه‌ماهه اول چیست؟', 'مجموعه‌ای از آزمایش خون و سونوگرافی است که معمولاً بین هفته ۱۱ تا ۱۳ برای ارزیابی سلامت جنین انجام می‌شود.'],
                    ],
                    [
                        ['What helps pregnancy constipation?', 'Fiber, enough water and gentle activity help; if it persists, consult your doctor.'],
                        ['What is first-trimester screening?', 'It is a set of blood tests and an ultrasound, usually done between weeks 11 and 13 to assess the baby’s health.'],
                    ]
                ),
            ],
            10 => [
                'fetal_development' => $this->bi(
                    'در هفته دهم جنین به اندازه یک توت‌فرنگی، حدود ۳.۱ سانتی‌متر و نزدیک به ۴ گرم است. اکنون رسماً وارد مرحله جنینی (fetus) شده و اندام‌های حیاتی اصلی شکل گرفته‌اند. ناخن‌ها و جوانه‌های دندان در حال رشد هستند.',
                    'At week ten the fetus is about the size of a strawberry, roughly 3.1 cm and nearly 4 g. It has officially entered the fetal stage with vital organs formed, and nails and tooth buds are developing.'
                ),
                'mother_body_changes' => $this->bi(
                    'تهوع ممکن است هنوز ادامه داشته باشد اما به‌زودی بهتر می‌شود. ممکن است رگ‌های پوستی برجسته‌تر و سینه‌ها بزرگ‌تر شوند. کمر لباس‌ها کمی تنگ‌تر می‌شود.',
                    'Nausea may still linger but should soon ease. Skin veins may become more visible and breasts larger, and your clothes may feel a bit tighter.'
                ),
                'dos_and_donts' => $this->bi(
                    'مصرف مکمل‌ها و تغذیه متعادل را ادامه دهید و از فعالیت‌های پرخطر بپرهیزید. مراقب بهداشت دهان و دندان باشید. در صورت خونریزی یا درد شدید فوراً با پزشک تماس بگیرید.',
                    'Continue supplements and balanced nutrition, and avoid risky activities. Take care of dental hygiene, and contact your doctor at once for bleeding or severe pain.'
                ),
                'care_plan' => $this->bi(
                    'درباره زمان غربالگری سه‌ماهه اول با پزشک هماهنگ کنید. ویزیت‌های منظم و ثبت علائم را ادامه دهید. برای سونوگرافی پیش رو آماده شوید.',
                    'Coordinate the timing of first-trimester screening with your doctor. Keep regular visits and symptom tracking, and prepare for the upcoming ultrasound.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم اکنون به اندازه یک گریپ‌فروت شده و کم‌کم از لگن بالاتر می‌آید. ممکن است گاهی درد رباط گرد را در پهلوها حس کنید. بدن به‌خوبی با بارداری سازگار می‌شود.',
                    'The uterus is now grapefruit-sized and rising above the pelvis. You may sometimes feel round ligament twinges in your sides as your body adapts well.'
                ),
                'emotional_status' => $this->bi(
                    'با نزدیک شدن به پایان سه‌ماهه اول ممکن است کمی احساس آرامش بیشتری کنید. با این حال نوسانات خلقی هنوز طبیعی است. مراقبت از خود و استراحت را ادامه دهید.',
                    'As the first trimester nears its end you may feel a bit more settled. Still, mood swings remain normal; keep up self-care and rest.'
                ),
                'key_nutrition' => $this->bi(
                    'کلسیم، آهن، اسید فولیک و پروتئین همچنان پایه تغذیه هستند. لبنیات کم‌چرب، حبوبات و سبزیجات برگ‌سبز را بگنجانید. آب کافی و میان‌وعده‌های سالم داشته باشید.',
                    'Calcium, iron, folic acid and protein remain the nutritional base. Include low-fat dairy, legumes and leafy greens, with enough water and healthy snacks.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و یوگای بارداری گزینه‌های خوبی برای این هفته هستند. تمرینات کف لگن را شروع کنید. از حرکات پرخطر و افتادن احتمالی دوری کنید.',
                    'Walking, swimming and prenatal yoga are good options this week. Begin pelvic floor exercises and avoid risky moves and possible falls.'
                ),
                'tests_and_checkups' => $this->bi(
                    'در این دوره ممکن است آزمایش خون NIPT (غربالگری سلولی) به‌عنوان گزینه غیرتهاجمی مطرح شود. ویزیت روتین و بررسی فشار خون ادامه دارد. با پزشک درباره غربالگری آتی صحبت کنید.',
                    'A non-invasive NIPT blood screen may be offered as an option now. Routine visits and blood-pressure checks continue; discuss upcoming screening with your doctor.'
                ),
                'faq' => $this->faq(
                    [
                        ['آزمایش NIPT چیست؟', 'یک آزمایش خون غیرتهاجمی است که خطر برخی اختلالات کروموزومی جنین را با دقت بالا ارزیابی می‌کند.'],
                        ['درد خفیف پهلو طبیعی است؟', 'کشیدگی رباط گرد رحم می‌تواند درد خفیف پهلو ایجاد کند؛ اما درد شدید یا مداوم باید بررسی شود.'],
                    ],
                    [
                        ['What is the NIPT test?', 'It is a non-invasive blood test that accurately assesses the risk of some chromosomal conditions.'],
                        ['Is mild side pain normal?', 'Round ligament stretching can cause mild side pain, but severe or constant pain should be checked.'],
                    ]
                ),
            ],
        ];
    }

    /** @return array<int, array<string, array>> */
    private function weeks11to20(): array
    {
        return [
            11 => [
                'fetal_development' => $this->bi(
                    'در هفته یازدهم جنین به اندازه یک انجیر، حدود ۴.۱ سانتی‌متر و نزدیک به ۷ گرم است. سر هنوز بزرگ است اما بدن به‌سرعت رشد می‌کند و انگشتان از هم جدا شده‌اند. جنین شروع به حرکات پراکنده و باز و بسته کردن مشت می‌کند.',
                    'At week eleven the fetus is about the size of a fig, roughly 4.1 cm and nearly 7 g. The head is still large but the body grows fast, fingers separate, and the baby makes scattered movements.'
                ),
                'mother_body_changes' => $this->bi(
                    'تهوع معمولاً رو به کاهش می‌گذارد و ممکن است انرژی بیشتری حس کنید. رحم به‌تدریج بالاتر می‌آید و شکم کمی پر به نظر می‌رسد. موها و ناخن‌ها ممکن است سریع‌تر رشد کنند.',
                    'Nausea usually starts to ease and you may feel more energetic. The uterus rises gradually, the belly looks a little fuller, and hair and nails may grow faster.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و برای غربالگری سه‌ماهه اول برنامه‌ریزی کنید. از رژیم‌های لاغری خودداری کنید. فعالیت بدنی ملایم را حفظ کنید.',
                    'Continue healthy eating and supplements and plan for first-trimester screening. Avoid weight-loss diets and keep up gentle activity.'
                ),
                'care_plan' => $this->bi(
                    'این هفته آغاز بازه غربالگری سه‌ماهه اول است؛ با پزشک درباره سونوگرافی NT هماهنگ کنید. ویزیت‌های منظم را دنبال کنید. سوالات خود را از قبل یادداشت کنید.',
                    'This week begins the first-trimester screening window; arrange the NT scan with your doctor. Keep regular visits and note your questions in advance.'
                ),
                'body_adaptation' => $this->bi(
                    'حجم خون بیشتر شده و ممکن است گاهی احساس گرما یا تعریق بیشتر کنید. لثه‌ها ممکن است حساس‌تر شوند. بدن به‌خوبی خود را با نیازهای جنین وفق می‌دهد.',
                    'Blood volume has increased and you may feel warmer or sweat more. Gums can become more sensitive as your body adapts to the baby’s needs.'
                ),
                'emotional_status' => $this->bi(
                    'با کاهش تهوع ممکن است روحیه بهتری داشته باشید و به بارداری امیدوارتر شوید. با این حال، انتظار برای نتایج غربالگری می‌تواند کمی اضطراب‌آور باشد. آرامش خود را حفظ کنید.',
                    'As nausea eases you may feel brighter and more hopeful. Still, waiting for screening results can bring some anxiety; try to stay calm.'
                ),
                'key_nutrition' => $this->bi(
                    'پروتئین، کلسیم و آهن را ادامه دهید و ید کافی از طریق نمک یددار و لبنیات دریافت کنید. میوه و سبزیجات رنگارنگ بخورید. آب کافی همچنان اهمیت دارد.',
                    'Continue protein, calcium and iron and get enough iodine from iodized salt and dairy. Eat colorful fruit and vegetables and stay well hydrated.'
                ),
                'physical_activity' => $this->bi(
                    'با بازگشت انرژی می‌توانید پیاده‌روی و یوگای بارداری را منظم‌تر انجام دهید. تمرینات کف لگن را ادامه دهید. از فعالیت‌های پرخطر پرهیز کنید.',
                    'With returning energy you can do walking and prenatal yoga more regularly. Keep up pelvic floor exercises and avoid risky activities.'
                ),
                'tests_and_checkups' => $this->bi(
                    'سونوگرافی شفافیت پشت گردن (NT) همراه با آزمایش خون برای غربالگری سه‌ماهه اول از این هفته انجام می‌شود. این آزمون خطر برخی اختلالات کروموزومی را ارزیابی می‌کند. فشار خون و وزن نیز ثبت می‌شود.',
                    'The nuchal translucency (NT) ultrasound with a blood test for first-trimester screening is done from this week. It assesses the risk of some chromosomal conditions.'
                ),
                'faq' => $this->faq(
                    [
                        ['سونوگرافی NT چه چیزی را بررسی می‌کند؟', 'ضخامت مایع پشت گردن جنین را اندازه می‌گیرد که همراه با آزمایش خون خطر اختلالات کروموزومی را ارزیابی می‌کند.'],
                        ['چرا انرژی‌ام بیشتر شده است؟', 'با پایان یافتن اوج تغییرات هورمونی سه‌ماهه اول، بسیاری از زنان کاهش تهوع و افزایش انرژی را تجربه می‌کنند.'],
                    ],
                    [
                        ['What does the NT scan check?', 'It measures the fluid at the back of the baby’s neck, which with a blood test assesses chromosomal risk.'],
                        ['Why do I have more energy?', 'As the peak of first-trimester hormones passes, many women feel less nausea and more energy.'],
                    ]
                ),
            ],
            12 => [
                'fetal_development' => $this->bi(
                    'در هفته دوازدهم جنین به اندازه یک لیموترش، حدود ۵.۴ سانتی‌متر و نزدیک به ۱۴ گرم است. رفلکس‌ها شکل گرفته‌اند و جنین می‌تواند انگشتانش را حرکت دهد. اندام‌های حیاتی اکنون در جای خود قرار گرفته‌اند و به رشد ادامه می‌دهند.',
                    'At week twelve the fetus is about the size of a lime, roughly 5.4 cm and nearly 14 g. Reflexes have formed, the baby can move its fingers, and vital organs are now in place and maturing.'
                ),
                'mother_body_changes' => $this->bi(
                    'با پایان سه‌ماهه اول، تهوع و خستگی معمولاً کاهش می‌یابد. ممکن است برجستگی خفیف شکم را متوجه شوید. اشتها به‌تدریج بهتر می‌شود.',
                    'As the first trimester ends, nausea and fatigue usually decrease. You may notice a slight baby bump and your appetite gradually improves.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه متعادل و مکمل‌ها را ادامه دهید و مراقب افزایش وزن سالم باشید. از استرس بیش از حد و بی‌خوابی بپرهیزید. فعالیت بدنی منظم و ملایم داشته باشید.',
                    'Continue balanced nutrition and supplements and aim for healthy weight gain. Avoid excess stress and sleeplessness, and keep gentle regular activity.'
                ),
                'care_plan' => $this->bi(
                    'اگر غربالگری سه‌ماهه اول را انجام نداده‌اید، این آخرین هفته‌های مناسب برای NT است. نتایج آزمایش‌ها را با پزشک مرور کنید. ویزیت‌های منظم را دنبال کنید.',
                    'If you haven’t done first-trimester screening, these are the last suitable weeks for the NT scan. Review test results with your doctor and keep regular visits.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم اکنون از استخوان لگن بالاتر آمده و شکم کم‌کم مشخص می‌شود. جریان خون بیشتر ممکن است باعث گرفتگی بینی یا خونریزی خفیف لثه شود. این تغییرات طبیعی هستند.',
                    'The uterus has risen above the pelvic bone and the bump begins to show. Increased blood flow may cause nasal congestion or mild gum bleeding, which is normal.'
                ),
                'emotional_status' => $this->bi(
                    'رسیدن به پایان سه‌ماهه اول اغلب حس آرامش و اطمینان بیشتری به همراه دارد. ممکن است برای اعلام خبر بارداری آماده‌تر شوید. با این حال احساسات متغیر همچنان طبیعی است.',
                    'Reaching the end of the first trimester often brings relief and reassurance. You may feel readier to share the news, though changing emotions remain normal.'
                ),
                'key_nutrition' => $this->bi(
                    'کالری کافی همراه با پروتئین، آهن، کلسیم و اسید فولیک را تأمین کنید. غلات کامل و میوه و سبزیجات را در اولویت بگذارید. از مصرف بیش از حد قند و غذاهای فرآوری‌شده بپرهیزید.',
                    'Provide enough calories with protein, iron, calcium and folic acid. Favor whole grains, fruit and vegetables, and limit excess sugar and processed foods.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و یوگای بارداری برای این دوره عالی هستند. تمرینات کف لگن را منظم انجام دهید. از حرکات با خطر افتادن یا ضربه پرهیز کنید.',
                    'Walking, swimming and prenatal yoga are excellent now. Do pelvic floor exercises regularly and avoid moves with a risk of falling or impact.'
                ),
                'tests_and_checkups' => $this->bi(
                    'هفته دوازدهم آخرین بازه مناسب برای سونوگرافی NT و غربالگری ترکیبی سه‌ماهه اول است. آزمایش خون برای بررسی نشانگرها انجام می‌شود. فشار خون و وزن ثبت می‌گردد.',
                    'Week twelve is the last suitable window for the NT scan and combined first-trimester screening. A blood test for markers is done, and vitals are recorded.'
                ),
                'faq' => $this->faq(
                    [
                        ['چه زمانی می‌توانم خبر بارداری را اعلام کنم؟', 'بسیاری پس از پایان سه‌ماهه اول که خطر سقط کاهش می‌یابد خبر را اعلام می‌کنند، اما این کاملاً انتخاب شخصی است.'],
                        ['آیا برجستگی شکم در این هفته طبیعی است؟', 'بله، در پایان سه‌ماهه اول برجستگی خفیف شکم شایع است و در هر فرد متفاوت ظاهر می‌شود.'],
                    ],
                    [
                        ['When can I announce the pregnancy?', 'Many share the news after the first trimester when miscarriage risk drops, but it is entirely a personal choice.'],
                        ['Is a bump normal at this week?', 'Yes, a slight bump is common at the end of the first trimester and appears differently for each woman.'],
                    ]
                ),
            ],
            13 => [
                'fetal_development' => $this->bi(
                    'در هفته سیزدهم که آغاز سه‌ماهه دوم است، جنین به اندازه یک لیمو، حدود ۷.۴ سانتی‌متر و نزدیک به ۲۳ گرم است. اثر انگشت شکل گرفته و تارهای صوتی در حال ساخته شدن هستند. جنین می‌تواند بمکد و ببلعد.',
                    'At week thirteen, the start of the second trimester, the fetus is about the size of a lemon, roughly 7.4 cm and nearly 23 g. Fingerprints have formed, vocal cords are developing, and the baby can suck and swallow.'
                ),
                'mother_body_changes' => $this->bi(
                    'با ورود به سه‌ماهه دوم معمولاً انرژی بازمی‌گردد و تهوع کمتر می‌شود. ممکن است اشتها افزایش یابد و شکم کمی برجسته‌تر شود. برخی زنان تغییرات پوستی مانند خط تیره شکم را می‌بینند.',
                    'Entering the second trimester, energy usually returns and nausea fades. Appetite may rise and the bump grow, and some women notice skin changes like the linea nigra.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و افزایش وزن متعادل را ادامه دهید و مصرف کلسیم و آهن را جدی بگیرید. از خوابیدن طولانی به پشت در هفته‌های بعد بپرهیزید. فعالیت منظم داشته باشید.',
                    'Continue healthy eating and moderate weight gain, and take calcium and iron seriously. In later weeks avoid lying flat on your back for long, and stay active.'
                ),
                'care_plan' => $this->bi(
                    'نتایج غربالگری سه‌ماهه اول را با پزشک مرور کنید و برنامه سونوگرافی آنومالی را در نظر داشته باشید. ویزیت‌های منظم را حفظ کنید. برای تغذیه و ورزش برنامه‌ریزی کنید.',
                    'Review first-trimester screening results with your doctor and plan for the anomaly scan ahead. Keep regular visits and plan your nutrition and exercise.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم بزرگ‌تر شده و رباط‌های نگهدارنده کشیده می‌شوند که می‌تواند درد خفیف پهلو ایجاد کند. جفت اکنون به‌طور کامل وظیفه تغذیه جنین را بر عهده دارد. بدن با ثبات بیشتری کار می‌کند.',
                    'The uterus is larger and supporting ligaments stretch, which can cause mild side pain. The placenta now fully nourishes the baby and the body works more steadily.'
                ),
                'emotional_status' => $this->bi(
                    'بسیاری از زنان در سه‌ماهه دوم احساس آرامش و شادابی بیشتری دارند. ممکن است ارتباط عاطفی قوی‌تری با جنین حس کنید. با این حال گاهی نگرانی‌های جدید هم پیش می‌آید.',
                    'Many women feel calmer and more vibrant in the second trimester. You may feel a stronger bond with the baby, though new worries can occasionally arise.'
                ),
                'key_nutrition' => $this->bi(
                    'با افزایش اشتها، کیفیت غذا مهم است؛ پروتئین، لبنیات، غلات کامل و سبزیجات را در اولویت بگذارید. آهن برای افزایش حجم خون ضروری است. آب کافی بنوشید.',
                    'As appetite grows, food quality matters; prioritize protein, dairy, whole grains and vegetables. Iron is vital for rising blood volume, and drink enough water.'
                ),
                'physical_activity' => $this->bi(
                    'سه‌ماهه دوم زمان خوبی برای فعالیت منظم مانند پیاده‌روی تند، شنا و یوگای بارداری است. تمرینات کف لگن و کششی مفید هستند. به بدن خود گوش دهید و افراط نکنید.',
                    'The second trimester is a good time for regular activity like brisk walking, swimming and prenatal yoga. Pelvic floor and stretching exercises help; don’t overdo it.'
                ),
                'tests_and_checkups' => $this->bi(
                    'اگر غربالگری ترکیبی انجام نشده، این آخرین فرصت است و از این پس غربالگری سه‌ماهه دوم (کواد مارکر) مطرح می‌شود. ویزیت‌های روتین ادامه دارد. علائم غیرعادی را گزارش دهید.',
                    'If combined screening wasn’t done this is the last chance, and second-trimester quad screening becomes an option. Routine visits continue; report unusual symptoms.'
                ),
                'faq' => $this->faq(
                    [
                        ['چرا در سه‌ماهه دوم بهتر حس می‌کنم؟', 'کاهش هورمون‌های عامل تهوع و سازگاری بدن باعث بازگشت انرژی و بهبود حال عمومی می‌شود.'],
                        ['خط تیره روی شکم چیست؟', 'به آن لینه‌آ نیگرا می‌گویند که به دلیل تغییرات هورمونی ایجاد می‌شود و معمولاً پس از زایمان محو می‌شود.'],
                    ],
                    [
                        ['Why do I feel better in the second trimester?', 'Lower nausea-causing hormones and body adaptation restore energy and wellbeing.'],
                        ['What is the dark line on my belly?', 'It is called the linea nigra, caused by hormonal changes, and usually fades after birth.'],
                    ]
                ),
            ],
            14 => [
                'fetal_development' => $this->bi(
                    'در هفته چهاردهم جنین به اندازه یک هلو، حدود ۸.۷ سانتی‌متر و نزدیک به ۴۳ گرم است. کرک ظریفی به نام لانوگو بدن را می‌پوشاند و جنین می‌تواند حالات صورت بگیرد. کبد و کلیه‌ها شروع به کار کرده‌اند.',
                    'At week fourteen the fetus is about the size of a peach, roughly 8.7 cm and nearly 43 g. Fine hair called lanugo covers the body, the baby can make facial expressions, and the liver and kidneys start working.'
                ),
                'mother_body_changes' => $this->bi(
                    'انرژی بیشتر و کاهش تهوع معمولاً ادامه دارد و اشتها بهتر می‌شود. ممکن است شکم به‌وضوح برجسته‌تر شود. برخی زنان درد رباط گرد را در پهلوها حس می‌کنند.',
                    'More energy and less nausea usually continue and appetite improves. The bump may become clearly visible, and some women feel round ligament pain in the sides.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه مغذی و مصرف مکمل‌ها را ادامه دهید و از پرخوری پرهیز کنید. لباس راحت و کفش مناسب بپوشید. فعالیت بدنی منظم را حفظ کنید.',
                    'Continue nutritious eating and supplements and avoid overeating. Wear comfortable clothes and supportive shoes, and keep regular activity.'
                ),
                'care_plan' => $this->bi(
                    'برای غربالگری سه‌ماهه دوم و سونوگرافی آنومالی پیش رو برنامه‌ریزی کنید. ویزیت‌های منظم و کنترل فشار خون را ادامه دهید. مراقبت از پوست و پیشگیری از یبوست را در نظر داشته باشید.',
                    'Plan for second-trimester screening and the upcoming anomaly scan. Keep regular visits and blood-pressure checks, and mind skin care and constipation.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم بالاتر می‌آید و ممکن است بالای استخوان شرمگاهی قابل لمس باشد. جریان خون بیشتر ممکن است پوست را شاداب‌تر کند. بدن با ثبات بیشتری خود را تنظیم می‌کند.',
                    'The uterus rises and may be felt above the pubic bone. Increased blood flow can give the skin a healthy glow as the body settles more steadily.'
                ),
                'emotional_status' => $this->bi(
                    'حال عمومی معمولاً بهتر است و ممکن است هیجان بیشتری برای آماده‌سازی نوزاد داشته باشید. ارتباط با جنین قوی‌تر می‌شود. اگر نگرانی دارید با پزشک یا نزدیکان صحبت کنید.',
                    'Overall mood is usually better and you may feel more excited to prepare for the baby. The bond grows; if worried, talk to your doctor or loved ones.'
                ),
                'key_nutrition' => $this->bi(
                    'پروتئین، آهن، کلسیم و امگا-۳ را برای رشد مغز و بدن جنین تأمین کنید. ماهی کم‌جیوه، تخم‌مرغ و مغزها مفید هستند. فیبر کافی برای پیشگیری از یبوست مصرف کنید.',
                    'Provide protein, iron, calcium and omega-3 for the baby’s brain and body. Low-mercury fish, eggs and nuts help; eat enough fiber to prevent constipation.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و تمرینات ملایم قدرتی برای این هفته مناسب است. تمرینات کف لگن را ادامه دهید. از دراز کشیدن طولانی به پشت و حرکات پرخطر بپرهیزید.',
                    'Walking, swimming and gentle strength exercises suit this week. Keep up pelvic floor work and avoid long back-lying and risky moves.'
                ),
                'tests_and_checkups' => $this->bi(
                    'غربالگری سه‌ماهه دوم شامل آزمایش خون کواد مارکر و AFP از این هفته قابل انجام است. این آزمون خطر نقص لوله عصبی و برخی اختلالات را ارزیابی می‌کند. ویزیت روتین ادامه دارد.',
                    'Second-trimester screening including the quad marker and AFP blood test can be done from this week. It assesses the risk of neural tube defects and some conditions.'
                ),
                'faq' => $this->faq(
                    [
                        ['آزمایش AFP چیست؟', 'اندازه‌گیری آلفافتوپروتئین در خون مادر است که برای غربالگری نقص لوله عصبی و برخی اختلالات جنین استفاده می‌شود.'],
                        ['درد رباط گرد چگونه است؟', 'دردی تیز و کوتاه در پهلوها هنگام حرکت است که به دلیل کشیده شدن رباط‌های رحم رخ می‌دهد و معمولاً بی‌خطر است.'],
                    ],
                    [
                        ['What is the AFP test?', 'It measures alpha-fetoprotein in the mother’s blood to screen for neural tube defects and some conditions.'],
                        ['What is round ligament pain?', 'It is a brief sharp pain in the sides during movement from stretching uterine ligaments, usually harmless.'],
                    ]
                ),
            ],
            15 => [
                'fetal_development' => $this->bi(
                    'در هفته پانزدهم جنین به اندازه یک سیب، حدود ۱۰ سانتی‌متر و نزدیک به ۷۰ گرم است. جنین می‌تواند نور را از پشت پلک‌های بسته حس کند و حرکات دست و پا بیشتر می‌شود. استخوان‌ها در حال محکم شدن هستند.',
                    'At week fifteen the fetus is about the size of an apple, roughly 10 cm and nearly 70 g. It can sense light through closed eyelids, moves its limbs more, and its bones are hardening.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است احساس انرژی بیشتری داشته باشید و اشتها افزایش یابد. برخی زنان گرفتگی بینی یا خونریزی خفیف لثه را تجربه می‌کنند. شکم به‌تدریج گردتر می‌شود.',
                    'You may feel more energetic with a growing appetite. Some women experience nasal congestion or mild gum bleeding, and the belly gradually rounds out.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه متعادل و مکمل‌ها را ادامه دهید و مراقب بهداشت دهان باشید. از قرار گرفتن طولانی در وضعیت‌های نامناسب بپرهیزید. فعالیت بدنی منظم را حفظ کنید.',
                    'Continue balanced nutrition and supplements and maintain oral hygiene. Avoid staying long in awkward positions and keep regular activity.'
                ),
                'care_plan' => $this->bi(
                    'اگر غربالگری کواد را انجام نداده‌اید، این هفته زمان مناسبی است. برای سونوگرافی آنومالی پیش رو هماهنگ کنید. ویزیت‌های منظم را دنبال کنید.',
                    'If you haven’t had the quad screen, this is a good week. Coordinate the upcoming anomaly scan and keep regular visits.'
                ),
                'body_adaptation' => $this->bi(
                    'حجم خون افزایش یافته و ممکن است گاهی سرگیجه خفیف داشته باشید. مفاصل کمی شل‌تر می‌شوند تا بدن برای رشد آماده شود. این تغییرات طبیعی هستند.',
                    'Blood volume has increased and you may feel occasional mild dizziness. Joints loosen a little to prepare the body for growth; these changes are normal.'
                ),
                'emotional_status' => $this->bi(
                    'روحیه معمولاً بهتر است و ممکن است برای آینده برنامه‌ریزی کنید. با این حال گاهی حواس‌پرتی یا فراموشی بارداری آزاردهنده است. با خودتان مهربان باشید.',
                    'Mood is usually better and you may start planning ahead. Still, pregnancy forgetfulness can be bothersome; be gentle with yourself.'
                ),
                'key_nutrition' => $this->bi(
                    'کلسیم و ویتامین D برای استخوان‌سازی جنین اهمیت دارند. لبنیات، سبزیجات برگ‌سبز و پروتئین باکیفیت مصرف کنید. آهن و آب کافی را فراموش نکنید.',
                    'Calcium and vitamin D matter for the baby’s bones. Eat dairy, leafy greens and quality protein, and don’t forget iron and enough water.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و یوگای بارداری برای حفظ تناسب مناسب هستند. تمرینات کف لگن و کششی را ادامه دهید. در صورت سرگیجه، آهسته حرکت کنید و آب بنوشید.',
                    'Walking, swimming and prenatal yoga help stay fit. Continue pelvic floor and stretching exercises, and if dizzy, move slowly and hydrate.'
                ),
                'tests_and_checkups' => $this->bi(
                    'غربالگری سه‌ماهه دوم (کواد مارکر و AFP) در بازه هفته ۱۵ تا ۱۸ انجام می‌شود. در صورت نیاز، پزشک ممکن است آمنیوسنتز را مطرح کند. ویزیت روتین و کنترل فشار خون ادامه دارد.',
                    'Second-trimester screening (quad marker and AFP) is done in the weeks 15 to 18 window. If needed, your doctor may discuss amniocentesis; routine checks continue.'
                ),
                'faq' => $this->faq(
                    [
                        ['آمنیوسنتز چیست و چه زمانی لازم است؟', 'نمونه‌گیری از مایع آمنیوتیک برای بررسی دقیق کروموزوم‌هاست و معمولاً در صورت غربالگری پرخطر یا سن بالای مادر مطرح می‌شود.'],
                        ['فراموشی بارداری واقعی است؟', 'بله، تغییرات هورمونی و کم‌خوابی می‌تواند باعث حواس‌پرتی خفیف شود که معمولاً موقتی است.'],
                    ],
                    [
                        ['What is amniocentesis and when is it needed?', 'It samples amniotic fluid to examine chromosomes, usually offered for high-risk screening or older maternal age.'],
                        ['Is pregnancy brain real?', 'Yes, hormonal changes and poor sleep can cause mild forgetfulness, which is usually temporary.'],
                    ]
                ),
            ],
            16 => [
                'fetal_development' => $this->bi(
                    'در هفته شانزدهم جنین به اندازه یک آووکادو، حدود ۱۱.۶ سانتی‌متر و نزدیک به ۱۰۰ گرم است. عضلات صورت کار می‌کنند و جنین می‌تواند اخم یا حالت‌های مختلف بگیرد. حرکات هماهنگ‌تر شده و ممکن است به‌زودی حس شود.',
                    'At week sixteen the fetus is about the size of an avocado, roughly 11.6 cm and nearly 100 g. Facial muscles work so the baby can frown, and movements grow more coordinated and may soon be felt.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است اولین حرکات خفیف جنین را در هفته‌های آینده حس کنید. شکم واضح‌تر می‌شود و ممکن است پوست شکم دچار خارش شود. اشتها و انرژی معمولاً خوب است.',
                    'You may feel the first faint baby movements in the coming weeks. The bump grows clearer, the belly skin may itch, and appetite and energy are usually good.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و از مرطوب‌کننده برای پوست شکم استفاده کنید. از بلند کردن اجسام سنگین خودداری کنید. فعالیت بدنی منظم داشته باشید.',
                    'Continue healthy eating and supplements and moisturize the belly skin. Avoid lifting heavy objects and keep regular activity.'
                ),
                'care_plan' => $this->bi(
                    'برای سونوگرافی آنومالی که به‌زودی انجام می‌شود آماده شوید. اگر غربالگری کواد را نداده‌اید، همین هفته اقدام کنید. ویزیت‌های منظم را دنبال کنید.',
                    'Prepare for the anomaly scan coming soon. If you haven’t had the quad screen, do it this week, and keep regular visits.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم اکنون بین ناف و استخوان شرمگاهی قرار دارد و مرکز ثقل بدن تغییر می‌کند. ممکن است کمردرد خفیف را تجربه کنید. بدن به‌خوبی وزن اضافه را تحمل می‌کند.',
                    'The uterus now sits between the navel and pubic bone and your center of gravity shifts. You may feel mild back pain as the body handles the added weight.'
                ),
                'emotional_status' => $this->bi(
                    'حس ارتباط با جنین قوی‌تر می‌شود، به‌ویژه با نزدیک شدن به حس کردن حرکات. ممکن است هیجان و انتظار بیشتری داشته باشید. مراقبت از سلامت روان را ادامه دهید.',
                    'The bond with the baby strengthens, especially as movements approach. You may feel more excitement and anticipation; keep caring for your mental health.'
                ),
                'key_nutrition' => $this->bi(
                    'پروتئین، آهن، کلسیم و امگا-۳ را ادامه دهید و آب کافی بنوشید. غذاهای غنی از فیبر یبوست را کاهش می‌دهند. از میان‌وعده‌های سالم استفاده کنید.',
                    'Continue protein, iron, calcium and omega-3 and drink enough water. Fiber-rich foods reduce constipation; choose healthy snacks.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و تمرینات تقویت‌کننده کمر و کف لگن مفید هستند. وضعیت بدنی صحیح را رعایت کنید تا کمردرد کمتر شود. از حرکات پرخطر بپرهیزید.',
                    'Walking, swimming and back and pelvic floor strengthening help. Maintain good posture to ease back pain and avoid risky moves.'
                ),
                'tests_and_checkups' => $this->bi(
                    'غربالگری سه‌ماهه دوم همچنان در این بازه قابل انجام است و سونوگرافی آنومالی به‌زودی برنامه‌ریزی می‌شود. ویزیت روتین شامل کنترل فشار خون و ضربان قلب جنین است.',
                    'Second-trimester screening is still available now, and the anomaly scan will be scheduled soon. Routine visits include blood pressure and fetal heartbeat checks.'
                ),
                'faq' => $this->faq(
                    [
                        ['چه زمانی حرکات جنین را حس می‌کنم؟', 'بسیاری از مادران اولین حرکات را بین هفته ۱۶ تا ۲۰ حس می‌کنند؛ در بارداری اول ممکن است کمی دیرتر باشد.'],
                        ['خارش پوست شکم طبیعی است؟', 'بله، کشیده شدن پوست باعث خارش می‌شود؛ مرطوب‌کننده کمک می‌کند، اما خارش شدید سراسری را با پزشک مطرح کنید.'],
                    ],
                    [
                        ['When will I feel the baby move?', 'Many mothers feel first movements between weeks 16 and 20; in a first pregnancy it may be a bit later.'],
                        ['Is belly itching normal?', 'Yes, stretching skin causes itching; moisturizer helps, but report severe widespread itching to your doctor.'],
                    ]
                ),
            ],
            17 => [
                'fetal_development' => $this->bi(
                    'در هفته هفدهم جنین به اندازه یک شلغم، حدود ۱۳ سانتی‌متر و نزدیک به ۱۴۰ گرم است. لایه چربی محافظ زیر پوست شروع به تشکیل می‌کند و اسکلت از غضروف به استخوان تبدیل می‌شود. جنین می‌تواند صداها را کم‌کم حس کند.',
                    'At week seventeen the fetus is about the size of a turnip, roughly 13 cm and nearly 140 g. A protective fat layer begins forming under the skin, the skeleton turns from cartilage to bone, and the baby starts sensing sounds.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است اشتها زیاد و انرژی خوبی داشته باشید و شکم واضح‌تر شود. برخی زنان تعریق بیشتر یا گرگرفتگی خفیف را تجربه می‌کنند. ممکن است حرکات اولیه جنین را حس کنید.',
                    'You may have a strong appetite and good energy as the bump grows clearer. Some women sweat more or feel mild flushes, and you may sense early baby movements.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه مغذی و مکمل‌ها را ادامه دهید و از افزایش وزن ناگهانی پرهیز کنید. کفش راحت بپوشید و وضعیت بدنی درست را رعایت کنید. فعالیت بدنی منظم داشته باشید.',
                    'Continue nutritious eating and supplements and avoid sudden weight gain. Wear comfortable shoes, keep good posture, and stay active.'
                ),
                'care_plan' => $this->bi(
                    'برای سونوگرافی آنومالی که در هفته‌های ۱۸ تا ۲۲ انجام می‌شود آماده شوید. ویزیت‌های منظم و کنترل فشار خون را دنبال کنید. علائم غیرعادی را گزارش دهید.',
                    'Prepare for the anomaly scan done in weeks 18 to 22. Keep regular visits and blood-pressure checks, and report unusual symptoms.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم در حال بزرگ شدن است و ممکن است گاهی احساس فشار در ناحیه کمر و لگن کنید. سیستم قلبی‌عروقی برای پشتیبانی از جنین سخت‌تر کار می‌کند. بدن به‌خوبی سازگار می‌شود.',
                    'The uterus is enlarging and you may feel pressure in the back and pelvis. The cardiovascular system works harder to support the baby as the body adapts well.'
                ),
                'emotional_status' => $this->bi(
                    'با حس کردن حرکات احتمالی جنین، ارتباط عاطفی عمیق‌تر می‌شود. ممکن است رویاهای واضح‌تری هم داشته باشید. حفظ آرامش و استراحت کافی مهم است.',
                    'Feeling possible baby movements deepens the emotional bond. You may also have more vivid dreams; staying calm and well rested matters.'
                ),
                'key_nutrition' => $this->bi(
                    'کلسیم، ویتامین D و پروتئین برای رشد استخوان و عضله جنین ضروری هستند. آهن را برای پیشگیری از کم‌خونی ادامه دهید. میوه، سبزیجات و آب کافی بخورید.',
                    'Calcium, vitamin D and protein are essential for the baby’s bones and muscles. Continue iron to prevent anemia and eat fruit, vegetables and enough water.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و یوگای بارداری برای این هفته مناسب هستند. تمرینات کف لگن و تقویت کمر را ادامه دهید. از حرکات ناگهانی و پرخطر بپرهیزید.',
                    'Walking, swimming and prenatal yoga suit this week. Keep pelvic floor and back-strengthening exercises and avoid sudden or risky moves.'
                ),
                'tests_and_checkups' => $this->bi(
                    'اگر غربالگری کواد را انجام نداده‌اید، این آخرین هفته‌های مناسب است. سونوگرافی آنومالی به‌زودی برنامه‌ریزی می‌شود. ویزیت روتین و بررسی رشد جنین ادامه دارد.',
                    'If you haven’t had the quad screen, these are the last suitable weeks. The anomaly scan will be scheduled soon; routine visits and growth checks continue.'
                ),
                'faq' => $this->faq(
                    [
                        ['چرا بیشتر عرق می‌کنم؟', 'افزایش حجم خون و متابولیسم در بارداری باعث تعریق بیشتر می‌شود که معمولاً طبیعی است.'],
                        ['رویاهای واضح بارداری چرا رخ می‌دهد؟', 'تغییرات هورمونی و کیفیت خواب می‌تواند باعث رویاهای زنده‌تر شود و جای نگرانی نیست.'],
                    ],
                    [
                        ['Why am I sweating more?', 'Increased blood volume and metabolism in pregnancy cause more sweating, which is usually normal.'],
                        ['Why do I have vivid pregnancy dreams?', 'Hormonal changes and sleep quality can cause more vivid dreams and are nothing to worry about.'],
                    ]
                ),
            ],
            18 => [
                'fetal_development' => $this->bi(
                    'در هفته هجدهم جنین به اندازه یک فلفل دلمه‌ای، حدود ۱۴.۲ سانتی‌متر و نزدیک به ۱۹۰ گرم است. گوش‌ها به موقعیت نهایی رسیده‌اند و جنین صداها را می‌شنود. سیستم عصبی به‌سرعت در حال تکامل است.',
                    'At week eighteen the fetus is about the size of a bell pepper, roughly 14.2 cm and nearly 190 g. The ears have reached their final position and the baby hears sounds, while the nervous system develops rapidly.'
                ),
                'mother_body_changes' => $this->bi(
                    'حرکات جنین ممکن است واضح‌تر حس شود. ممکن است کمردرد، گرفتگی عضلات پا و افزایش اشتها را تجربه کنید. مرکز ثقل بدن تغییر کرده و تعادل کمی متفاوت است.',
                    'Baby movements may feel clearer. You might experience back pain, leg cramps and more appetite, and your shifting center of gravity slightly changes balance.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مصرف مکمل‌ها را ادامه دهید و از خوابیدن طولانی به پشت بپرهیزید. کفش راحت بپوشید و وضعیت بدنی درست را رعایت کنید. آب کافی بنوشید.',
                    'Continue healthy eating and supplements and avoid long back-lying. Wear comfortable shoes, keep good posture, and drink enough water.'
                ),
                'care_plan' => $this->bi(
                    'سونوگرافی آنومالی (بررسی دقیق آناتومی جنین) از این هفته انجام می‌شود؛ آن را برنامه‌ریزی کنید. ویزیت‌های منظم را دنبال کنید. سوالات خود را برای پزشک آماده کنید.',
                    'The anomaly scan (detailed fetal anatomy check) is done from this week; schedule it. Keep regular visits and prepare your questions for the doctor.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم اکنون به نزدیکی ناف رسیده و فشار روی عروق ممکن است گاهی سرگیجه ایجاد کند. به پهلو خوابیدن راحت‌تر می‌شود. بدن با تغییرات وزن سازگار می‌شود.',
                    'The uterus now nears the navel and pressure on vessels may cause occasional dizziness. Side-sleeping becomes more comfortable as the body adapts to weight changes.'
                ),
                'emotional_status' => $this->bi(
                    'حس کردن حرکات جنین اغلب شادی و اطمینان می‌آورد. ممکن است برای آماده‌سازی نوزاد هیجان‌زده باشید. اگر اضطراب دارید، درباره آن صحبت کنید.',
                    'Feeling the baby move often brings joy and reassurance. You may be excited to prepare for the baby; if anxious, talk about it.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، منیزیم و پتاسیم می‌توانند به کاهش گرفتگی عضلات کمک کنند. موز، لبنیات، مغزها و سبزیجات برگ‌سبز مفید هستند. آب کافی بنوشید.',
                    'Iron, calcium, magnesium and potassium can help reduce cramps. Bananas, dairy, nuts and leafy greens help; drink enough water.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و کشش ملایم عضلات پا از گرفتگی جلوگیری می‌کند. تمرینات کف لگن را ادامه دهید. از ایستادن طولانی بپرهیزید.',
                    'Walking, swimming and gentle leg stretches help prevent cramps. Keep pelvic floor exercises and avoid standing for long periods.'
                ),
                'tests_and_checkups' => $this->bi(
                    'سونوگرافی آنومالی برای بررسی دقیق ساختار قلب، مغز، ستون فقرات و اندام‌های جنین از این هفته انجام می‌شود. جنسیت جنین نیز اغلب قابل تشخیص است. ویزیت روتین ادامه دارد.',
                    'The anomaly ultrasound to examine the heart, brain, spine and organs in detail is done from this week. The baby’s sex is often identifiable, and routine visits continue.'
                ),
                'faq' => $this->faq(
                    [
                        ['سونوگرافی آنومالی چه چیزهایی را بررسی می‌کند؟', 'ساختار اندام‌ها، قلب، مغز، ستون فقرات، جفت و مایع آمنیوتیک را با دقت ارزیابی می‌کند تا از رشد سالم اطمینان حاصل شود.'],
                        ['برای گرفتگی عضلات پا چه کنم؟', 'کشش ملایم، آب کافی و مواد معدنی مانند منیزیم و پتاسیم کمک می‌کند؛ در صورت درد شدید و مداوم با پزشک مشورت کنید.'],
                    ],
                    [
                        ['What does the anomaly scan check?', 'It carefully assesses the organs, heart, brain, spine, placenta and amniotic fluid to confirm healthy development.'],
                        ['What helps leg cramps?', 'Gentle stretching, enough water and minerals like magnesium and potassium help; see a doctor if pain is severe or constant.'],
                    ]
                ),
            ],
            19 => [
                'fetal_development' => $this->bi(
                    'در هفته نوزدهم جنین به اندازه یک انبه، حدود ۱۵.۳ سانتی‌متر و نزدیک به ۲۴۰ گرم است. پوشش محافظ چربی‌مانندی به نام ورنیکس روی پوست شکل می‌گیرد. حواس بینایی، شنوایی و چشایی در حال تکامل هستند.',
                    'At week nineteen the fetus is about the size of a mango, roughly 15.3 cm and nearly 240 g. A protective waxy coating called vernix forms on the skin, and sight, hearing and taste are developing.'
                ),
                'mother_body_changes' => $this->bi(
                    'حرکات جنین منظم‌تر و واضح‌تر می‌شود. ممکن است کمردرد، تغییرات پوستی یا گرفتگی بینی داشته باشید. رشد شکم می‌تواند بر تعادل و راه رفتن اثر بگذارد.',
                    'Baby movements become more regular and clear. You may have back pain, skin changes or nasal congestion, and the growing bump can affect balance and walking.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه متعادل و مکمل‌ها را ادامه دهید و از استرس زیاد بپرهیزید. برای کاهش کمردرد از وضعیت درست بدن و استراحت استفاده کنید. فعالیت بدنی منظم داشته باشید.',
                    'Continue balanced nutrition and supplements and avoid excess stress. Use good posture and rest to ease back pain, and stay active.'
                ),
                'care_plan' => $this->bi(
                    'اگر سونوگرافی آنومالی را انجام نداده‌اید، این هفته زمان مناسبی است. ویزیت‌های منظم و کنترل فشار خون را دنبال کنید. علائم هشدار مانند ورم شدید را جدی بگیرید.',
                    'If you haven’t had the anomaly scan, this is a good week. Keep regular visits and blood-pressure checks, and take warning signs like severe swelling seriously.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم به سطح ناف رسیده و ممکن است پوست شکم کشیده و حساس شود. رباط‌ها شل‌تر شده‌اند که می‌تواند بر مفاصل اثر بگذارد. بدن با موفقیت خود را تنظیم می‌کند.',
                    'The uterus reaches the navel and the belly skin may stretch and feel tender. Ligaments are looser, which can affect joints, as the body adjusts successfully.'
                ),
                'emotional_status' => $this->bi(
                    'با تجربه حرکات منظم، حس مادرانه قوی‌تر می‌شود. ممکن است برای انتخاب نام و خرید لوازم هیجان داشته باشید. اگر نگرانی دارید با پزشک صحبت کنید.',
                    'With regular movements the maternal bond grows stronger. You may be excited to choose names and buy items; if worried, talk to your doctor.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن و ویتامین C را با هم مصرف کنید تا جذب آهن بهتر شود و از کم‌خونی جلوگیری کند. کلسیم و پروتئین را ادامه دهید. آب کافی و فیبر برای گوارش مهم است.',
                    'Take iron with vitamin C to boost absorption and prevent anemia. Continue calcium and protein, and keep enough water and fiber for digestion.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و یوگای بارداری به کاهش کمردرد و بهبود خواب کمک می‌کند. تمرینات کف لگن را ادامه دهید. از تعادل خود مراقبت کنید و افراط نکنید.',
                    'Walking, swimming and prenatal yoga ease back pain and improve sleep. Keep pelvic floor exercises, mind your balance, and don’t overdo it.'
                ),
                'tests_and_checkups' => $this->bi(
                    'سونوگرافی آنومالی همچنان در این بازه انجام‌شدنی است و رشد جنین بررسی می‌شود. ویزیت روتین شامل کنترل فشار خون، وزن و ضربان قلب جنین است. علائم غیرعادی را گزارش دهید.',
                    'The anomaly scan is still possible now and fetal growth is assessed. Routine visits include blood pressure, weight and fetal heartbeat; report unusual symptoms.'
                ),
                'faq' => $this->faq(
                    [
                        ['ورنیکس چیست؟', 'پوشش مومی‌شکلی است که پوست جنین را در برابر مایع آمنیوتیک محافظت می‌کند و تا نزدیک زایمان باقی می‌ماند.'],
                        ['کمردرد بارداری را چطور کاهش دهم؟', 'وضعیت بدنی درست، کفش راحت، کشش ملایم و استفاده از بالش هنگام خواب کمک می‌کند؛ درد شدید را با پزشک مطرح کنید.'],
                    ],
                    [
                        ['What is vernix?', 'It is a waxy coating that protects the baby’s skin from amniotic fluid and remains until near birth.'],
                        ['How can I reduce pregnancy back pain?', 'Good posture, comfortable shoes, gentle stretching and a pillow while sleeping help; report severe pain to your doctor.'],
                    ]
                ),
            ],
            20 => [
                'fetal_development' => $this->bi(
                    'در هفته بیستم که نیمه راه بارداری است، جنین به اندازه یک موز، حدود ۲۵ سانتی‌متر (از سر تا پاشنه) و نزدیک به ۳۰۰ گرم است. جنین به‌طور منظم حرکت می‌کند، می‌بلعد و الگوی خواب و بیداری دارد. مو و ناخن‌ها در حال رشد هستند.',
                    'At week twenty, the halfway point, the fetus is about the size of a banana, roughly 25 cm crown-to-heel and nearly 300 g. The baby moves regularly, swallows, and has sleep-wake patterns, while hair and nails grow.'
                ),
                'mother_body_changes' => $this->bi(
                    'شکم اکنون به‌وضوح برجسته است و حرکات جنین منظم حس می‌شود. ممکن است سوزش سردل، تنگی نفس خفیف یا تورم پا داشته باشید. پوست ممکن است دچار ترک‌های کششی شود.',
                    'The bump is now clearly visible and baby movements are felt regularly. You may have heartburn, mild shortness of breath or leg swelling, and stretch marks can appear.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و به پهلو (ترجیحاً چپ) بخوابید. از وعده‌های سنگین برای کاهش سوزش سردل بپرهیزید. فعالیت بدنی منظم داشته باشید.',
                    'Continue healthy eating and supplements and sleep on your side, preferably the left. Avoid heavy meals to reduce heartburn and stay active.'
                ),
                'care_plan' => $this->bi(
                    'هفته بیستم زمان معمول سونوگرافی آنومالی است؛ اگر انجام نداده‌اید حتماً هماهنگ کنید. ویزیت‌های منظم و کنترل فشار خون را دنبال کنید. شمارش حرکات جنین را کم‌کم شروع کنید.',
                    'Week twenty is the usual time for the anomaly scan; if not done, arrange it. Keep regular visits and blood-pressure checks and begin noting fetal movements.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم به سطح ناف رسیده و بدن با مرکز ثقل جدید سازگار شده است. فشار بر معده می‌تواند سوزش سردل ایجاد کند. سیستم گردش خون با قدرت کار می‌کند.',
                    'The uterus reaches the navel and the body has adjusted to a new center of gravity. Pressure on the stomach can cause heartburn as circulation works strongly.'
                ),
                'emotional_status' => $this->bi(
                    'رسیدن به نیمه راه بارداری اغلب حس دستاورد و امیدواری می‌آورد. ارتباط با جنین از طریق حرکاتش عمیق‌تر می‌شود. مراقبت از خود و استراحت را ادامه دهید.',
                    'Reaching the halfway point often brings a sense of achievement and hope. The bond deepens through the baby’s movements; keep up self-care and rest.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم و پروتئین را ادامه دهید و برای سوزش سردل وعده‌های کوچک و مکرر بخورید. از غذاهای تند و چرب بپرهیزید. آب و فیبر کافی مصرف کنید.',
                    'Continue iron, calcium and protein and eat small frequent meals for heartburn. Avoid spicy and greasy foods and get enough water and fiber.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و یوگای بارداری برای این هفته ایده‌آل هستند. تمرینات کف لگن به آمادگی زایمان کمک می‌کند. از دراز کشیدن طولانی به پشت بپرهیزید.',
                    'Walking, swimming and prenatal yoga are ideal this week. Pelvic floor exercises help prepare for birth; avoid long back-lying.'
                ),
                'tests_and_checkups' => $this->bi(
                    'سونوگرافی آنومالی معمولاً در هفته ۱۸ تا ۲۲ و اغلب در همین هفته انجام می‌شود و آناتومی کامل جنین را بررسی می‌کند. ویزیت روتین شامل فشار خون، وزن و ضربان قلب جنین است.',
                    'The anomaly scan is usually done in weeks 18 to 22, often this week, examining the baby’s full anatomy. Routine visits include blood pressure, weight and fetal heartbeat.'
                ),
                'faq' => $this->faq(
                    [
                        ['چرا سونوگرافی هفته بیستم مهم است؟', 'این سونوگرافی آناتومی کامل جنین را بررسی می‌کند و بسیاری از ناهنجاری‌های ساختاری را در این مرحله شناسایی می‌کند.'],
                        ['برای سوزش سردل چه کنم؟', 'وعده‌های کوچک، پرهیز از غذای تند و چرب، و نخوابیدن بلافاصله پس از غذا کمک می‌کند؛ در صورت شدید بودن با پزشک مشورت کنید.'],
                    ],
                    [
                        ['Why is the week-20 scan important?', 'It reviews the baby’s full anatomy and detects many structural anomalies at this stage.'],
                        ['What helps heartburn?', 'Small meals, avoiding spicy and greasy food, and not lying down right after eating help; see a doctor if severe.'],
                    ]
                ),
            ],
        ];
    }

    /** @return array<int, array<string, array>> */
    private function weeks21to30(): array
    {
        return [
            21 => [
                'fetal_development' => $this->bi(
                    'در هفته بیست‌ویکم جنین به اندازه یک هویج، حدود ۲۶.۷ سانتی‌متر و نزدیک به ۳۶۰ گرم است. جنین مایع آمنیوتیک را می‌بلعد که به رشد دستگاه گوارش کمک می‌کند. حرکات دست و پا قوی‌تر و منظم‌تر می‌شود.',
                    'At week twenty-one the fetus is about the size of a carrot, roughly 26.7 cm and nearly 360 g. The baby swallows amniotic fluid, aiding gut development, and limb movements grow stronger and more regular.'
                ),
                'mother_body_changes' => $this->bi(
                    'حرکات جنین اکنون واضح و منظم است. ممکن است افزایش اشتها، تورم خفیف پا و ترک‌های پوستی داشته باشید. انرژی معمولاً هنوز خوب است.',
                    'Baby movements are now clear and regular. You may have more appetite, mild leg swelling and stretch marks, and energy is usually still good.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و پاها را هنگام استراحت بالا بگذارید. از ایستادن طولانی بپرهیزید. آب کافی بنوشید و نمک را متعادل مصرف کنید.',
                    'Continue healthy eating and supplements and elevate your legs when resting. Avoid standing long, drink enough water, and moderate salt.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌های منظم و کنترل رشد جنین را دنبال کنید. شمارش حرکات جنین را به یک عادت روزانه تبدیل کنید. برای غربالگری دیابت بارداری در هفته‌های آینده آماده شوید.',
                    'Keep regular visits and growth checks. Make counting baby movements a daily habit and prepare for gestational diabetes screening in coming weeks.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم بالای ناف رسیده و ممکن است فشار بر عروق پا باعث تورم شود. مفاصل شل‌تر شده‌اند و راه رفتن ممکن است متفاوت باشد. بدن با موفقیت خود را تنظیم می‌کند.',
                    'The uterus is above the navel and pressure on leg veins may cause swelling. Joints are looser and walking may feel different as the body adapts well.'
                ),
                'emotional_status' => $this->bi(
                    'حس ارتباط با جنین از طریق حرکاتش عمیق‌تر می‌شود. ممکن است برای آماده‌سازی اتاق نوزاد هیجان داشته باشید. اگر گاهی نگران هستید، طبیعی است.',
                    'The bond deepens through the baby’s movements. You may be excited to prepare the nursery; occasional worry is normal.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم و پروتئین را ادامه دهید و فیبر کافی برای پیشگیری از یبوست مصرف کنید. غذاهای غنی از پتاسیم به کاهش تورم کمک می‌کند. از قند و نمک زیاد بپرهیزید.',
                    'Continue iron, calcium and protein and eat enough fiber to prevent constipation. Potassium-rich foods help reduce swelling; limit excess sugar and salt.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و یوگای بارداری به کاهش تورم و بهبود خواب کمک می‌کند. تمرینات کف لگن را ادامه دهید. در صورت خستگی استراحت کنید.',
                    'Walking, swimming and prenatal yoga reduce swelling and improve sleep. Keep pelvic floor exercises and rest when tired.'
                ),
                'tests_and_checkups' => $this->bi(
                    'در این هفته ویزیت روتین شامل کنترل فشار خون، وزن و ضربان قلب جنین است. پزشک ممکن است درباره آزمایش تحمل گلوکز پیش رو توضیح دهد. علائم هشدار را گزارش دهید.',
                    'Routine visits this week include blood pressure, weight and fetal heartbeat. Your doctor may explain the upcoming glucose tolerance test; report warning signs.'
                ),
                'faq' => $this->faq(
                    [
                        ['چرا پاهایم ورم می‌کند؟', 'فشار رحم بر عروق و افزایش مایعات بدن باعث تورم خفیف پا می‌شود؛ بالا گذاشتن پاها کمک می‌کند اما تورم ناگهانی و شدید را به پزشک بگویید.'],
                        ['شمارش حرکات جنین چگونه است؟', 'در زمان مشخصی از روز که جنین فعال‌تر است، حرکات را بشمارید؛ کاهش محسوس حرکات باید بررسی شود.'],
                    ],
                    [
                        ['Why are my legs swelling?', 'Uterine pressure on veins and more body fluid cause mild leg swelling; elevating the legs helps, but report sudden severe swelling.'],
                        ['How do I count baby movements?', 'Count movements at a set time when the baby is active; a noticeable drop should be checked.'],
                    ]
                ),
            ],
            22 => [
                'fetal_development' => $this->bi(
                    'در هفته بیست‌ودوم جنین به اندازه یک پاپایا، حدود ۲۷.۸ سانتی‌متر و نزدیک به ۴۳۰ گرم است. حس لامسه در حال تکامل است و جنین ممکن است صورت و بند ناف را لمس کند. ابروها و مژه‌ها ظاهر شده‌اند.',
                    'At week twenty-two the fetus is about the size of a papaya, roughly 27.8 cm and nearly 430 g. The sense of touch is developing and the baby may touch its face and cord, while eyebrows and eyelashes have appeared.'
                ),
                'mother_body_changes' => $this->bi(
                    'شکم به‌وضوح بزرگ شده و حرکات جنین قوی‌تر است. ممکن است تورم خفیف، واریس یا حساسیت لثه داشته باشید. موها ممکن است پرپشت‌تر به نظر برسند.',
                    'The belly is clearly larger and movements are stronger. You may have mild swelling, varicose veins or gum sensitivity, and hair may look fuller.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و برای پیشگیری از واریس زیاد نایستید. بهداشت دهان را جدی بگیرید. فعالیت بدنی منظم داشته باشید.',
                    'Continue healthy eating and supplements and avoid prolonged standing to prevent varicose veins. Take oral hygiene seriously and stay active.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌های منظم را دنبال کنید و درباره آزمایش دیابت بارداری با پزشک هماهنگ کنید. شمارش حرکات جنین را ادامه دهید. برای کلاس‌های آمادگی زایمان برنامه‌ریزی کنید.',
                    'Keep regular visits and coordinate the gestational diabetes test with your doctor. Continue counting movements and plan for childbirth classes.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم بالای ناف است و ممکن است فشار بر عروق لگن باعث واریس یا بواسیر شود. حجم خون همچنان بالاست. بدن به‌خوبی این تغییرات را مدیریت می‌کند.',
                    'The uterus is above the navel and pelvic vein pressure may cause varicose veins or hemorrhoids. Blood volume remains high as the body manages these changes.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است بین هیجان و نگرانی درباره مسئولیت‌های پیش رو در نوسان باشید. ارتباط با جنین قوی‌تر می‌شود. صحبت با همسر و دوستان کمک‌کننده است.',
                    'You may swing between excitement and worry about upcoming responsibilities. The bond strengthens; talking with your partner and friends helps.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، فیبر و آب کافی برای پیشگیری از یبوست و بواسیر مهم هستند. پروتئین و سبزیجات را ادامه دهید. از غذاهای فرآوری‌شده و قند زیاد بپرهیزید.',
                    'Iron, calcium, fiber and enough water matter to prevent constipation and hemorrhoids. Keep protein and vegetables; limit processed foods and sugar.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و تمرینات ملایم به گردش خون کمک می‌کند و از واریس می‌کاهد. تمرینات کف لگن را ادامه دهید. از نشستن یا ایستادن طولانی بپرهیزید.',
                    'Walking, swimming and gentle exercise aid circulation and reduce varicose veins. Keep pelvic floor work and avoid sitting or standing too long.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت روتین شامل کنترل فشار خون، وزن و رشد جنین است. آزمایش تحمل گلوکز برای دیابت بارداری معمولاً در هفته‌های ۲۴ تا ۲۸ برنامه‌ریزی می‌شود. علائم غیرعادی را گزارش دهید.',
                    'Routine visits include blood pressure, weight and fetal growth. The glucose tolerance test for gestational diabetes is usually scheduled in weeks 24 to 28.'
                ),
                'faq' => $this->faq(
                    [
                        ['واریس بارداری خطرناک است؟', 'واریس معمولاً بی‌خطر است و اغلب پس از زایمان بهتر می‌شود؛ حرکت منظم و بالا گذاشتن پاها کمک می‌کند.'],
                        ['بواسیر در بارداری چرا رخ می‌دهد؟', 'فشار رحم و یبوست می‌تواند باعث بواسیر شود؛ فیبر، آب کافی و پرهیز از زور زدن کمک می‌کند.'],
                    ],
                    [
                        ['Are pregnancy varicose veins dangerous?', 'They are usually harmless and often improve after birth; regular movement and elevating the legs help.'],
                        ['Why do hemorrhoids happen in pregnancy?', 'Uterine pressure and constipation can cause them; fiber, water and avoiding straining help.'],
                    ]
                ),
            ],
            23 => [
                'fetal_development' => $this->bi(
                    'در هفته بیست‌وسوم جنین به اندازه یک گریپ‌فروت، حدود ۲۸.۹ سانتی‌متر و نزدیک به ۵۰۱ گرم است. ریه‌ها در حال تکامل رگ‌های خونی برای تنفس آینده هستند. جنین می‌تواند صداهای بیرون را بشنود و به آن‌ها واکنش نشان دهد.',
                    'At week twenty-three the fetus is about the size of a grapefruit, roughly 28.9 cm and nearly 501 g. The lungs are developing blood vessels for future breathing, and the baby can hear and respond to outside sounds.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است تورم پا و دست، سوزش سردل و کمردرد بیشتر شود. حرکات جنین قوی و قابل مشاهده است. پوست شکم ممکن است بیشتر کشیده و حساس شود.',
                    'Leg and hand swelling, heartburn and back pain may increase. Baby movements are strong and visible, and the belly skin may stretch and feel more tender.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و علائم فشار خون بالا مانند سردرد شدید را جدی بگیرید. پاها را بالا بگذارید و استراحت کنید. آب کافی بنوشید.',
                    'Continue healthy eating and supplements and take high-blood-pressure signs like severe headache seriously. Elevate your legs, rest, and drink enough water.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌های منظم را دنبال کنید و برای آزمایش دیابت بارداری آماده شوید. شمارش حرکات جنین را ادامه دهید. علائم زایمان زودرس را بشناسید و در صورت بروز فوراً اقدام کنید.',
                    'Keep regular visits and prepare for the diabetes test. Continue counting movements, learn preterm labor signs, and act quickly if they appear.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم بالای ناف است و ممکن است انقباضات تمرینی خفیف (براکستون هیکس) را حس کنید. فشار بر مثانه و معده بیشتر شده است. بدن برای مراحل بعد آماده می‌شود.',
                    'The uterus is above the navel and you may feel mild practice contractions (Braxton Hicks). Pressure on the bladder and stomach increases as the body prepares.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است با نزدیک شدن به سه‌ماهه سوم، افکار بیشتری درباره زایمان داشته باشید. کمی اضطراب طبیعی است. آموزش و آمادگی می‌تواند آرامش بیشتری بدهد.',
                    'As the third trimester nears you may think more about birth. Some anxiety is normal; education and preparation can bring more calm.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم و پروتئین را ادامه دهید و مصرف نمک را برای کنترل تورم متعادل کنید. فیبر و آب کافی برای گوارش مهم است. میان‌وعده‌های سالم بخورید.',
                    'Continue iron, calcium and protein and moderate salt to manage swelling. Fiber and enough water matter for digestion; choose healthy snacks.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و یوگای بارداری همچنان مفید هستند. تمرینات تنفسی برای آمادگی زایمان را تمرین کنید. از فعالیت‌های پرخطر و خستگی بیش از حد بپرهیزید.',
                    'Walking, swimming and prenatal yoga remain helpful. Practice breathing exercises for birth and avoid risky activities and overexertion.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت روتین شامل کنترل فشار خون، وزن و رشد جنین است. آزمایش تحمل گلوکز به‌زودی انجام می‌شود. در صورت وجود ریسک زایمان زودرس، پزشک ممکن است بررسی‌های بیشتری توصیه کند.',
                    'Routine visits include blood pressure, weight and fetal growth. The glucose tolerance test is done soon; if preterm risk exists, the doctor may advise more checks.'
                ),
                'faq' => $this->faq(
                    [
                        ['انقباضات براکستون هیکس چیست؟', 'انقباضات تمرینی و بی‌درد رحم هستند که بدن را برای زایمان آماده می‌کنند؛ اگر منظم و دردناک شدند به پزشک مراجعه کنید.'],
                        ['نشانه‌های زایمان زودرس چیست؟', 'انقباضات منظم، فشار لگنی، کمردرد مداوم یا نشت مایع می‌تواند نشانه باشد و نیاز به مراجعه فوری دارد.'],
                    ],
                    [
                        ['What are Braxton Hicks contractions?', 'They are painless practice contractions preparing the body for labor; see a doctor if they become regular and painful.'],
                        ['What are signs of preterm labor?', 'Regular contractions, pelvic pressure, constant back pain or fluid leakage can be signs needing urgent care.'],
                    ]
                ),
            ],
            24 => [
                'fetal_development' => $this->bi(
                    'در هفته بیست‌وچهارم جنین به اندازه یک بلال، حدود ۳۰ سانتی‌متر و نزدیک به ۶۰۰ گرم است. این هفته مرز حیات‌پذیری در نظر گرفته می‌شود و ریه‌ها ماده سورفکتانت تولید می‌کنند. پوست جنین هنوز چروکیده است اما چربی در حال افزایش است.',
                    'At week twenty-four the fetus is about the size of an ear of corn, roughly 30 cm and nearly 600 g. This week marks the threshold of viability and the lungs produce surfactant, while the skin is still wrinkled but fat is increasing.'
                ),
                'mother_body_changes' => $this->bi(
                    'شکم بزرگ‌تر شده و ممکن است سوزش سردل، تورم و کمردرد بیشتر شود. حرکات جنین قوی است. ممکن است پوست شکم بیشتر بکشد و خارش داشته باشید.',
                    'The belly is larger and heartburn, swelling and back pain may increase. Movements are strong, and the belly skin may stretch more and itch.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و آزمایش دیابت بارداری را به‌موقع انجام دهید. از افزایش وزن ناگهانی پرهیز کنید. علائم فشار خون بالا را جدی بگیرید.',
                    'Continue healthy eating and supplements and do the diabetes test on time. Avoid sudden weight gain and take high-blood-pressure signs seriously.'
                ),
                'care_plan' => $this->bi(
                    'آزمایش تحمل گلوکز برای غربالگری دیابت بارداری از این هفته انجام می‌شود؛ آن را انجام دهید. ویزیت‌های منظم و شمارش حرکات جنین را ادامه دهید. علائم هشدار را بشناسید.',
                    'The glucose tolerance test to screen for gestational diabetes is done from this week; complete it. Keep regular visits and movement counts, and know warning signs.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم به بالای ناف رسیده و فشار بر دیافراگم ممکن است تنفس را کمی سخت‌تر کند. بدن برای پشتیبانی از رشد سریع جنین سخت کار می‌کند. این تغییرات طبیعی هستند.',
                    'The uterus is above the navel and pressure on the diaphragm may make breathing a bit harder. The body works hard to support the baby’s rapid growth.'
                ),
                'emotional_status' => $this->bi(
                    'با ورود به مرحله حیات‌پذیری ممکن است حس اطمینان بیشتری داشته باشید. با این حال، فکر کردن به زایمان می‌تواند اضطراب‌آور باشد. آموزش و حمایت کمک می‌کند.',
                    'Reaching viability may bring more reassurance. Still, thinking about birth can cause anxiety; education and support help.'
                ),
                'key_nutrition' => $this->bi(
                    'برای کنترل قند خون، کربوهیدرات‌های پیچیده و پروتئین را جایگزین قند ساده کنید. آهن، کلسیم و فیبر را ادامه دهید. آب کافی بنوشید و وعده‌ها را متعادل نگه دارید.',
                    'To manage blood sugar, replace simple sugars with complex carbs and protein. Continue iron, calcium and fiber, drink enough water, and keep balanced meals.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی و شنا به کنترل قند خون و کاهش تورم کمک می‌کند. تمرینات کف لگن و تنفسی را ادامه دهید. از فعالیت‌های شدید و پرخطر بپرهیزید.',
                    'Walking and swimming help control blood sugar and reduce swelling. Keep pelvic floor and breathing exercises and avoid intense or risky activities.'
                ),
                'tests_and_checkups' => $this->bi(
                    'آزمایش تحمل گلوکز (GTT) برای غربالگری دیابت بارداری در بازه هفته ۲۴ تا ۲۸ انجام می‌شود. ویزیت روتین شامل فشار خون، وزن و رشد جنین است. علائم غیرعادی را گزارش دهید.',
                    'The glucose tolerance test (GTT) for gestational diabetes screening is done in the weeks 24 to 28 window. Routine visits include blood pressure, weight and fetal growth.'
                ),
                'faq' => $this->faq(
                    [
                        ['آزمایش دیابت بارداری چگونه است؟', 'محلول قندی می‌نوشید و سطح قند خون در فواصل مشخص اندازه‌گیری می‌شود تا نحوه پردازش قند توسط بدن بررسی شود.'],
                        ['حیات‌پذیری جنین یعنی چه؟', 'یعنی از این هفته جنین با مراقبت‌های ویژه پزشکی شانس زنده ماندن خارج از رحم را دارد، هرچند رسیدن به ترم بسیار مهم‌تر است.'],
                    ],
                    [
                        ['How does the diabetes test work?', 'You drink a glucose solution and blood sugar is measured at intervals to see how your body processes sugar.'],
                        ['What does fetal viability mean?', 'From this week the baby has a chance of surviving outside the womb with intensive care, though reaching term is far better.'],
                    ]
                ),
            ],
            25 => [
                'fetal_development' => $this->bi(
                    'در هفته بیست‌وپنجم جنین به اندازه یک گل‌کلم، حدود ۳۴.۶ سانتی‌متر و نزدیک به ۶۶۰ گرم است. پوست صاف‌تر و صورتی‌تر می‌شود و چربی زیر پوست افزایش می‌یابد. جنین شروع به تجربه رفلکس گرفتن با دست می‌کند.',
                    'At week twenty-five the fetus is about the size of a cauliflower, roughly 34.6 cm and nearly 660 g. The skin becomes smoother and pinker as subcutaneous fat increases, and the baby develops a grasp reflex.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است کمردرد، سوزش سردل و تورم بیشتر شود و انرژی کمی کاهش یابد. حرکات جنین قوی و منظم است. ممکن است دچار سندرم تونل کارپال خفیف در دست شوید.',
                    'Back pain, heartburn and swelling may increase and energy dip slightly. Movements are strong and regular, and you may develop mild carpal tunnel in the hands.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و اگر دیابت بارداری دارید رژیم تجویزی را رعایت کنید. از فشار طولانی بر مچ دست بپرهیزید. استراحت کافی داشته باشید.',
                    'Continue healthy eating and supplements and follow any prescribed diet if you have gestational diabetes. Avoid prolonged wrist strain and rest enough.'
                ),
                'care_plan' => $this->bi(
                    'اگر آزمایش دیابت را نداده‌اید، این هفته انجام دهید. ویزیت‌های منظم و شمارش حرکات جنین را ادامه دهید. برای کلاس‌های آمادگی زایمان ثبت‌نام کنید.',
                    'If you haven’t done the diabetes test, do it this week. Keep regular visits and movement counts, and enroll in childbirth classes.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم بالاتر آمده و فشار بر معده و ریه‌ها بیشتر شده است. احتباس مایعات می‌تواند باعث تورم و بی‌حسی دست شود. بدن با این تغییرات سازگار می‌شود.',
                    'The uterus is higher and pressure on the stomach and lungs increases. Fluid retention can cause swelling and hand numbness as the body adapts.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است هیجان و کمی خستگی روانی را همزمان تجربه کنید. فکر کردن به آماده‌سازی نوزاد شایع است. استراحت و مراقبت از خود را در اولویت بگذارید.',
                    'You may feel both excitement and some mental fatigue. Thinking about preparing for the baby is common; prioritize rest and self-care.'
                ),
                'key_nutrition' => $this->bi(
                    'پروتئین، آهن، کلسیم و فیبر را ادامه دهید و قند ساده را محدود کنید. غذاهای غنی از منیزیم به کاهش گرفتگی کمک می‌کند. آب کافی بنوشید.',
                    'Continue protein, iron, calcium and fiber and limit simple sugars. Magnesium-rich foods help reduce cramps; drink enough water.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و کشش ملایم به کاهش کمردرد و بهبود خواب کمک می‌کند. تمرینات کف لگن را ادامه دهید. حرکات مچ دست را برای کاهش بی‌حسی انجام دهید.',
                    'Walking, swimming and gentle stretching ease back pain and improve sleep. Keep pelvic floor exercises and do wrist movements to reduce numbness.'
                ),
                'tests_and_checkups' => $this->bi(
                    'آزمایش تحمل گلوکز همچنان در این بازه قابل انجام است. ویزیت روتین شامل فشار خون، وزن و رشد جنین است. در صورت کم‌خونی، مکمل آهن ممکن است تنظیم شود.',
                    'The glucose tolerance test is still available now. Routine visits include blood pressure, weight and fetal growth, and iron may be adjusted if anemic.'
                ),
                'faq' => $this->faq(
                    [
                        ['بی‌حسی دست در بارداری طبیعی است؟', 'بله، احتباس مایعات می‌تواند باعث سندرم تونل کارپال خفیف شود که معمولاً پس از زایمان بهتر می‌شود.'],
                        ['اگر دیابت بارداری داشته باشم چه کنم؟', 'با رعایت رژیم غذایی، فعالیت بدنی و گاهی دارو، قند خون کنترل می‌شود؛ پزشک برنامه مناسب را تعیین می‌کند.'],
                    ],
                    [
                        ['Is hand numbness normal in pregnancy?', 'Yes, fluid retention can cause mild carpal tunnel, which usually improves after birth.'],
                        ['What if I have gestational diabetes?', 'It is managed with diet, activity and sometimes medication; your doctor sets the right plan.'],
                    ]
                ),
            ],
            26 => [
                'fetal_development' => $this->bi(
                    'در هفته بیست‌وششم جنین به اندازه یک کاهو، حدود ۳۵.۶ سانتی‌متر و نزدیک به ۷۶۰ گرم است. چشم‌ها شروع به باز شدن می‌کنند و جنین به نور واکنش نشان می‌دهد. ریه‌ها به تمرین حرکات تنفسی ادامه می‌دهند.',
                    'At week twenty-six the fetus is about the size of a head of lettuce, roughly 35.6 cm and nearly 760 g. The eyes begin to open and the baby responds to light, while the lungs keep practicing breathing movements.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است فشار خون، تورم و کمردرد بیشتر شود و خواب سخت‌تر شود. حرکات جنین قوی و گاهی همراه با سکسکه است. ممکن است انقباضات تمرینی را حس کنید.',
                    'Blood pressure, swelling and back pain may rise and sleep becomes harder. Movements are strong, sometimes with hiccups, and you may feel practice contractions.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و علائم پره‌اکلامپسی مانند سردرد شدید و تاری دید را جدی بگیرید. به پهلو بخوابید. آب کافی بنوشید.',
                    'Continue healthy eating and supplements and take pre-eclampsia signs like severe headache and blurred vision seriously. Sleep on your side and hydrate.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌های منظم را دنبال کنید و اگر آزمایش دیابت را نداده‌اید انجام دهید. شمارش حرکات جنین را ادامه دهید. برای شروع سه‌ماهه سوم برنامه‌ریزی کنید.',
                    'Keep regular visits and do the diabetes test if not done. Continue counting movements and plan for the start of the third trimester.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم بالای ناف است و فشار بر معده و مثانه بیشتر شده است. احتباس مایعات ادامه دارد. بدن با آماده شدن برای سه‌ماهه سوم سازگار می‌شود.',
                    'The uterus is above the navel and pressure on the stomach and bladder increases. Fluid retention continues as the body prepares for the third trimester.'
                ),
                'emotional_status' => $this->bi(
                    'با نزدیک شدن به سه‌ماهه سوم ممکن است ترکیبی از هیجان و اضطراب داشته باشید. خواب ناکافی می‌تواند بر خلق‌وخو اثر بگذارد. استراحت و حمایت اهمیت دارد.',
                    'As the third trimester nears you may feel a mix of excitement and anxiety. Poor sleep can affect mood; rest and support matter.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم و پروتئین را ادامه دهید و اگر دیابت دارید قند را کنترل کنید. غذاهای غنی از فیبر و پتاسیم مفید هستند. آب کافی بنوشید و نمک را متعادل کنید.',
                    'Continue iron, calcium and protein and control sugar if diabetic. Fiber- and potassium-rich foods help; drink enough water and moderate salt.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و یوگای بارداری همچنان مفید است. تمرینات تنفسی و کف لگن را ادامه دهید. از فعالیت‌های سنگین و خستگی بیش از حد بپرهیزید.',
                    'Walking, swimming and prenatal yoga remain helpful. Keep breathing and pelvic floor exercises and avoid heavy activity and overexertion.'
                ),
                'tests_and_checkups' => $this->bi(
                    'آزمایش تحمل گلوکز همچنان در این بازه انجام‌شدنی است و فشار خون برای بررسی پره‌اکلامپسی کنترل می‌شود. ویزیت روتین شامل وزن و رشد جنین است. علائم هشدار را گزارش دهید.',
                    'The glucose tolerance test is still possible now and blood pressure is checked for pre-eclampsia. Routine visits include weight and fetal growth.'
                ),
                'faq' => $this->faq(
                    [
                        ['علائم پره‌اکلامپسی چیست؟', 'فشار خون بالا، سردرد شدید، تاری دید، درد بالای شکم و ورم ناگهانی از نشانه‌هاست و نیاز به مراجعه فوری دارد.'],
                        ['سکسکه جنین طبیعی است؟', 'بله، سکسکه جنین که به‌صورت حرکات ریتمیک حس می‌شود کاملاً طبیعی و نشانه رشد سالم است.'],
                    ],
                    [
                        ['What are pre-eclampsia signs?', 'High blood pressure, severe headache, blurred vision, upper-abdomen pain and sudden swelling are signs needing urgent care.'],
                        ['Are fetal hiccups normal?', 'Yes, fetal hiccups felt as rhythmic movements are entirely normal and a sign of healthy development.'],
                    ]
                ),
            ],
            27 => [
                'fetal_development' => $this->bi(
                    'در هفته بیست‌وهفتم که پایان سه‌ماهه دوم است، جنین حدود ۳۶.۶ سانتی‌متر و نزدیک به ۸۷۵ گرم است. مغز به‌سرعت رشد می‌کند و جنین الگوهای منظم خواب و بیداری دارد. ریه‌ها به بلوغ خود ادامه می‌دهند.',
                    'At week twenty-seven, the end of the second trimester, the fetus is about 36.6 cm and nearly 875 g. The brain grows rapidly, the baby has regular sleep-wake patterns, and the lungs keep maturing.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است کمردرد، تنگی نفس و گرفتگی عضلات پا بیشتر شود. حرکات جنین قوی و منظم است. ممکن است خواب به دلیل بزرگی شکم دشوارتر شود.',
                    'Back pain, shortness of breath and leg cramps may increase. Movements are strong and regular, and sleep may be harder due to the larger belly.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و برای خواب بهتر از بالش بارداری استفاده کنید. علائم زایمان زودرس را بشناسید. به پهلو بخوابید و آب کافی بنوشید.',
                    'Continue healthy eating and supplements and use a pregnancy pillow for better sleep. Know preterm labor signs, sleep on your side, and hydrate.'
                ),
                'care_plan' => $this->bi(
                    'با ورود به سه‌ماهه سوم، ویزیت‌ها ممکن است مکرر‌تر شوند. شمارش حرکات جنین را جدی بگیرید. برای کلاس‌های آمادگی زایمان و تهیه لوازم نوزاد برنامه‌ریزی کنید.',
                    'Entering the third trimester, visits may become more frequent. Take movement counting seriously and plan childbirth classes and baby supplies.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم بالای ناف رسیده و فشار بر دیافراگم تنفس را کمی سخت‌تر می‌کند. مفاصل و رباط‌ها شل‌تر هستند. بدن برای مراحل نهایی بارداری آماده می‌شود.',
                    'The uterus is above the navel and pressure on the diaphragm makes breathing a bit harder. Joints and ligaments are looser as the body prepares for the final stages.'
                ),
                'emotional_status' => $this->bi(
                    'با نزدیک شدن به مراحل پایانی، افکار درباره زایمان و والدگری بیشتر می‌شود. کمی اضطراب یا هیجان طبیعی است. حمایت عاطفی و آموزش کمک‌کننده است.',
                    'As the final stages approach, thoughts about birth and parenting grow. Some anxiety or excitement is normal; emotional support and education help.'
                ),
                'key_nutrition' => $this->bi(
                    'پروتئین، آهن، کلسیم و امگا-۳ برای رشد مغز جنین مهم هستند. فیبر و آب کافی از یبوست جلوگیری می‌کند. از قند و نمک زیاد بپرهیزید.',
                    'Protein, iron, calcium and omega-3 matter for the baby’s brain growth. Fiber and enough water prevent constipation; limit excess sugar and salt.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و کشش ملایم به کاهش دردها و بهبود خواب کمک می‌کند. تمرینات تنفسی و کف لگن را ادامه دهید. به بدن خود گوش دهید و افراط نکنید.',
                    'Walking, swimming and gentle stretching ease aches and improve sleep. Keep breathing and pelvic floor exercises; listen to your body and don’t overdo it.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت روتین شامل فشار خون، وزن و رشد جنین است و ممکن است نتایج آزمایش دیابت بررسی شود. برای مادران Rh منفی، تزریق آنتی‌دی (روگام) به‌زودی مطرح می‌شود. علائم هشدار را گزارش دهید.',
                    'Routine visits include blood pressure, weight and fetal growth, and diabetes test results may be reviewed. For Rh-negative mothers, anti-D (RhoGAM) will be discussed soon.'
                ),
                'faq' => $this->faq(
                    [
                        ['چطور بهتر بخوابم؟', 'خوابیدن به پهلوی چپ، استفاده از بالش بین پاها و پرهیز از غذای سنگین قبل از خواب کمک می‌کند.'],
                        ['چرا نفسم سنگین شده است؟', 'فشار رحم بر دیافراگم باعث تنگی نفس خفیف می‌شود که طبیعی است؛ اما تنگی نفس ناگهانی و شدید باید بررسی شود.'],
                    ],
                    [
                        ['How can I sleep better?', 'Sleeping on your left side, a pillow between the legs and avoiding heavy meals before bed help.'],
                        ['Why is my breathing heavier?', 'Uterine pressure on the diaphragm causes mild breathlessness, which is normal; sudden severe breathlessness should be checked.'],
                    ]
                ),
            ],
            28 => [
                'fetal_development' => $this->bi(
                    'در هفته بیست‌وهشتم که آغاز سه‌ماهه سوم است، جنین به اندازه یک بادمجان، حدود ۳۷.۶ سانتی‌متر و نزدیک به ۱ کیلوگرم است. جنین می‌تواند پلک بزند و رویا ببیند و مغز شیارهای بیشتری پیدا می‌کند. چشم‌ها اکنون باز و بسته می‌شوند.',
                    'At week twenty-eight, the start of the third trimester, the fetus is about the size of an eggplant, roughly 37.6 cm and nearly 1 kg. The baby can blink and dream, the brain develops more folds, and the eyes now open and close.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است تنگی نفس، سوزش سردل، تورم و کمردرد بیشتر شود. حرکات جنین قوی و منظم است و باید پیگیری شود. ممکن است خستگی سه‌ماهه سوم بازگردد.',
                    'Shortness of breath, heartburn, swelling and back pain may increase. Movements are strong and regular and should be tracked, and third-trimester fatigue may return.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و شمارش حرکات جنین را جدی بگیرید. علائم پره‌اکلامپسی و زایمان زودرس را بشناسید. به پهلو بخوابید و استراحت کافی داشته باشید.',
                    'Continue healthy eating and supplements and take movement counting seriously. Know pre-eclampsia and preterm labor signs, sleep on your side, and rest enough.'
                ),
                'care_plan' => $this->bi(
                    'در سه‌ماهه سوم ویزیت‌ها معمولاً هر دو هفته یک‌بار می‌شود. برای مادران Rh منفی، تزریق آنتی‌دی در این هفته توصیه می‌شود. برنامه زایمان و لوازم نوزاد را آماده کنید.',
                    'In the third trimester visits usually become every two weeks. For Rh-negative mothers, an anti-D injection is recommended this week; prepare your birth plan and baby supplies.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم به‌سرعت بزرگ می‌شود و فشار بر اندام‌های داخلی بیشتر می‌شود. بدن مایعات بیشتری نگه می‌دارد. سیستم قلبی‌عروقی با اوج حجم خون کار می‌کند.',
                    'The uterus grows quickly and presses more on internal organs. The body retains more fluid and the cardiovascular system works at peak blood volume.'
                ),
                'emotional_status' => $this->bi(
                    'ورود به مرحله نهایی می‌تواند هیجان و اضطراب درباره زایمان را افزایش دهد. فکر کردن به نقش والدگری شایع است. مراقبت از سلامت روان بسیار مهم است.',
                    'Entering the final stage can heighten excitement and anxiety about birth. Thinking about the parenting role is common; mental health care is very important.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، پروتئین و امگا-۳ برای رشد جنین ضروری هستند. فیبر و آب کافی از یبوست جلوگیری می‌کند. وعده‌های کوچک و مکرر سوزش سردل را کاهش می‌دهد.',
                    'Iron, calcium, protein and omega-3 are essential for the baby’s growth. Fiber and enough water prevent constipation, and small frequent meals reduce heartburn.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و تمرینات ملایم به گردش خون و آمادگی زایمان کمک می‌کند. تمرینات کف لگن و تنفسی را ادامه دهید. از فعالیت‌های پرخطر بپرهیزید.',
                    'Walking, swimming and gentle exercise aid circulation and birth readiness. Keep pelvic floor and breathing exercises and avoid risky activities.'
                ),
                'tests_and_checkups' => $this->bi(
                    'برای مادران Rh منفی، تزریق ایمونوگلوبولین آنتی‌دی (روگام) در هفته ۲۸ برای پیشگیری از حساسیت خونی انجام می‌شود. آزمایش خون برای کم‌خونی و کنترل قند نیز ممکن است تکرار شود. ویزیت روتین ادامه دارد.',
                    'For Rh-negative mothers, an anti-D immunoglobulin (RhoGAM) injection is given at week 28 to prevent blood sensitization. Blood tests for anemia and sugar may be repeated.'
                ),
                'faq' => $this->faq(
                    [
                        ['تزریق روگام (آنتی‌دی) برای چیست؟', 'اگر خون مادر Rh منفی و جنین Rh مثبت باشد، این تزریق از ساخته شدن آنتی‌بادی خطرناک علیه خون جنین جلوگیری می‌کند.'],
                        ['چند بار در سه‌ماهه سوم به پزشک مراجعه کنم؟', 'معمولاً تا هفته ۳۶ هر دو هفته یک‌بار و پس از آن هفتگی، مگر اینکه پزشک برنامه دیگری تعیین کند.'],
                    ],
                    [
                        ['What is the RhoGAM (anti-D) injection for?', 'If the mother is Rh-negative and the baby Rh-positive, it prevents harmful antibodies from forming against the baby’s blood.'],
                        ['How often should I visit in the third trimester?', 'Usually every two weeks until week 36 and weekly after, unless your doctor sets a different plan.'],
                    ]
                ),
            ],
            29 => [
                'fetal_development' => $this->bi(
                    'در هفته بیست‌ونهم جنین به اندازه یک کدو حلوایی، حدود ۳۸.۶ سانتی‌متر و نزدیک به ۱.۲ کیلوگرم است. عضلات و ریه‌ها در حال بلوغ هستند و مغز به‌سرعت رشد می‌کند. جنین اکنون می‌تواند دمای بدن خود را تا حدی تنظیم کند.',
                    'At week twenty-nine the fetus is about the size of a butternut squash, roughly 38.6 cm and nearly 1.2 kg. Muscles and lungs are maturing, the brain grows fast, and the baby can now partly regulate its own temperature.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است تنگی نفس، سوزش سردل، بواسیر و گرفتگی عضلات بیشتر شود. حرکات جنین قوی و گاهی دردناک است. ممکن است خستگی و مشکلات خواب داشته باشید.',
                    'Shortness of breath, heartburn, hemorrhoids and cramps may increase. Movements are strong and sometimes uncomfortable, and you may have fatigue and sleep issues.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و برای بواسیر فیبر و آب کافی مصرف کنید. شمارش حرکات جنین را جدی بگیرید. از ایستادن طولانی بپرهیزید و استراحت کنید.',
                    'Continue healthy eating and supplements and eat enough fiber and water for hemorrhoids. Take movement counting seriously, avoid long standing, and rest.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌های منظم را دنبال کنید و درباره برنامه زایمان با پزشک صحبت کنید. اگر Rh منفی هستید و تزریق آنتی‌دی نشده، پیگیری کنید. برای تهیه ساک زایمان برنامه‌ریزی کنید.',
                    'Keep regular visits and discuss your birth plan with your doctor. If Rh-negative and anti-D not given, follow up, and plan to prepare your hospital bag.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم به‌سرعت بزرگ می‌شود و فشار بر معده، ریه‌ها و مثانه بیشتر است. احتباس مایعات ممکن است تورم را افزایش دهد. بدن برای هفته‌های پایانی آماده می‌شود.',
                    'The uterus grows fast and presses more on the stomach, lungs and bladder. Fluid retention may increase swelling as the body prepares for the final weeks.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است هیجان دیدن نوزاد با نگرانی درباره زایمان همراه باشد. خستگی می‌تواند بر خلق‌وخو اثر بگذارد. صحبت درباره احساسات و استراحت کمک‌کننده است.',
                    'Excitement to meet the baby may mix with worry about birth. Fatigue can affect mood; sharing feelings and resting help.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، پروتئین و فیبر را ادامه دهید و برای گرفتگی عضلات منیزیم و پتاسیم مصرف کنید. آب کافی بنوشید. وعده‌های کوچک سوزش سردل را کاهش می‌دهد.',
                    'Continue iron, calcium, protein and fiber and take magnesium and potassium for cramps. Drink enough water; small meals reduce heartburn.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و کشش ملایم به کاهش دردها و بهبود خواب کمک می‌کند. تمرینات کف لگن و تنفسی را ادامه دهید. به بدن خود گوش دهید.',
                    'Walking, swimming and gentle stretching ease aches and improve sleep. Keep pelvic floor and breathing exercises and listen to your body.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت روتین شامل فشار خون، وزن، رشد جنین و بررسی وضعیت قرارگیری جنین است. آزمایش خون برای کم‌خونی ممکن است تکرار شود. علائم پره‌اکلامپسی و زایمان زودرس را گزارش دهید.',
                    'Routine visits include blood pressure, weight, fetal growth and baby’s position. A blood test for anemia may be repeated; report pre-eclampsia and preterm labor signs.'
                ),
                'faq' => $this->faq(
                    [
                        ['چه زمانی ساک زایمان را آماده کنم؟', 'بهتر است تا حدود هفته ۳۲ تا ۳۵ ساک زایمان را آماده کنید تا در صورت زایمان زودتر آماده باشید.'],
                        ['حرکات دردناک جنین طبیعی است؟', 'بله، با بزرگ‌تر شدن جنین لگدها ممکن است محکم‌تر و گاهی ناراحت‌کننده باشد که طبیعی است.'],
                    ],
                    [
                        ['When should I pack my hospital bag?', 'It is best to pack it by around weeks 32 to 35 so you are ready in case of earlier labor.'],
                        ['Are uncomfortable baby movements normal?', 'Yes, as the baby grows, kicks may feel firmer and sometimes uncomfortable, which is normal.'],
                    ]
                ),
            ],
            30 => [
                'fetal_development' => $this->bi(
                    'در هفته سی‌ام جنین به اندازه یک کلم، حدود ۳۹.۹ سانتی‌متر و نزدیک به ۱.۳ کیلوگرم است. مغز شیارها و چین‌های بیشتری پیدا می‌کند و مغز استخوان تولید گلبول قرمز را بر عهده می‌گیرد. کرک لانوگو کم‌کم ناپدید می‌شود.',
                    'At week thirty the fetus is about the size of a cabbage, roughly 39.9 cm and nearly 1.3 kg. The brain develops more grooves and folds, the bone marrow takes over red blood cell production, and the lanugo begins to disappear.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است تنگی نفس، کمردرد، تورم و مشکلات خواب بیشتر شود. حرکات جنین قوی است. ممکن است انقباضات تمرینی بیشتری را حس کنید.',
                    'Shortness of breath, back pain, swelling and sleep issues may increase. Movements are strong, and you may feel more practice contractions.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و برای خواب بهتر از بالش استفاده کنید. شمارش حرکات جنین را جدی بگیرید. علائم زایمان زودرس را بشناسید.',
                    'Continue healthy eating and supplements and use pillows for better sleep. Take movement counting seriously and know preterm labor signs.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌های منظم را دنبال کنید و برنامه زایمان خود را نهایی کنید. برای کلاس‌های آمادگی زایمان و شیردهی ثبت‌نام کنید. ساک زایمان را کم‌کم آماده کنید.',
                    'Keep regular visits and finalize your birth plan. Enroll in childbirth and breastfeeding classes and start preparing your hospital bag.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم بالای ناف است و فشار بر دیافراگم و مثانه بیشتر می‌شود. مفاصل لگن برای زایمان کم‌کم شل‌تر می‌شوند. بدن به‌خوبی خود را برای مراحل نهایی آماده می‌کند.',
                    'The uterus is above the navel and pressure on the diaphragm and bladder increases. Pelvic joints loosen for birth as the body prepares for the final stages.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است انتظار برای دیدن نوزاد با خستگی و اضطراب همراه باشد. فکر کردن به زایمان طبیعی است. آموزش و حمایت به کاهش نگرانی کمک می‌کند.',
                    'Anticipation to meet the baby may mix with fatigue and anxiety. Thinking about birth is natural; education and support reduce worry.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، پروتئین و امگا-۳ را ادامه دهید و فیبر کافی مصرف کنید. آب کافی از یبوست و تورم جلوگیری می‌کند. از قند و نمک زیاد بپرهیزید.',
                    'Continue iron, calcium, protein and omega-3 and eat enough fiber. Enough water helps prevent constipation and swelling; limit excess sugar and salt.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و یوگای بارداری به آمادگی زایمان کمک می‌کند. تمرینات کف لگن و تنفسی را ادامه دهید. از فعالیت‌های سنگین بپرهیزید و استراحت کنید.',
                    'Walking, swimming and prenatal yoga aid birth readiness. Keep pelvic floor and breathing exercises, avoid heavy activity, and rest.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت روتین شامل فشار خون، وزن، رشد و وضعیت قرارگیری جنین است. پزشک ممکن است در صورت نیاز سونوگرافی رشد را توصیه کند. علائم هشدار را گزارش دهید.',
                    'Routine visits include blood pressure, weight, growth and the baby’s position. Your doctor may advise a growth ultrasound if needed; report warning signs.'
                ),
                'faq' => $this->faq(
                    [
                        ['کلاس‌های آمادگی زایمان مفیدند؟', 'بله، این کلاس‌ها درباره مراحل زایمان، تکنیک‌های تنفس و مراقبت از نوزاد آموزش می‌دهند و اعتماد‌به‌نفس شما را بالا می‌برند.'],
                        ['وضعیت قرارگیری جنین مهم است؟', 'در این هفته هنوز جنین ممکن است بچرخد؛ وضعیت نهایی معمولاً در هفته‌های بعد اهمیت بیشتری پیدا می‌کند.'],
                    ],
                    [
                        ['Are childbirth classes helpful?', 'Yes, they teach the stages of labor, breathing techniques and newborn care and boost your confidence.'],
                        ['Does the baby’s position matter now?', 'The baby may still turn at this week; the final position becomes more important in later weeks.'],
                    ]
                ),
            ],
        ];
    }

    /** @return array<int, array<string, array>> */
    private function weeks31to40(): array
    {
        return [
            31 => [
                'fetal_development' => $this->bi(
                    'در هفته سی‌ویکم جنین به اندازه یک نارگیل، حدود ۴۱.۱ سانتی‌متر و نزدیک به ۱.۵ کیلوگرم است. مغز با سرعت زیادی در حال تکامل ارتباطات عصبی است و جنین می‌تواند سر خود را بچرخاند. ریه‌ها به بلوغ ادامه می‌دهند.',
                    'At week thirty-one the fetus is about the size of a coconut, roughly 41.1 cm and nearly 1.5 kg. The brain is rapidly forming neural connections, the baby can turn its head, and the lungs keep maturing.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است تنگی نفس، کمردرد، انقباضات تمرینی و نشت خفیف کلستروم از سینه را تجربه کنید. حرکات جنین قوی است. خواب ممکن است دشوارتر شود.',
                    'You may experience shortness of breath, back pain, practice contractions and slight colostrum leakage from the breasts. Movements are strong and sleep may be harder.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و برای خواب بهتر به پهلو بخوابید. شمارش حرکات جنین را جدی بگیرید. علائم زایمان زودرس و پره‌اکلامپسی را بشناسید.',
                    'Continue healthy eating and supplements and sleep on your side for better rest. Take movement counting seriously and know preterm labor and pre-eclampsia signs.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌های منظم را دنبال کنید و برنامه زایمان را با پزشک مرور کنید. ساک زایمان را آماده کنید. برای مراقبت از نوزاد و شیردهی اطلاعات کسب کنید.',
                    'Keep regular visits and review your birth plan with your doctor. Prepare your hospital bag and learn about newborn care and breastfeeding.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم بالای ناف رسیده و فشار بر ریه‌ها و مثانه بیشتر شده است. سینه‌ها برای شیردهی آماده می‌شوند و کلستروم تولید می‌کنند. بدن برای زایمان آماده می‌شود.',
                    'The uterus is above the navel and pressure on the lungs and bladder increases. The breasts prepare for feeding by producing colostrum as the body readies for birth.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است هیجان و کمی اضطراب درباره نزدیک شدن زایمان داشته باشید. خستگی می‌تواند بر خلق‌وخو اثر بگذارد. آموزش و حمایت آرامش می‌آورد.',
                    'You may feel excitement and some anxiety about the approaching birth. Fatigue can affect mood; education and support bring calm.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، پروتئین و امگا-۳ را ادامه دهید و فیبر کافی مصرف کنید. آب کافی از تورم و یبوست جلوگیری می‌کند. وعده‌های کوچک سوزش سردل را کاهش می‌دهد.',
                    'Continue iron, calcium, protein and omega-3 and eat enough fiber. Enough water helps prevent swelling and constipation, and small meals reduce heartburn.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و کشش ملایم به کاهش دردها کمک می‌کند. تمرینات کف لگن و تنفسی را برای زایمان تمرین کنید. از فعالیت‌های سنگین بپرهیزید.',
                    'Walking, swimming and gentle stretching ease aches. Practice pelvic floor and breathing exercises for birth and avoid heavy activity.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت روتین شامل فشار خون، وزن، رشد و وضعیت جنین است. پزشک ممکن است سونوگرافی رشد یا بررسی مایع آمنیوتیک را توصیه کند. علائم هشدار را گزارش دهید.',
                    'Routine visits include blood pressure, weight, growth and the baby’s position. Your doctor may advise a growth ultrasound or amniotic fluid check; report warning signs.'
                ),
                'faq' => $this->faq(
                    [
                        ['نشت کلستروم از سینه طبیعی است؟', 'بله، تراوش مایع زردرنگ کلستروم در اواخر بارداری طبیعی است و نشانه آماده شدن بدن برای شیردهی است.'],
                        ['چطور با بی‌خوابی کنار بیایم؟', 'خوابیدن به پهلو با بالش حمایتی، کاهش مایعات قبل از خواب و روتین آرام‌بخش شبانه کمک می‌کند.'],
                    ],
                    [
                        ['Is colostrum leakage normal?', 'Yes, leaking yellowish colostrum in late pregnancy is normal and shows the body is preparing to breastfeed.'],
                        ['How can I cope with insomnia?', 'Side-sleeping with a support pillow, fewer fluids before bed and a calming night routine help.'],
                    ]
                ),
            ],
            32 => [
                'fetal_development' => $this->bi(
                    'در هفته سی‌ودوم جنین به اندازه یک کدو، حدود ۴۲.۴ سانتی‌متر و نزدیک به ۱.۷ کیلوگرم است. ناخن‌ها کامل شده و جنین چربی بیشتری زیر پوست ذخیره می‌کند. بسیاری از جنین‌ها در این هفته رو به پایین قرار می‌گیرند.',
                    'At week thirty-two the fetus is about the size of a squash, roughly 42.4 cm and nearly 1.7 kg. The nails are complete and more fat is stored under the skin, and many babies settle head-down this week.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است تنگی نفس، سوزش سردل، تورم و انقباضات تمرینی بیشتر شود. حرکات جنین قوی اما فضای کمتری دارد. خستگی سه‌ماهه سوم ادامه دارد.',
                    'Shortness of breath, heartburn, swelling and practice contractions may increase. Movements are strong but with less room, and third-trimester fatigue continues.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و شمارش حرکات جنین را جدی بگیرید. علائم زایمان زودرس را بشناسید. به پهلو بخوابید و از خستگی زیاد پرهیز کنید.',
                    'Continue healthy eating and supplements and take movement counting seriously. Know preterm labor signs, sleep on your side, and avoid overexertion.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌ها معمولاً هر دو هفته یک‌بار است؛ برنامه زایمان و ساک زایمان را آماده کنید. درباره وضعیت قرارگیری جنین با پزشک صحبت کنید. برای شیردهی آماده شوید.',
                    'Visits are usually every two weeks; prepare your birth plan and hospital bag. Discuss the baby’s position with your doctor and get ready for breastfeeding.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم به بالای ناف رسیده و فشار بر معده و ریه‌ها زیاد است. مفاصل لگن برای زایمان شل‌تر می‌شوند. بدن با ذخیره انرژی برای زایمان آماده می‌شود.',
                    'The uterus is above the navel with high pressure on the stomach and lungs. Pelvic joints loosen for birth as the body stores energy for labor.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است هیجان و اضطراب درباره زایمان و والدگری بیشتر شود. خستگی می‌تواند صبر شما را کم کند. صحبت درباره احساسات و استراحت کمک‌کننده است.',
                    'Excitement and anxiety about birth and parenting may grow. Fatigue can shorten your patience; sharing feelings and resting help.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، پروتئین و فیبر را ادامه دهید و آب کافی بنوشید. غذاهای غنی از انرژی برای ذخیره‌سازی زایمان مفید است. از قند و نمک زیاد بپرهیزید.',
                    'Continue iron, calcium, protein and fiber and drink enough water. Energy-rich foods help build reserves for labor; limit excess sugar and salt.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و یوگای بارداری همچنان مفید است. تمرینات کف لگن و تنفسی را برای زایمان ادامه دهید. به بدن خود گوش دهید و افراط نکنید.',
                    'Walking, swimming and prenatal yoga remain helpful. Keep pelvic floor and breathing exercises for birth; listen to your body and don’t overdo it.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت روتین شامل فشار خون، وزن، رشد و وضعیت قرارگیری جنین است. سونوگرافی ممکن است برای بررسی وضعیت جنین و مایع آمنیوتیک انجام شود. علائم هشدار را گزارش دهید.',
                    'Routine visits include blood pressure, weight, growth and the baby’s position. An ultrasound may check the baby’s position and amniotic fluid; report warning signs.'
                ),
                'faq' => $this->faq(
                    [
                        ['اگر جنین رو به بالا باشد چه می‌شود؟', 'هنوز فرصت برای چرخش جنین وجود دارد؛ اگر تا هفته‌های پایانی بریچ بماند، پزشک گزینه‌های مناسب را بررسی می‌کند.'],
                        ['کاهش حرکات جنین را چطور تشخیص دهم؟', 'اگر الگوی معمول حرکات کم شد یا تغییر کرد، بلافاصله با پزشک تماس بگیرید.'],
                    ],
                    [
                        ['What if the baby is breech?', 'There is still time to turn; if it stays breech near the end, your doctor will review suitable options.'],
                        ['How do I notice reduced movements?', 'If the usual movement pattern decreases or changes, contact your doctor immediately.'],
                    ]
                ),
            ],
            33 => [
                'fetal_development' => $this->bi(
                    'در هفته سی‌وسوم جنین به اندازه یک آناناس، حدود ۴۳.۷ سانتی‌متر و نزدیک به ۱.۹ کیلوگرم است. استخوان‌های جمجمه هنوز نرم و انعطاف‌پذیر هستند تا از کانال زایمان عبور کنند. سیستم ایمنی جنین در حال تقویت است.',
                    'At week thirty-three the fetus is about the size of a pineapple, roughly 43.7 cm and nearly 1.9 kg. The skull bones are still soft and flexible to pass through the birth canal, and the baby’s immune system is strengthening.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است تنگی نفس، تورم، کمردرد و انقباضات تمرینی بیشتر شود. حرکات جنین قوی است. ممکن است احساس فشار در لگن داشته باشید.',
                    'Shortness of breath, swelling, back pain and practice contractions may increase. Movements are strong, and you may feel pressure in the pelvis.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و علائم پره‌اکلامپسی مانند ورم ناگهانی و سردرد شدید را جدی بگیرید. شمارش حرکات جنین را ادامه دهید. استراحت کافی داشته باشید.',
                    'Continue healthy eating and supplements and take pre-eclampsia signs like sudden swelling and severe headache seriously. Keep counting movements and rest enough.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌های منظم را دنبال کنید و برنامه زایمان را نهایی کنید. ساک زایمان باید آماده باشد. علائم شروع زایمان را مرور کنید و شماره‌های اضطراری را در دسترس داشته باشید.',
                    'Keep regular visits and finalize your birth plan. Your hospital bag should be ready; review the signs of labor and keep emergency numbers handy.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم فضای زیادی اشغال کرده و فشار بر اندام‌ها زیاد است. احتباس مایعات ممکن است تورم را افزایش دهد. بدن برای زایمان نزدیک آماده می‌شود.',
                    'The uterus fills much space with high pressure on the organs. Fluid retention may increase swelling as the body prepares for the approaching birth.'
                ),
                'emotional_status' => $this->bi(
                    'با نزدیک شدن زایمان ممکن است هیجان و اضطراب همزمان داشته باشید. غریزه آماده‌سازی خانه (لانه‌سازی) ممکن است فعال شود. آموزش و حمایت کمک‌کننده است.',
                    'As birth nears you may feel both excitement and anxiety. A nesting instinct to prepare the home may appear; education and support help.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، پروتئین و امگا-۳ را ادامه دهید و فیبر و آب کافی مصرف کنید. غذاهای مغذی و متعادل انرژی لازم برای زایمان را فراهم می‌کند. از قند و نمک زیاد بپرهیزید.',
                    'Continue iron, calcium, protein and omega-3 and get enough fiber and water. Nutritious balanced meals provide energy for labor; limit excess sugar and salt.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی، شنا و تمرینات ملایم به آمادگی زایمان کمک می‌کند. تمرینات کف لگن و تنفسی را ادامه دهید. از فعالیت‌های سنگین و پرخطر بپرهیزید.',
                    'Walking, swimming and gentle exercise aid birth readiness. Keep pelvic floor and breathing exercises and avoid heavy or risky activities.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت روتین شامل فشار خون، وزن، رشد و وضعیت جنین است. پزشک ممکن است سونوگرافی برای بررسی وضعیت قرارگیری جنین انجام دهد. علائم زایمان و پره‌اکلامپسی را گزارش دهید.',
                    'Routine visits include blood pressure, weight, growth and the baby’s position. Your doctor may do an ultrasound to check position; report labor and pre-eclampsia signs.'
                ),
                'faq' => $this->faq(
                    [
                        ['غریزه لانه‌سازی چیست؟', 'میل ناگهانی به تمیز کردن و آماده‌سازی خانه برای نوزاد است که در اواخر بارداری شایع است و طبیعی محسوب می‌شود.'],
                        ['فشار در لگن طبیعی است؟', 'با پایین آمدن جنین، فشار لگنی طبیعی است؛ اما فشار ریتمیک همراه با درد ممکن است نشانه زایمان باشد.'],
                    ],
                    [
                        ['What is the nesting instinct?', 'It is a sudden urge to clean and prepare the home for the baby, common in late pregnancy and considered normal.'],
                        ['Is pelvic pressure normal?', 'As the baby drops, pelvic pressure is normal, but rhythmic pressure with pain may signal labor.'],
                    ]
                ),
            ],
            34 => [
                'fetal_development' => $this->bi(
                    'در هفته سی‌وچهارم جنین به اندازه یک طالبی، حدود ۴۵ سانتی‌متر و نزدیک به ۲.۱ کیلوگرم است. ریه‌ها تقریباً بالغ شده‌اند و سیستم عصبی مرکزی در حال تکامل است. اگر نوزاد اکنون به دنیا بیاید، شانس خوبی برای سلامت دارد.',
                    'At week thirty-four the fetus is about the size of a cantaloupe, roughly 45 cm and nearly 2.1 kg. The lungs are nearly mature and the central nervous system is developing; a baby born now has a good chance of doing well.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است تنگی نفس، تورم، فشار لگنی و انقباضات تمرینی بیشتر شود. حرکات جنین قوی اما محدودتر است. ممکن است خستگی و مشکلات خواب داشته باشید.',
                    'Shortness of breath, swelling, pelvic pressure and practice contractions may increase. Movements are strong but more limited, and you may have fatigue and sleep issues.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و شمارش حرکات جنین را جدی بگیرید. علائم زایمان و پره‌اکلامپسی را بشناسید. استراحت کافی داشته باشید و به پهلو بخوابید.',
                    'Continue healthy eating and supplements and take movement counting seriously. Know labor and pre-eclampsia signs, rest enough, and sleep on your side.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌های منظم را دنبال کنید و درباره غربالگری GBS پیش رو با پزشک صحبت کنید. برنامه زایمان و ساک زایمان باید آماده باشد. مسیر بیمارستان را از قبل مشخص کنید.',
                    'Keep regular visits and discuss the upcoming GBS screening with your doctor. Your birth plan and hospital bag should be ready; plan your route to the hospital.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم فضای زیادی گرفته و فشار بر ریه‌ها و لگن زیاد است. مفاصل لگن برای زایمان شل‌تر می‌شوند. بدن انرژی را برای زایمان ذخیره می‌کند.',
                    'The uterus takes much space with high pressure on the lungs and pelvis. Pelvic joints loosen for birth as the body stores energy for labor.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است انتظار برای دیدن نوزاد با اضطراب زایمان همراه باشد. خستگی می‌تواند بر خلق‌وخو اثر بگذارد. حمایت عاطفی و آموزش زایمان آرامش می‌آورد.',
                    'Anticipation to meet the baby may mix with birth anxiety. Fatigue can affect mood; emotional support and birth education bring calm.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، پروتئین و فیبر را ادامه دهید و آب کافی بنوشید. غذاهای مغذی برای ذخیره انرژی زایمان مفید است. وعده‌های کوچک سوزش سردل را کاهش می‌دهد.',
                    'Continue iron, calcium, protein and fiber and drink enough water. Nutritious foods help store energy for labor, and small meals reduce heartburn.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی و شنا به گردش خون و آمادگی زایمان کمک می‌کند. تمرینات کف لگن و تنفسی را ادامه دهید. از فعالیت‌های سنگین بپرهیزید و استراحت کنید.',
                    'Walking and swimming aid circulation and birth readiness. Keep pelvic floor and breathing exercises, avoid heavy activity, and rest.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت روتین شامل فشار خون، وزن، رشد و وضعیت جنین است. غربالگری استرپتوکوک گروه B (GBS) به‌زودی در هفته‌های ۳۵ تا ۳۷ انجام می‌شود. علائم هشدار را گزارش دهید.',
                    'Routine visits include blood pressure, weight, growth and the baby’s position. Group B strep (GBS) screening is done soon in weeks 35 to 37; report warning signs.'
                ),
                'faq' => $this->faq(
                    [
                        ['اگر زایمان زودتر شروع شود چه می‌شود؟', 'در این هفته ریه‌ها تقریباً بالغ‌اند و نوزاد معمولاً وضعیت خوبی دارد، هرچند ممکن است به مراقبت کوتاه نیاز باشد.'],
                        ['غربالگری GBS چیست؟', 'یک نمونه‌گیری ساده برای بررسی باکتری استرپتوکوک گروه B است تا در صورت وجود، هنگام زایمان آنتی‌بیوتیک تجویز شود.'],
                    ],
                    [
                        ['What if labor starts earlier?', 'By this week the lungs are nearly mature and the baby usually does well, though brief care may be needed.'],
                        ['What is GBS screening?', 'It is a simple swab to check for group B strep bacteria so antibiotics can be given during labor if present.'],
                    ]
                ),
            ],
            35 => [
                'fetal_development' => $this->bi(
                    'در هفته سی‌وپنجم جنین به اندازه یک خربزه، حدود ۴۶.۲ سانتی‌متر و نزدیک به ۲.۴ کیلوگرم است. کلیه‌ها به‌طور کامل رشد کرده‌اند و کبد شروع به پردازش مواد زائد می‌کند. جنین به‌سرعت وزن اضافه می‌کند و فضای رحم تنگ‌تر می‌شود.',
                    'At week thirty-five the fetus is about the size of a honeydew, roughly 46.2 cm and nearly 2.4 kg. The kidneys are fully developed and the liver begins processing waste, while the baby gains weight fast and the womb grows snug.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است فشار لگنی، تکرر ادرار و تنگی نفس بیشتر شود. حرکات جنین بیشتر به‌صورت پیچ‌وتاب حس می‌شود. ممکن است انقباضات تمرینی مکرر داشته باشید.',
                    'Pelvic pressure, frequent urination and shortness of breath may increase. Movements feel more like squirms, and you may have frequent practice contractions.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و شمارش حرکات جنین را جدی بگیرید. علائم زایمان واقعی و پره‌اکلامپسی را بشناسید. استراحت کافی داشته باشید و ساک زایمان را آماده نگه دارید.',
                    'Continue healthy eating and supplements and take movement counting seriously. Know true labor and pre-eclampsia signs, rest enough, and keep the hospital bag ready.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌ها ممکن است مکرر‌تر شود؛ غربالگری GBS از این هفته انجام می‌شود. برنامه زایمان را نهایی کنید و مسیر بیمارستان را مشخص کنید. برای شیردهی و مراقبت از نوزاد آماده شوید.',
                    'Visits may become more frequent, and GBS screening is done from this week. Finalize your birth plan and hospital route and prepare for breastfeeding and newborn care.'
                ),
                'body_adaptation' => $this->bi(
                    'رحم تا نزدیک قفسه سینه بالا آمده و فشار بر ریه‌ها و معده زیاد است. لگن برای زایمان کم‌کم شل می‌شود. بدن در مرحله نهایی آماده‌سازی است.',
                    'The uterus has risen near the ribs with high pressure on the lungs and stomach. The pelvis loosens for birth as the body enters its final preparation.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است هیجان دیدن نوزاد با اضطراب زایمان و انتظار همراه باشد. غریزه لانه‌سازی ممکن است قوی شود. استراحت و حمایت عاطفی مهم است.',
                    'Excitement to meet the baby may mix with birth anxiety and waiting. The nesting instinct may grow strong; rest and emotional support matter.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، پروتئین و امگا-۳ را ادامه دهید و فیبر و آب کافی مصرف کنید. وعده‌های کوچک و مکرر به دلیل فضای کم معده بهتر است. از قند و نمک زیاد بپرهیزید.',
                    'Continue iron, calcium, protein and omega-3 and get enough fiber and water. Small frequent meals suit the reduced stomach space; limit excess sugar and salt.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی سبک و تمرینات کف لگن به آمادگی زایمان کمک می‌کند. تمرینات تنفسی را برای مدیریت درد زایمان تمرین کنید. از فعالیت‌های سنگین بپرهیزید.',
                    'Light walking and pelvic floor exercises aid birth readiness. Practice breathing exercises to manage labor pain and avoid heavy activity.'
                ),
                'tests_and_checkups' => $this->bi(
                    'غربالگری استرپتوکوک گروه B (GBS) با نمونه‌گیری از این هفته انجام می‌شود. ویزیت روتین شامل فشار خون، وزن، وضعیت جنین و گاهی معاینه دهانه رحم است. علائم زایمان را گزارش دهید.',
                    'Group B strep (GBS) screening by swab is done from this week. Routine visits include blood pressure, weight, the baby’s position and sometimes a cervical check.'
                ),
                'faq' => $this->faq(
                    [
                        ['تفاوت انقباض واقعی و تمرینی چیست؟', 'انقباض واقعی منظم، فزاینده و دردناک است و با استراحت متوقف نمی‌شود، اما انقباض تمرینی نامنظم و گذراست.'],
                        ['غربالگری GBS چگونه است؟', 'یک نمونه ساده از واژن و مقعد گرفته می‌شود و در صورت مثبت بودن، هنگام زایمان آنتی‌بیوتیک داده می‌شود.'],
                    ],
                    [
                        ['What is the difference between real and practice contractions?', 'Real contractions are regular, intensifying and painful and don’t stop with rest, while practice ones are irregular and fleeting.'],
                        ['How is GBS screening done?', 'A simple swab of the vagina and rectum is taken, and if positive, antibiotics are given during labor.'],
                    ]
                ),
            ],
            36 => [
                'fetal_development' => $this->bi(
                    'در هفته سی‌وششم جنین به اندازه یک کاهو رومی، حدود ۴۷.۴ سانتی‌متر و نزدیک به ۲.۶ کیلوگرم است. جنین معمولاً در وضعیت رو به پایین قرار می‌گیرد و کرک لانوگو تقریباً از بین رفته است. ریه‌ها در حال کامل شدن هستند.',
                    'At week thirty-six the fetus is about the size of a romaine lettuce, roughly 47.4 cm and nearly 2.6 kg. The baby usually settles head-down, the lanugo is nearly gone, and the lungs are almost fully developed.'
                ),
                'mother_body_changes' => $this->bi(
                    'با پایین آمدن جنین ممکن است تنفس راحت‌تر اما فشار لگنی بیشتر شود. تکرر ادرار و انقباضات تمرینی افزایش می‌یابد. ممکن است خستگی زیادی احساس کنید.',
                    'As the baby drops, breathing may ease but pelvic pressure increases. Frequent urination and practice contractions rise, and you may feel very tired.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و شمارش حرکات جنین را جدی بگیرید. علائم زایمان واقعی را بشناسید و در صورت پارگی کیسه آب فوراً اقدام کنید. استراحت کافی داشته باشید.',
                    'Continue healthy eating and supplements and take movement counting seriously. Know true labor signs and act at once if your waters break; rest enough.'
                ),
                'care_plan' => $this->bi(
                    'از این هفته ویزیت‌ها معمولاً هفتگی می‌شود. اگر غربالگری GBS را نداده‌اید انجام دهید. ساک زایمان و برنامه زایمان باید کاملاً آماده باشد.',
                    'From this week visits usually become weekly. Do the GBS screening if not done, and your hospital bag and birth plan should be fully ready.'
                ),
                'body_adaptation' => $this->bi(
                    'جنین به سمت لگن پایین می‌آید که فشار قفسه سینه را کم اما فشار مثانه را زیاد می‌کند. دهانه رحم ممکن است شروع به نرم شدن کند. بدن برای زایمان آماده می‌شود.',
                    'The baby moves down into the pelvis, easing chest pressure but increasing bladder pressure. The cervix may begin to soften as the body prepares for labor.'
                ),
                'emotional_status' => $this->bi(
                    'با نزدیک شدن به موعد زایمان ممکن است هیجان و بی‌صبری همراه با اضطراب داشته باشید. آماده‌سازی نهایی می‌تواند آرامش بدهد. حمایت اطرافیان مهم است.',
                    'As the due date nears you may feel excitement and impatience along with anxiety. Final preparations can bring calm, and support from others matters.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، پروتئین و فیبر را ادامه دهید و آب کافی بنوشید. وعده‌های کوچک و مکرر راحت‌تر هضم می‌شود. غذاهای مغذی برای انرژی زایمان مفید است.',
                    'Continue iron, calcium, protein and fiber and drink enough water. Small frequent meals digest more easily, and nutritious foods help energy for labor.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی سبک و تمرینات کف لگن به آماده شدن برای زایمان کمک می‌کند. تمرینات تنفسی را تمرین کنید. به بدن خود گوش دهید و از خستگی زیاد بپرهیزید.',
                    'Light walking and pelvic floor exercises help prepare for labor. Practice breathing exercises, listen to your body, and avoid overexertion.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت‌ها از این هفته هفتگی می‌شود و شامل فشار خون، وزن، وضعیت جنین و معاینه دهانه رحم است. نتیجه غربالگری GBS بررسی می‌شود. علائم زایمان را گزارش دهید.',
                    'Visits become weekly from this week and include blood pressure, weight, the baby’s position and a cervical check. The GBS result is reviewed; report labor signs.'
                ),
                'faq' => $this->faq(
                    [
                        ['پایین آمدن جنین یعنی زایمان نزدیک است؟', 'پایین آمدن جنین نشانه آماده شدن بدن است اما لزوماً به معنای زایمان فوری نیست و می‌تواند هفته‌ها زودتر رخ دهد.'],
                        ['اگر کیسه آب پاره شود چه کنم؟', 'در صورت نشت یا پارگی کیسه آب، فوراً با بیمارستان یا پزشک تماس بگیرید حتی اگر انقباض نداشته باشید.'],
                    ],
                    [
                        ['Does the baby dropping mean labor is near?', 'Dropping shows the body is getting ready but doesn’t necessarily mean immediate labor and can happen weeks early.'],
                        ['What if my waters break?', 'If your waters leak or break, contact the hospital or doctor immediately even without contractions.'],
                    ]
                ),
            ],
            37 => [
                'fetal_development' => $this->bi(
                    'در هفته سی‌وهفتم جنین به اندازه یک برگ چغندر، حدود ۴۸.۶ سانتی‌متر و نزدیک به ۲.۹ کیلوگرم است و اکنون «ترم اولیه» محسوب می‌شود. جنین برای تنفس، مکیدن و بلعیدن تمرین می‌کند و ریه‌ها تقریباً کامل هستند. چربی بیشتری زیر پوست ذخیره می‌کند.',
                    'At week thirty-seven the fetus is about the size of a bunch of chard, roughly 48.6 cm and nearly 2.9 kg, and is now considered early term. The baby practices breathing, sucking and swallowing with nearly complete lungs and stores more fat.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است فشار لگنی، انقباضات تمرینی و ترشحات بیشتر شود. ممکن است افتادگی مخاطی (نشانه خونی) را ببینید. خستگی و بی‌قراری شایع است.',
                    'Pelvic pressure, practice contractions and discharge may increase. You may notice the mucus plug (bloody show), and fatigue and restlessness are common.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و علائم زایمان واقعی را جدی بگیرید. در صورت پارگی کیسه آب یا انقباضات منظم به بیمارستان مراجعه کنید. استراحت کافی داشته باشید.',
                    'Continue healthy eating and supplements and take true labor signs seriously. Go to the hospital if your waters break or contractions become regular; rest enough.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌ها هفتگی است و شامل معاینه دهانه رحم می‌شود. برنامه و ساک زایمان باید آماده باشد. علائم شروع زایمان و زمان مراجعه به بیمارستان را مرور کنید.',
                    'Visits are weekly and include a cervical check. Your birth plan and hospital bag should be ready; review labor signs and when to go to the hospital.'
                ),
                'body_adaptation' => $this->bi(
                    'دهانه رحم ممکن است شروع به نازک و باز شدن کند و جنین در لگن پایین‌تر می‌آید. انقباضات تمرینی مکرر‌تر می‌شود. بدن در آستانه زایمان است.',
                    'The cervix may begin to thin and open and the baby settles lower in the pelvis. Practice contractions grow more frequent as the body nears labor.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است ترکیبی از هیجان، بی‌صبری و اضطراب زایمان داشته باشید. انتظار می‌تواند سخت باشد. تمرکز بر آمادگی و آرامش کمک‌کننده است.',
                    'You may feel a mix of excitement, impatience and birth anxiety. The waiting can be hard; focusing on preparation and calm helps.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، پروتئین و فیبر را ادامه دهید و آب کافی بنوشید. غذاهای مغذی و انرژی‌زا برای زایمان مفید است. وعده‌های کوچک راحت‌تر هضم می‌شود.',
                    'Continue iron, calcium, protein and fiber and drink enough water. Nutritious, energizing foods help for labor, and small meals digest more easily.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی سبک و تمرینات کف لگن به آماده شدن برای زایمان کمک می‌کند. تمرینات تنفسی را تمرین کنید. به بدن خود گوش دهید و افراط نکنید.',
                    'Light walking and pelvic floor exercises help prepare for labor. Practice breathing exercises, listen to your body, and don’t overdo it.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت هفتگی شامل فشار خون، وزن، وضعیت جنین و معاینه دهانه رحم برای بررسی نازک و باز شدن است. اگر غربالگری GBS را نداده‌اید انجام دهید. علائم زایمان را گزارش دهید.',
                    'Weekly visits include blood pressure, weight, the baby’s position and a cervical check for thinning and dilation. Do GBS screening if not done; report labor signs.'
                ),
                'faq' => $this->faq(
                    [
                        ['«ترم اولیه» یعنی چه؟', 'یعنی نوزاد از هفته ۳۷ به بلوغ کافی برای تولد رسیده است، هرچند ماندن تا هفته ۳۹ برای رشد کامل بهتر است.'],
                        ['نشانه خونی (افتادگی مخاطی) چیست؟', 'خروج توده مخاطی همراه با کمی خون نشانه آماده شدن دهانه رحم است و می‌تواند چند روز پیش از زایمان رخ دهد.'],
                    ],
                    [
                        ['What does early term mean?', 'It means from week 37 the baby is mature enough for birth, though staying to week 39 is better for full development.'],
                        ['What is the bloody show?', 'It is the mucus plug with a little blood, a sign the cervix is preparing, and can occur days before labor.'],
                    ]
                ),
            ],
            38 => [
                'fetal_development' => $this->bi(
                    'در هفته سی‌وهشتم جنین به اندازه یک تره‌فرنگی، حدود ۴۹.۸ سانتی‌متر و نزدیک به ۳.۱ کیلوگرم است. اندام‌ها کامل شده و جنین برای زندگی خارج از رحم آماده می‌شود. چنگ زدن قوی‌تر شده و رنگ چشم‌ها هنوز نهایی نیست.',
                    'At week thirty-eight the fetus is about the size of a leek, roughly 49.8 cm and nearly 3.1 kg. The organs are complete and the baby is ready for life outside the womb, with a firmer grasp and eye color not yet final.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است فشار لگنی، تورم پا و انقباضات تمرینی بیشتر شود. ممکن است افزایش ترشحات یا نشانه خونی را ببینید. خواب و راحتی دشوارتر است.',
                    'Pelvic pressure, leg swelling and practice contractions may increase. You may notice more discharge or the bloody show, and sleep and comfort are harder.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و علائم زایمان واقعی را جدی بگیرید. در صورت پارگی کیسه آب یا انقباضات منظم به بیمارستان بروید. شمارش حرکات جنین را ادامه دهید.',
                    'Continue healthy eating and supplements and take true labor signs seriously. Go to the hospital if your waters break or contractions become regular; keep counting movements.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت هفتگی را دنبال کنید و درباره برنامه زایمان و علائم مراجعه اطمینان حاصل کنید. ساک زایمان باید آماده باشد. شماره‌های اضطراری و مسیر بیمارستان را در دسترس داشته باشید.',
                    'Keep weekly visits and confirm your birth plan and when to come in. Your hospital bag should be ready, and keep emergency numbers and the route handy.'
                ),
                'body_adaptation' => $this->bi(
                    'دهانه رحم ممکن است بیشتر نازک و باز شود و جنین کاملاً در لگن قرار گیرد. انقباضات تمرینی قوی‌تر می‌شود. بدن آماده شروع زایمان است.',
                    'The cervix may thin and open further and the baby settles fully into the pelvis. Practice contractions strengthen as the body readies for labor.'
                ),
                'emotional_status' => $this->bi(
                    'انتظار برای زایمان می‌تواند هیجان، بی‌صبری و اضطراب ایجاد کند. ممکن است هر نشانه‌ای را با دقت دنبال کنید. آرامش و حمایت اطرافیان کمک‌کننده است.',
                    'Waiting for labor can bring excitement, impatience and anxiety. You may watch every sign closely; calm and support from others help.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، پروتئین و فیبر را ادامه دهید و آب کافی بنوشید. غذاهای مغذی و انرژی‌زا برای زایمان مفید است. وعده‌های کوچک راحت‌تر هضم می‌شود.',
                    'Continue iron, calcium, protein and fiber and drink enough water. Nutritious, energizing foods help for labor, and small meals digest more easily.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی سبک و تمرینات کف لگن ممکن است به آماده شدن برای زایمان کمک کند. تمرینات تنفسی را تمرین کنید. به بدن خود گوش دهید و استراحت کنید.',
                    'Light walking and pelvic floor exercises may help prepare for labor. Practice breathing exercises, listen to your body, and rest.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت هفتگی شامل فشار خون، وزن، وضعیت جنین و معاینه دهانه رحم است. پزشک ممکن است ضربان قلب جنین و مایع آمنیوتیک را بررسی کند. علائم زایمان را گزارش دهید.',
                    'Weekly visits include blood pressure, weight, the baby’s position and a cervical check. Your doctor may assess the fetal heartbeat and amniotic fluid; report labor signs.'
                ),
                'faq' => $this->faq(
                    [
                        ['چه زمانی باید به بیمارستان بروم؟', 'با انقباضات منظم و دردناک با فواصل کوتاه، پارگی کیسه آب یا خونریزی به بیمارستان مراجعه کنید.'],
                        ['اگر تا موعد زایمان نشد چه می‌شود؟', 'تولد بین هفته ۳۷ تا ۴۲ طبیعی است؛ اگر از موعد گذشت، پزشک وضعیت را پایش می‌کند و در صورت لزوم اقدام می‌کند.'],
                    ],
                    [
                        ['When should I go to the hospital?', 'Go with regular painful contractions at short intervals, if your waters break, or with bleeding.'],
                        ['What if labor doesn’t start by the due date?', 'Birth between weeks 37 and 42 is normal; if you go past due, your doctor monitors and acts if needed.'],
                    ]
                ),
            ],
            39 => [
                'fetal_development' => $this->bi(
                    'در هفته سی‌ونهم که «ترم کامل» محسوب می‌شود، جنین به اندازه یک هندوانه کوچک، حدود ۵۰.۷ سانتی‌متر و نزدیک به ۳.۳ کیلوگرم است. اندام‌ها کاملاً بالغ و آماده تولد هستند. جنین چربی کافی برای تنظیم دمای بدن دارد.',
                    'At week thirty-nine, considered full term, the fetus is about the size of a mini watermelon, roughly 50.7 cm and nearly 3.3 kg. The organs are fully mature and ready for birth, and the baby has enough fat to regulate body temperature.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است فشار لگنی، انقباضات و ترشحات بیشتر شود. خواب و حرکت دشوارتر است. ممکن است نشانه‌های اولیه زایمان را تجربه کنید.',
                    'Pelvic pressure, contractions and discharge may increase. Sleep and movement are harder, and you may notice early signs of labor.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و علائم زایمان واقعی را جدی بگیرید. در صورت پارگی کیسه آب یا انقباضات منظم فوراً به بیمارستان بروید. شمارش حرکات جنین را ادامه دهید.',
                    'Continue healthy eating and supplements and take true labor signs seriously. Go to the hospital at once if your waters break or contractions are regular; keep counting movements.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت هفتگی را دنبال کنید و آماده رفتن به بیمارستان باشید. ساک زایمان و برنامه زایمان باید آماده باشد. با پزشک درباره نشانه‌های شروع زایمان صحبت کنید.',
                    'Keep weekly visits and be ready to go to the hospital. Your bag and birth plan should be ready; discuss labor onset signs with your doctor.'
                ),
                'body_adaptation' => $this->bi(
                    'دهانه رحم برای زایمان نرم، نازک و کم‌کم باز می‌شود. جنین کاملاً در لگن قرار گرفته است. بدن در آستانه زایمان قرار دارد.',
                    'The cervix softens, thins and gradually opens for labor, and the baby is fully settled in the pelvis. The body is on the threshold of labor.'
                ),
                'emotional_status' => $this->bi(
                    'انتظار روزهای پایانی می‌تواند بی‌صبری و اضطراب همراه با هیجان ایجاد کند. تمرکز بر آرامش و آمادگی کمک‌کننده است. حمایت اطرافیان بسیار مهم است.',
                    'Waiting in these final days can bring impatience and anxiety along with excitement. Focusing on calm and readiness helps, and support from others is very important.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، پروتئین و فیبر را ادامه دهید و آب کافی بنوشید. غذاهای مغذی برای انرژی زایمان مفید است. وعده‌های کوچک و مکرر راحت‌تر است.',
                    'Continue iron, calcium, protein and fiber and drink enough water. Nutritious foods help energy for labor, and small frequent meals are easier.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی سبک ممکن است به پیشرفت زایمان کمک کند. تمرینات تنفسی و کف لگن را تمرین کنید. به بدن خود گوش دهید و بین فعالیت و استراحت تعادل برقرار کنید.',
                    'Light walking may help labor progress. Practice breathing and pelvic floor exercises, listen to your body, and balance activity with rest.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت هفتگی شامل فشار خون، وزن، وضعیت جنین و معاینه دهانه رحم است. پزشک ضربان قلب جنین و در صورت نیاز مایع آمنیوتیک را بررسی می‌کند. علائم زایمان را گزارش دهید.',
                    'Weekly visits include blood pressure, weight, the baby’s position and a cervical check. The doctor assesses the fetal heartbeat and, if needed, amniotic fluid.'
                ),
                'faq' => $this->faq(
                    [
                        ['«ترم کامل» چه اهمیتی دارد؟', 'از هفته ۳۹ نوزاد به بلوغ کامل رسیده و بهترین زمان برای تولد طبیعی محسوب می‌شود.'],
                        ['چطور شروع زایمان را تشخیص دهم؟', 'انقباضات منظم و فزاینده، پارگی کیسه آب یا نشانه خونی از علائم شروع زایمان هستند و نیاز به مراجعه دارند.'],
                    ],
                    [
                        ['Why does full term matter?', 'From week 39 the baby has reached full maturity and it is the best time for a natural birth.'],
                        ['How do I know labor is starting?', 'Regular intensifying contractions, waters breaking or the bloody show are signs of labor needing attention.'],
                    ]
                ),
            ],
            40 => [
                'fetal_development' => $this->bi(
                    'در هفته چهلم که موعد زایمان است، جنین به اندازه یک کدو تنبل کوچک، حدود ۵۱.۲ سانتی‌متر و نزدیک به ۳.۴ کیلوگرم است. نوزاد کاملاً رشد کرده و آماده تولد است. استخوان‌های جمجمه هنوز نرم‌اند تا از کانال زایمان عبور کنند.',
                    'At week forty, the due date, the fetus is about the size of a small pumpkin, roughly 51.2 cm and nearly 3.4 kg. The baby is fully grown and ready to be born, with skull bones still soft to pass through the birth canal.'
                ),
                'mother_body_changes' => $this->bi(
                    'ممکن است فشار لگنی شدید، انقباضات و بی‌قراری داشته باشید. نشانه‌های زایمان مانند نشانه خونی یا پارگی کیسه آب ممکن است ظاهر شود. انتظار می‌تواند خسته‌کننده باشد.',
                    'You may have strong pelvic pressure, contractions and restlessness. Labor signs like the bloody show or waters breaking may appear, and waiting can be tiring.'
                ),
                'dos_and_donts' => $this->bi(
                    'تغذیه سالم و مکمل‌ها را ادامه دهید و علائم زایمان را جدی بگیرید. در صورت پارگی کیسه آب، خونریزی یا انقباضات منظم فوراً به بیمارستان بروید. شمارش حرکات جنین را ادامه دهید.',
                    'Continue healthy eating and supplements and take labor signs seriously. Go to the hospital at once if your waters break, you bleed, or contractions are regular; keep counting movements.'
                ),
                'care_plan' => $this->bi(
                    'ویزیت‌ها ادامه دارد و پزشک وضعیت جنین و دهانه رحم را پایش می‌کند. اگر تا چند روز پس از موعد زایمان نشد، پزشک ممکن است القای زایمان را بررسی کند. آماده رفتن به بیمارستان باشید.',
                    'Visits continue and the doctor monitors the baby and cervix. If labor doesn’t start a few days past the due date, the doctor may consider induction; be ready to go in.'
                ),
                'body_adaptation' => $this->bi(
                    'دهانه رحم برای زایمان باز و آماده می‌شود و انقباضات ممکن است هر لحظه شروع شوند. بدن تمام آمادگی خود را برای زایمان به کار می‌گیرد. این مرحله نهایی سفر بارداری است.',
                    'The cervix opens and readies for birth and contractions may begin any moment. The body brings all its readiness to labor in this final stage of the pregnancy journey.'
                ),
                'emotional_status' => $this->bi(
                    'ممکن است ترکیبی از هیجان شدید، بی‌صبری و اضطراب زایمان را تجربه کنید. اگر از موعد گذشت، ممکن است کمی ناامید شوید که طبیعی است. آرامش و حمایت را حفظ کنید.',
                    'You may feel a mix of intense excitement, impatience and birth anxiety. If you go past due you may feel a little discouraged, which is normal; keep calm and supported.'
                ),
                'key_nutrition' => $this->bi(
                    'آهن، کلسیم، پروتئین و فیبر را ادامه دهید و آب کافی بنوشید. غذاهای سبک و انرژی‌زا برای زایمان مفید است. وعده‌های کوچک راحت‌تر هضم می‌شود.',
                    'Continue iron, calcium, protein and fiber and drink enough water. Light, energizing foods help for labor, and small meals digest more easily.'
                ),
                'physical_activity' => $this->bi(
                    'پیاده‌روی سبک ممکن است به شروع یا پیشرفت زایمان کمک کند. تمرینات تنفسی را برای مدیریت درد تمرین کنید. به بدن خود گوش دهید و استراحت کافی داشته باشید.',
                    'Light walking may help start or progress labor. Practice breathing exercises to manage pain, listen to your body, and rest enough.'
                ),
                'tests_and_checkups' => $this->bi(
                    'ویزیت شامل فشار خون، وزن، معاینه دهانه رحم و پایش ضربان قلب جنین است. اگر از موعد گذشت، پزشک ممکن است سونوگرافی و تست سلامت جنین (NST) را برای اطمینان توصیه کند. علائم زایمان را گزارش دهید.',
                    'Visits include blood pressure, weight, a cervical check and fetal heartbeat monitoring. Past due, the doctor may advise an ultrasound and a non-stress test (NST) for reassurance.'
                ),
                'faq' => $this->faq(
                    [
                        ['اگر از موعد زایمان بگذرم چه می‌شود؟', 'تا هفته ۴۲ ممکن است طبیعی باشد؛ پزشک با پایش سلامت جنین تصمیم می‌گیرد و در صورت لزوم زایمان را القا می‌کند.'],
                        ['القای زایمان چیست؟', 'روشی پزشکی برای شروع زایمان است که در صورت گذشتن از موعد یا وجود دلایل پزشکی توسط پزشک انجام می‌شود.'],
                    ],
                    [
                        ['What if I go past my due date?', 'Up to week 42 can be normal; the doctor monitors the baby’s health and induces labor if needed.'],
                        ['What is labor induction?', 'It is a medical way to start labor, done by the doctor when you are past due or for medical reasons.'],
                    ]
                ),
            ],
        ];
    }
}


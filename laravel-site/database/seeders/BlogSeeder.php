<?php

declare(strict_types=1);

namespace Database\Seeders;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Blog\Enums\PostStatus;
use App\Domain\Blog\Models\Author;
use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\PostSlug;
use App\Domain\Blog\Models\Tag;
use Illuminate\Database\Seeder;

/**
 * DEMO magazine content from the design (blog.html, article.html): the seven category chips, the article
 * «درد پریود…» and the blog-list sample posts. Not called from DatabaseSeeder — demo data never reaches production
 * by accident:
 *
 *     php artisan db:seed --class=BlogSeeder
 *
 * Idempotent: rows are matched by slug and existing posts are left untouched (admin edits survive a re-run).
 * Reviewer/sources keep the design's `[...]` placeholders (docs/AUDIT.md §5.2) — no invented medical reviewer.
 * Copy follows the content red lines: no diagnosis claims, no «حتماً/قطعاً/دقیق‌ترین/تضمینی», no sales pressure.
 */
final class BlogSeeder extends Seeder
{
    /** Demo slug of the design article (docs/AUDIT.md §7); `dard-period` is kept as an old slug (301). */
    public const ARTICLE_SLUG = 'period-pain';

    public const ARTICLE_OLD_SLUG = 'dard-period';

    public function run(): void
    {
        $categories = $this->categories();
        $tags = $this->tags();
        [$author, $reviewer] = $this->authors();

        foreach ($this->posts() as $i => $data) {
            if (Post::query()->where('slug', $data['slug'])->exists()) {
                continue;
            }

            $post = new Post([
                'title' => $data['title'],
                'slug' => $data['slug'],
                'excerpt' => $data['excerpt'],
                'body' => $data['body'],
                'sources' => '<p>[فهرست منابع علمی]</p>',
                'category_id' => $categories[$data['category']]->id,
                'life_stage' => LifeStage::tryFrom($data['category']),
                'author_id' => $author->id,
                'reviewer_id' => $reviewer->id,
                'reviewed_at' => now()->subDays(10 + $i),
                'status' => PostStatus::Published,
                'published_at' => now()->subDays(2 + $i * 3)->setTime(9, 30),
                'is_featured' => $data['featured'],
            ]);
            $post->save();
            $post->tags()->sync(array_map(static fn (string $slug): int => $tags[$slug]->id, $data['tags']));
        }

        $article = Post::query()->where('slug', self::ARTICLE_SLUG)->first();
        if ($article !== null && ! PostSlug::query()->where('slug', self::ARTICLE_OLD_SLUG)->exists() && ! Post::query()->where('slug', self::ARTICLE_OLD_SLUG)->exists()) {
            PostSlug::query()->create(['post_id' => $article->id, 'slug' => self::ARTICLE_OLD_SLUG]);
        }
    }

    /**
     * @return array<string, Category>
     */
    private function categories(): array
    {
        $rows = [
            ['slug' => 'cycle', 'name' => 'چرخه و پریود', 'label' => 'چرخه', 'stage' => LifeStage::Cycle],
            ['slug' => 'ttc', 'name' => 'اقدام به بارداری', 'label' => 'باروری', 'stage' => LifeStage::Ttc],
            ['slug' => 'pregnancy', 'name' => 'بارداری', 'label' => 'بارداری', 'stage' => LifeStage::Pregnancy],
            ['slug' => 'postpartum', 'name' => 'پس از زایمان و کودک', 'label' => 'کودک', 'stage' => LifeStage::Postpartum],
            ['slug' => 'menopause', 'name' => 'یائسگی', 'label' => 'یائسگی', 'stage' => LifeStage::Menopause],
            ['slug' => 'teen', 'name' => 'نوجوان و والدین', 'label' => 'نوجوان', 'stage' => LifeStage::Teen],
            ['slug' => 'family', 'name' => 'خانواده و همدم', 'label' => 'خانواده', 'stage' => null],
        ];

        $categories = [];
        foreach ($rows as $order => $row) {
            $categories[$row['slug']] = Category::query()->firstOrCreate(['slug' => $row['slug']], [
                'name' => $row['name'],
                'label' => $row['label'],
                'life_stage' => $row['stage'],
                'sort_order' => $order + 1,
            ]);
        }

        return $categories;
    }

    /**
     * @return array<string, Tag>
     */
    private function tags(): array
    {
        $rows = [
            'period-pain' => 'درد پریود',
            'menstrual-cycle' => 'چرخه قاعدگی',
            'hospital-bag' => 'ساک بیمارستان',
            'breastfeeding' => 'شیردهی',
            'newborn' => 'نوزاد',
            'hot-flashes' => 'گرگرفتگی',
            'first-period' => 'اولین پریود',
            'ovulation' => 'تخمک‌گذاری',
            'self-care' => 'مراقبت از خود',
        ];

        $tags = [];
        foreach ($rows as $slug => $name) {
            $tags[$slug] = Tag::query()->firstOrCreate(['slug' => $slug], ['name' => $name]);
        }

        return $tags;
    }

    /**
     * @return array{0: Author, 1: Author}
     */
    private function authors(): array
    {
        $author = Author::query()->firstOrCreate(['slug' => 'ritme-editorial'], [
            'name' => 'تیم محتوای ریتمی',
            'job_title' => 'تحریریه مجله ریتمی',
            'bio' => 'تحریریه ریتمی مطالب را بر پایه منابع علمی می‌نویسد و پیش از انتشار برای بازبینی به متخصص می‌سپارد.',
            'same_as' => [],
            'is_medical_reviewer' => false,
        ]);

        $reviewer = Author::query()->firstOrCreate(['slug' => 'medical-reviewer'], [
            'name' => '[نام متخصص]',
            'job_title' => '[تخصص]',
            'credentials' => '[تخصص]',
            'bio' => '[معرفی کوتاه متخصص بازبین]',
            'same_as' => [],
            'is_medical_reviewer' => true,
        ]);

        return [$author, $reviewer];
    }

    /**
     * @return list<array{slug: string, title: string, excerpt: string, category: string, tags: list<string>, featured: bool, body: string}>
     */
    private function posts(): array
    {
        return [
            [
                'slug' => self::ARTICLE_SLUG,
                'title' => 'درد پریود؛ کی عادی است و کی ارزش پیگیری دارد؟',
                'excerpt' => 'درد پریود شایع است، ولی هر دردی را نباید تحمل کرد. نشانه‌هایی که بهتر است با پزشک درمیان بگذاری.',
                'category' => 'cycle',
                'tags' => ['period-pain', 'menstrual-cycle'],
                'featured' => true,
                'body' => <<<'HTML'
<p>درد زیر شکم در روزهای اول پریود برای بسیاری از زن‌ها آشناست. معمولاً از کمی قبل یا همزمان با شروع خونریزی آغاز می‌شود و در یکی دو روز آرام‌تر می‌شود.</p>
<h2>درد پریود چرا ایجاد می‌شود</h2>
<p>در روزهای پریود، دیواره رحم منقبض می‌شود تا لایه داخلی آن دفع شود. این انقباض‌ها ممکن است درد یا گرفتگی ایجاد کنند که شدتش از فردی به فرد دیگر و حتی از ماهی به ماه دیگر فرق می‌کند.</p>
<h2>چه چیزهایی ممکن است کمک کند</h2>
<ul>
<li>گرما روی شکم یا کمر</li>
<li>کمی تحرک ملایم، مثل پیاده‌روی</li>
<li>خواب کافی</li>
<li>مسکن ساده، با رعایت دستور مصرف یا مشورت با داروساز</li>
</ul>
<h2>کی ارزش پیگیری دارد</h2>
<ul>
<li>درد آن‌قدر شدید است که کارهای روزمره‌ات را متوقف می‌کند</li>
<li>با مسکن‌های ساده بهتر نمی‌شود</li>
<li>درد تازه شروع شده یا نسبت به قبل خیلی بیشتر شده</li>
<li>همراه خونریزی خیلی زیاد، تب یا درد هنگام رابطه است</li>
</ul>
<p>در این موارد بهتر است با پزشک یا ماما صحبت کنی. ثبت درد در ریتمی کمک می‌کند تصویر روشن‌تری به پزشک بدهی.</p>
<h2>در ریتمی چطور ثبت کنیم</h2>
<p>از دکمه + «درد» را انتخاب کن، شدت و محل را بزن؛ یا فقط بگو «از دیشب زیر دلم درد می‌کند، حدود شش از ده». ریتمی الگوی درد را در چرخه‌های مختلف نشانت می‌دهد.</p>
HTML,
            ],
            [
                'slug' => 'hospital-bag',
                'title' => 'ساک بیمارستان را از کی و با چه چیزهایی ببندیم؟',
                'excerpt' => 'یک فهرست ساده برای خودت، نوزاد و همراهت؛ و اینکه از چه هفته‌ای آماده‌اش کنی که خیالت راحت‌تر باشد.',
                'category' => 'pregnancy',
                'tags' => ['hospital-bag'],
                'featured' => false,
                'body' => <<<'HTML'
<p>آماده بودن ساک بیمارستان از چند هفته قبل، روزهای آخر بارداری را آرام‌تر می‌کند. بسیاری از ماماها پیشنهاد می‌کنند ساک از حدود هفته ۳۴ تا ۳۶ آماده باشد.</p>
<h2>برای خودت</h2>
<ul>
<li>کارت شناسایی، دفترچه بیمه و پرونده بارداری</li>
<li>لباس راحت با دکمه جلو برای شیردهی</li>
<li>نوار بهداشتی مخصوص پس از زایمان و لباس زیر راحت</li>
<li>وسایل شخصی، شارژر گوشی و یک بطری آب</li>
</ul>
<h2>برای نوزاد</h2>
<ul>
<li>چند دست لباس نوزادی، کلاه و جوراب</li>
<li>پتو یا قنداق سبک</li>
<li>پوشک سایز نوزاد</li>
</ul>
<h2>برای همراه</h2>
<p>لباس اضافه، کمی خوراکی و فهرست شماره‌های مهم. اگر بیمارستان خودش فهرستی دارد، آن را هم کنار این فهرست بگذار.</p>
HTML,
            ],
            [
                'slug' => 'breastfeeding-first-weeks',
                'title' => 'شیردهی در هفته‌های اول: سؤال‌هایی که همه دارند',
                'excerpt' => 'هر چند وقت یک‌بار؟ از کجا بفهمم شیر کافی است؟ پاسخ‌های کوتاه به پرتکرارترین سؤال‌های هفته‌های اول.',
                'category' => 'postpartum',
                'tags' => ['breastfeeding', 'newborn'],
                'featured' => false,
                'body' => <<<'HTML'
<p>هفته‌های اول شیردهی برای مادر و نوزاد دوره یادگیری است. طبیعی است که سؤال زیاد داشته باشی و هر روز کمی با روز قبل فرق کند.</p>
<h2>هر چند وقت یک‌بار شیر بدهم؟</h2>
<p>نوزادها در هفته‌های اول معمولاً هر دو تا سه ساعت یک‌بار شیر می‌خورند. دنبال کردن نشانه‌های گرسنگی نوزاد، مثل مکیدن دست یا چرخاندن سر، اغلب از ساعت مفیدتر است.</p>
<h2>از کجا بفهمم شیر کافی است؟</h2>
<p>تعداد پوشک‌های خیس و کثیف و روند وزن‌گیری در معاینه‌های دوره‌ای نشانه‌های خوبی هستند. اگر نگرانی، با ماما، مشاور شیردهی یا پزشک کودک صحبت کن.</p>
<h2>درد هنگام شیردهی</h2>
<p>کمی حساسیت در روزهای اول شایع است، اما درد شدید یا ترک خوردن نوک سینه ارزش مشورت دارد؛ گاهی تغییر وضعیت در آغوش گرفتن نوزاد کمک می‌کند.</p>
HTML,
            ],
            [
                'slug' => 'hot-flashes',
                'title' => 'گرگرفتگی: چه چیزهایی ممکن است کمک کند',
                'excerpt' => 'گرگرفتگی در سال‌های اطراف یائسگی شایع است. چند تغییر ساده روزمره که برای بعضی‌ها مفید بوده است.',
                'category' => 'menopause',
                'tags' => ['hot-flashes', 'self-care'],
                'featured' => false,
                'body' => <<<'HTML'
<p>گرگرفتگی احساس ناگهانی گرما در صورت، گردن و سینه است که گاهی با تعریق و تپش قلب همراه می‌شود. در سال‌های اطراف یائسگی برای بسیاری از زن‌ها پیش می‌آید.</p>
<h2>تغییرهای ساده روزمره</h2>
<ul>
<li>لباس چندلایه و نخی که راحت درآورده شود</li>
<li>خنک نگه داشتن اتاق خواب</li>
<li>توجه به محرک‌هایی مثل نوشیدنی داغ، غذای تند یا استرس</li>
<li>تحرک منظم و تمرین‌های تنفس آرام</li>
</ul>
<h2>کی با پزشک صحبت کنیم</h2>
<p>اگر گرگرفتگی خواب یا کارهای روزانه‌ات را به‌هم می‌زند، با پزشک درباره راه‌های مختلف کنترل آن صحبت کن. ثبت زمان و شدت گرگرفتگی در ریتمی گفت‌وگو با پزشک را ساده‌تر می‌کند.</p>
HTML,
            ],
            [
                'slug' => 'first-period-guide',
                'title' => 'اولین پریود: راهنمای دختر و مادر',
                'excerpt' => 'چطور درباره اولین پریود حرف بزنیم و چه چیزهایی را از قبل آماده کنیم تا این تجربه کمتر نگران‌کننده باشد.',
                'category' => 'teen',
                'tags' => ['first-period', 'menstrual-cycle'],
                'featured' => false,
                'body' => <<<'HTML'
<p>اولین پریود معمولاً بین ۱۰ تا ۱۵ سالگی اتفاق می‌افتد. حرف زدن از قبل کمک می‌کند این تجربه برای دختر نوجوان آشناتر و کمتر نگران‌کننده باشد.</p>
<h2>چطور شروع کنیم</h2>
<p>یک گفت‌وگوی آرام و بدون قضاوت، با کلمه‌های ساده، بهتر از یک توضیح طولانی یک‌باره است. اجازه بده سؤال بپرسد و اگر جواب را نمی‌دانی، با هم پیدایش کنید.</p>
<h2>چه چیزهایی آماده باشد</h2>
<ul>
<li>چند نوار بهداشتی در کیف مدرسه</li>
<li>یک لباس زیر اضافه</li>
<li>آشنایی با اینکه در مدرسه از چه کسی کمک بخواهد</li>
</ul>
<h2>چرخه‌های اول</h2>
<p>در سال‌های اول، نامنظم بودن چرخه شایع است. اگر درد خیلی شدید بود یا سؤالی پیش آمد، با پزشک یا ماما مشورت کنید.</p>
HTML,
            ],
            [
                'slug' => 'lh-test',
                'title' => 'تست LH را کی و چطور بزنیم؟',
                'excerpt' => 'تست LH کمک می‌کند حدود زمان تخمک‌گذاری را بهتر بشناسی. از چه روزی شروع کنیم و نتیجه را چطور بخوانیم.',
                'category' => 'ttc',
                'tags' => ['ovulation'],
                'featured' => false,
                'body' => <<<'HTML'
<p>تست LH افزایش هورمون LH در ادرار را نشان می‌دهد؛ افزایشی که معمولاً یک تا دو روز پیش از تخمک‌گذاری رخ می‌دهد.</p>
<h2>از چه روزی شروع کنیم</h2>
<p>زمان شروع به طول چرخه‌ات بستگی دارد. اگر چرخه حدود ۲۸ روزه است، شروع از حدود روز دهم رایج است. ریتمی با توجه به چرخه‌های ثبت‌شده‌ات روزهای پیشنهادی را نشان می‌دهد.</p>
<h2>چطور بخوانیم</h2>
<ul>
<li>دستور سازنده کیت را دنبال کن</li>
<li>هر روز در ساعت تقریباً ثابتی تست بزن</li>
<li>خط تست هم‌رنگ یا پررنگ‌تر از خط کنترل معمولاً یعنی LH بالا رفته است</li>
</ul>
<h2>اگر نتیجه گیج‌کننده بود</h2>
<p>چرخه‌های نامنظم یا بعضی شرایط ممکن است خواندن نتیجه را سخت کند. در این حالت با پزشک یا ماما درباره راه‌های دیگر پیگیری صحبت کن.</p>
HTML,
            ],
        ];
    }
}

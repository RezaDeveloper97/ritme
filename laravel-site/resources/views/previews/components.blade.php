{{--
    Component kit preview (L3-01) — every shared component from docs/COMPONENTS.md on one page, with design copy, for
    review and screenshots (tools/shot.mjs). Route `preview.components` (/_components) exists outside production only.
    Demo data only: store links are in-page anchors, the QR encodes this site's home URL, images use the local
    illustrations (no media rows needed).
--}}
@php
    $appLinks = new \App\Domain\Settings\Data\AppLinksSettings(
        bazaar: '#bazaar', myket: '#myket', googlePlay: '#google-play', appStore: '#ios', webApp: null,
    );
    $qrUrl = url('/');
    $tools = [
        ['href' => route('tools'), 'icon' => 'calculator', 'color' => 'ttc', 'title' => 'محاسبه روزهای باروری', 'text' => 'پنجره تقریبی از روی چرخه‌ات', 'where' => 'روی سایت'],
        ['href' => route('tools'), 'icon' => 'flame', 'color' => 'cycle', 'title' => 'دفترچه درد', 'text' => 'ثبت شدت و محل درد', 'where' => 'در اپ'],
        ['href' => route('tools'), 'icon' => 'pill', 'color' => 'postpartum', 'title' => 'یادآور قرص پیشگیری', 'text' => 'با راهنمای «قرص جا افتاد»', 'where' => 'در اپ'],
    ];
    $services = [
        ['href' => route('services'), 'icon' => 'stethoscope', 'color' => 'postpartum', 'title' => 'پزشک و ماما', 'text' => 'ویزیت آنلاین یا حضوری با پرونده‌ای که خودت اجازه‌اش را می‌دهی.'],
        ['href' => route('services'), 'icon' => 'sparkle', 'color' => 'primary', 'title' => 'دستیار سلامت', 'text' => 'پاسخ به سؤال‌های روزمره و ارجاع به پزشک در صورت نیاز.'],
        ['href' => route('shop.index'), 'icon' => 'store', 'color' => 'muted', 'title' => 'فروشگاه', 'text' => 'سیسمونی و محصولات بهداشتی؛ جدا از داده سلامت.'],
    ];
    $readings = [
        ['href' => route('blog.show', 'period-pain'), 'title' => 'درد پریود؛ کی عادی است و کی ارزش پیگیری دارد؟', 'stage' => 'cycle', 'label' => 'چرخه', 'minutes' => 5],
        ['href' => route('blog.show', 'period-pain'), 'title' => 'ساک بیمارستان را از کی و با چه چیزهایی ببندیم؟', 'stage' => 'pregnancy', 'label' => 'بارداری', 'minutes' => 5],
        ['href' => route('blog.show', 'period-pain'), 'title' => 'شیردهی در هفته‌های اول: سؤال‌هایی که همه دارند', 'stage' => 'postpartum', 'label' => 'کودک', 'minutes' => 5],
    ];
@endphp
@extends('layouts.app', ['headerVariant' => 'light', 'navRoute' => 'faq', 'appCta' => true])

@section('content')
    <x-ui.page-intro eyebrow="L3-01" title="کیت کامپوننت‌های سایت" lead="همه اجزای مشترک صفحه‌ها، با متن طراحی؛ برای بازبینی و اسکرین‌شات. داده‌ها نمونه‌اند." id="kit-intro">
        <x-ui.chip-nav label="دسته‌های مجله" :items="[
            ['label' => 'همه', 'href' => '#kit-intro', 'active' => true],
            ['label' => 'چرخه و پریود', 'href' => '#kit-primitives'],
            ['label' => 'اقدام به بارداری', 'href' => '#kit-cards'],
            ['label' => 'بارداری', 'href' => '#kit-stage'],
            ['label' => 'یائسگی', 'href' => '#kit-forms'],
        ]"/>
    </x-ui.page-intro>

    {{-- Primitives --}}
    <x-ui.section id="kit-primitives" bg="surface" aria-labelledby="kit-primitives-title" class="flex flex-col gap-10">
        <x-ui.section-header eyebrow="برای هر مرحله" title="مرحله‌ات را انتخاب کن" id="kit-primitives-title" align="center"
            lead="ریتمی با مرحله زندگی‌ات عوض می‌شود؛ صفحه امروز، ثبت روزانه و ابزارها همان چیزی است که الان لازم داری."/>

        <div class="flex flex-wrap items-center gap-3">
            <x-ui.button href="#download" size="lg" icon="download">دانلود رایگان ریتمی</x-ui.button>
            <x-ui.button href="#kit-cards" variant="outline" size="lg">چطور کار می‌کند؟</x-ui.button>
            <x-ui.button href="#kit-cards" variant="ghost" icon-end="arrow-left">بیشتر</x-ui.button>
            <x-ui.eyebrow>ابزارهای این مرحله</x-ui.eyebrow>
            <x-ui.badge icon="shield-check">مدارک بررسی شد</x-ui.badge>
            <x-ui.pill dot>امروز ۱۷:۰۰</x-ui.pill>
            <x-ui.pill tone="surface" size="md" icon="shield-check" icon-class="text-stage-teen" class="border border-line">مدارک بررسی شد</x-ui.pill>
            <x-ui.pill tone="surface" size="xs" class="border border-line">پنبه ۱۰۰٪</x-ui.pill>
        </div>

        <div class="flex flex-wrap items-center gap-8">
            <x-ui.rating :value="4.8" :count="126" size="md"/>
            <x-ui.rating :value="4" stars/>
            <x-ui.price :amount="485000" :compare="520000"/>
            <x-ui.price :amount="320000" from unit="هر جلسه" size="lg"/>
            <x-ui.price :amount="1250000"/>
        </div>

        <x-ui.alert-emergency number="115">اگر خونریزی خیلی زیاد داری، درد شدیدی که با مسکن بهتر نمی‌شود یا خونریزی همراه سرگیجه و ضعف، معطل نکن و با پزشک یا اورژانس تماس بگیر.</x-ui.alert-emergency>

        <div class="grid grid-cols-2 gap-8 max-lg:grid-cols-1">
            <x-ui.accordion :items="[
                ['question' => 'اگر چرخه‌ام نامنظم باشد، ریتمی به دردم می‌خورد؟', 'answer' => 'بله. برای چرخه نامنظم بازه احتمالی و سطح اطمینان نشان می‌دهیم و می‌گوییم کدام ثبت‌ها تصویر را روشن‌تر می‌کند.'],
                ['question' => 'پیش‌بینی ریتمی برای پیشگیری از بارداری کافی است؟', 'answer' => 'نه. پیش‌بینی روزهای باروری روش پیشگیری نیست. برای انتخاب روش با پزشک یا ماما صحبت کن.'],
                ['question' => 'می‌توانم گزارش را به پزشکم نشان بدهم؟', 'answer' => 'بله؛ از بخش تحلیل یک گزارش ساده بساز و بفرست.'],
            ]"/>
            <div class="flex flex-col gap-2 rounded-5xl border border-line bg-surface px-7 py-2">
                <x-ui.toggle-row title="یادآور خرید قبل از پریود" text="فقط تاریخ تقریبی پریود بعدی"/>
                <x-ui.toggle-row title="پیشنهادهای سن کودک در خدمات شهری" text="فقط سن کودک" divided/>
                <x-ui.toggle-row variant="switch" title="یادآور خرید" text="فقط در اپ و فقط با اجازه خودت" divided/>
            </div>
        </div>
    </x-ui.section>

    {{-- Progress --}}
    <x-ui.section id="kit-progress" aria-labelledby="kit-progress-title" class="flex flex-col gap-10">
        <x-ui.section-header title="چطور ثبت کنم؟" id="kit-progress-title"/>
        <x-ui.steps :items="[
            ['title' => 'معرفی مجموعه', 'text' => 'نام، نوع خدمت، شهر و راه تماس'],
            ['title' => 'مکان و تصاویر', 'text' => 'سنجاق روی نقشه، عکس واقعی از فضا، رده سنی و ساعت کاری'],
            ['title' => 'خدمات و قیمت', 'text' => 'کلاس‌ها، مدت، ظرفیت، قیمت و نحوه رزرو'],
            ['title' => 'مدارک و بررسی', 'text' => 'مجوز فعالیت و مدرک مربی‌ها؛ بعد از بررسی صفحه‌ات منتشر می‌شود'],
        ]"/>
        <div class="flex gap-8 max-lg:flex-wrap">
            <x-ui.stepper class="w-70 shrink-0" :current="2" :steps="[
                ['label' => 'معرفی', 'hint' => 'کامل شد'],
                ['label' => 'مکان و تصاویر', 'hint' => 'در حال پر کردن'],
                ['label' => 'خدمات و قیمت'],
                ['label' => 'مدارک'],
            ]">
                <p class="m-0 mt-3.5 rounded-3xl bg-lavender p-4 text-sm-plus leading-relaxed font-semibold text-muted">پیش‌نویس خودکار ذخیره می‌شود؛ هر وقت خواستی برگرد و ادامه بده.</p>
            </x-ui.stepper>
            <div class="flex flex-1 flex-col gap-8">
                <x-ui.stepper variant="bar" label="مراحل خرید" :current="2" :steps="[['label' => 'سبد'], ['label' => 'ارسال و پرداخت'], ['label' => 'تأیید']]"/>
                <div class="rounded-6xl border border-line bg-surface px-8 pt-7 pb-3">
                    <x-ui.timeline label="وضعیت درخواست" :items="[
                        ['title' => 'درخواست ثبت شد', 'time' => 'امروز ۱۲:۴۰', 'state' => 'done'],
                        ['title' => 'بررسی مدارک', 'time' => '[زمان بررسی] روز کاری', 'state' => 'current'],
                        ['title' => 'تماس کارشناس ریتمی', 'time' => 'برای هماهنگی بازدید یا تکمیل اطلاعات', 'state' => 'todo'],
                        ['title' => 'انتشار صفحه مجموعه', 'time' => 'بعد از تأیید، با پیامک و ایمیل خبرت می‌کنیم', 'state' => 'todo'],
                    ]"/>
                </div>
            </div>
        </div>
        <div class="grid grid-cols-2 gap-8 max-lg:grid-cols-1">
            <x-ui.success-hero as="h2" icon="paper-plane" tone="lavender" title="درخواستت رسید">آب‌پری بعد از بررسی روی نقشه و در جست‌وجوی ریتمی نمایش داده می‌شود. کد پیگیری ۷۳۱۰۴</x-ui.success-hero>
            <x-ui.success-hero as="h2" title="سفارشت ثبت شد">
                کد سفارش ۵۲۰۸۴۱ · کالاهای هر فروشنده جداگانه می‌رسند.
                <x-slot:actions>
                    <x-ui.button href="#kit-progress" size="xl" icon="package">پیگیری سفارش</x-ui.button>
                    <x-ui.button href="#kit-cards" variant="outline" size="xl">ادامه خرید</x-ui.button>
                </x-slot:actions>
            </x-ui.success-hero>
        </div>
    </x-ui.section>

    {{-- Cards --}}
    <x-ui.section id="kit-cards" bg="surface" aria-labelledby="kit-cards-title" class="flex flex-col gap-10">
        <x-ui.section-header eyebrow="کارت‌ها" title="کارت‌های مشترک" id="kit-cards-title" more-href="#kit-cards" more-label="همه" more-icon="arrow-left"/>
        <div class="grid grid-cols-3 gap-4.5 max-sm:grid-cols-1">
            <x-cards.stage href="{{ route('stage.cycle') }}" icon="drop" color="cycle" title="چرخه و پریود" text="پیش‌بینی پریود با سطح اطمینان، ثبت علائم و شناختن الگوی خودت."/>
            <x-cards.service href="{{ route('services') }}" icon="stethoscope" color="postpartum" title="پزشک و ماما" text="ویزیت آنلاین یا حضوری با پرونده‌ای که خودت اجازه‌اش را می‌دهی."/>
            <x-cards.value icon="heart" color="cycle" title="بی‌قضاوت" text="هیچ بدنی «مشکل‌دار» نیست؛ هر بدن ریتم خودش را دارد."/>
        </div>
        <div class="grid grid-cols-3 gap-4 max-sm:grid-cols-1">
            <x-cards.feature href="{{ route('tools') }}" icon="calculator" color="ttc" title="محاسبه روزهای باروری" text="پنجره تقریبی از روی چرخه‌ات" where="روی سایت"/>
            <x-cards.value variant="compact" icon="lock" title="قفل اپ با رمز یا اثر انگشت"/>
            <div class="rounded-5xl border border-line bg-surface px-8 py-7"><x-cards.value variant="inline" icon="truck" title="ارسال سریع" text="[شرایط ارسال رایگان]"/></div>
        </div>
        <x-cards.article featured href="{{ route('blog.show', 'period-pain') }}" stage="cycle" label="چرخه" :minutes="6" title="درد پریود؛ کی عادی است و کی ارزش پیگیری دارد؟" excerpt="درد پریود شایع است، ولی هر دردی را نباید تحمل کرد. نشانه‌هایی که بهتر است با پزشک درمیان بگذاری." reviewer="[نام و تخصص متخصص]"/>
        <div class="grid grid-cols-5 gap-x-5 gap-y-7 max-lg:grid-cols-3 max-sm:grid-cols-2">
            <x-cards.product href="{{ route('shop.product', 'long-sleeve-cotton-bodysuit-3') }}" title="بادی آستین‌بلند نخی · ۳ عدد" seller="پوشاک پنبه‌ریز" :rating="4.7" :reviews="214" :price="485000" badge="پنبه ۱۰۰٪" illustration="product-bodysuit" tint="pregnancy"/>
            <x-cards.product href="{{ route('shop.product', 'long-sleeve-cotton-bodysuit-3') }}" title="سرهمی خواب پنبه‌ای" seller="خواب ناز" :rating="4.8" :reviews="96" :price="420000" :compare="460000" illustration="product-sleepsuit" tint="postpartum"/>
            <x-cards.product href="{{ route('shop.product', 'long-sleeve-cotton-bodysuit-3') }}" title="شیشه شیر ضدنفخ" seller="مادرانه" :rating="4.6" :reviews="58" :price="310000" illustration="product-bottle" tint="primary"/>
        </div>
        <div class="grid grid-cols-2 gap-5 max-sm:grid-cols-1">
            <x-cards.place href="{{ route('directory.place', 'ab-pari') }}" name="استخر مادر و کودک آب‌پری" :rating="4.8" :reviews="126" category="استخر مادر و کودک" district="ونک" distance="۱٫۲ کیلومتر" ages="مناسب ۶ ماه تا ۴ سال" :price-from="320000" :slots="['امروز ۱۷:۰۰']" verified illustration="place-cover-pool" selected/>
            <x-cards.place href="{{ route('directory.place', 'ab-pari') }}" name="خانه بازی تاب‌تاب" :rating="4.6" :reviews="84" category="خانه بازی" district="یوسف‌آباد" distance="۲٫۴ کیلومتر" ages="مناسب ۱ تا ۶ سال" :price-from="180000" illustration="place-cover-playhouse"/>
        </div>
        <div class="grid grid-cols-3 gap-4 max-sm:grid-cols-1">
            <x-cards.review name="مادر پرنیا" :rating="5" meta="خرید تأییدشده · سایز ۳-۶ ماه" text="پارچه نرم است و بعد از چند بار شستن جمع نشد."/>
            <x-cards.review avatar name="مادر رها" :rating="5" meta="۲ هفته پیش · رزرو از ریتمی" text="مربی خیلی صبور بود و آب واقعاً گرم. اتاق شیردهی تمیز و آرام بود."/>
        </div>
        <div class="grid grid-cols-2 gap-8 max-lg:grid-cols-1">
            <x-ui.gallery alt="استخر مادر و کودک آب‌پری" :images="[]" :total="12" more-href="#kit-cards"/>
            <x-ui.gallery variant="product" alt="بادی آستین‌بلند نخی" :images="[]"/>
        </div>
    </x-ui.section>

    {{-- Stage blocks --}}
    <div id="kit-stage">
        <x-stage.tools-block :items="$tools"/>
        <x-stage.help-block :items="$services"/>
        <x-stage.readings :posts="$readings" :more-href="route('blog.index')"/>
    </div>

    {{-- Section blocks --}}
    <x-ui.section id="kit-blocks" aria-label="بلوک‌های بخش" class="flex flex-col gap-10">
        <x-ui.promo-split :items="[
            ['href' => route('shop.category', 'baby-clothes'), 'tone' => 'lavender', 'eyebrow' => 'سیسمونی و نوزاد', 'title' => 'برای آمدن نوزاد آماده شو', 'text' => 'لباس، خواب، تغذیه و بیرون رفتن؛ همراه با لیست سیسمونی قابل تنظیم.', 'cta' => 'ورود به فروشگاه', 'illustration' => 'product-bodysuit'],
            ['href' => route('shop.category', 'baby-clothes'), 'tone' => 'blush', 'eyebrow' => 'آرایشی و بهداشتی', 'title' => 'مراقبت از خودت، هر روز', 'text' => 'بهداشت بانوان، مراقبت پوست و مو، با شناسه IRC و ترکیبات کامل هر محصول.', 'cta' => 'ورود به فروشگاه', 'illustration' => 'product-serum'],
        ]"/>
        <x-ui.promo-split :items="[
            ['href' => route('directory.index'), 'tone' => 'lavender', 'icon' => 'map', 'title' => 'خدمات مادر و کودک', 'text' => 'کلاس مادر و کودک، استخر، خانه بازی و کارگاه‌ها روی نقشه؛ با صفحه هر مجموعه و رزرو.', 'cta' => 'جست‌وجو در نقشه'],
            ['href' => route('shop.index'), 'tone' => 'dashed', 'icon' => 'store', 'title' => 'فروشگاه', 'text' => 'سیسمونی، لباس نوزاد و محصولات آرایشی و بهداشتی از فروشنده‌های بررسی‌شده. فروشگاه از داده سلامت تو جداست.', 'cta' => 'ورود به فروشگاه'],
        ]"/>
        <x-ui.newsletter/>
        <div class="flex gap-8 max-lg:flex-wrap">
            <x-ui.app-cta variant="aside" id="" heading-id="kit-aside-title" icon="bell" class="w-110 max-sm:w-full" title="یادآور و رزروهایت در اپ ریتمی" lead="رزروت در تقویم ریتمی هم هست. با اپ، یک روز قبل و دو ساعت قبل یادت می‌اندازیم و همه رزروها یک‌جا می‌ماند." :links="$appLinks"/>
            <x-ui.app-cta variant="strip" id="" heading-id="kit-strip-title" class="flex-1 self-start" title="سفارش‌ها و لیست سیسمونی در اپ" lead="در اپ ریتمی وضعیت سفارش را لحظه‌ای ببین؛ کالاهای رسیده خودکار در لیست سیسمونی «دارم» می‌شوند." :links="$appLinks"/>
        </div>
    </x-ui.section>

    {{-- Forms --}}
    <x-ui.section id="kit-forms" bg="surface" aria-labelledby="kit-forms-title" class="flex flex-col gap-8">
        <x-ui.section-header title="فرم تماس" size="sm" id="kit-forms-title"/>
        <form class="flex max-w-140 flex-col gap-4.5" action="#kit-forms" method="get" novalidate>
            <x-ui.form.field label="روش پرداخت" as="fieldset">
                <div class="grid grid-cols-2 gap-3 max-sm:grid-cols-1">
                    <x-ui.form.radio-card name="pay" value="online" icon="card" title="درگاه بانکی" text="همه کارت‌های عضو شتاب" checked/>
                    <x-ui.form.radio-card name="pay" value="cod" icon="credit-card" title="پرداخت در محل" text="فقط برای سفارش‌های زیر [سقف مبلغ]"/>
                </div>
            </x-ui.form.field>
            <div class="grid grid-cols-2 gap-3 max-sm:grid-cols-1">
                <x-ui.form.field for="kit-name" label="نام" required>
                    <x-ui.form.input id="kit-name" name="name" placeholder="نام تو" autocomplete="name"/>
                </x-ui.form.field>
                <x-ui.form.field for="kit-contact" label="ایمیل یا شماره همراه" error="این فیلد لازم است.">
                    <x-ui.form.input id="kit-contact" name="contact" placeholder="برای پاسخ" invalid described/>
                </x-ui.form.field>
            </div>
            <x-ui.form.field for="kit-city" label="شهر">
                <x-ui.form.select id="kit-city" name="city" :options="['tehran' => 'تهران', 'karaj' => 'کرج']" placeholder="انتخاب کن"/>
            </x-ui.form.field>
            <x-ui.form.field for="kit-message" label="پیام" hint="اطلاعات سلامت را در این فرم ننویس؛ برای مسائل حساب کاربری از پشتیبانی داخل اپ استفاده کن.">
                <x-ui.form.textarea id="kit-message" name="message" placeholder="بنویس…" described/>
            </x-ui.form.field>
            <x-ui.form.checkbox name="terms" value="1">شرایط استفاده را خواندم</x-ui.form.checkbox>
            <div><x-ui.button type="submit" size="xl" icon="arrow-left">ارسال پیام</x-ui.button></div>
        </form>
    </x-ui.section>

    <x-ui.app-cta title="ریتمی را رایگان نصب کن" lead="در اپ، مرحله‌ات را انتخاب کن؛ بقیه‌اش با ماست." :links="$appLinks" :qr-url="$qrUrl"/>
    <x-ui.promise-banner href="{{ route('social-responsibility') }}"/>
@endsection

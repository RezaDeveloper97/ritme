<?php

declare(strict_types=1);

namespace App\Domain\Seo\Analysis;

use App\Support\Text\PersianDigits;

/**
 * Persian-aware on-page SEO analyser for posts, products, places and static pages. Pure and deterministic: the caller
 * resolves every input (AnalysisInput), the analyser only reads strings — no database, cache, HTTP or clock — so it
 * can run on every debounced keystroke of the admin form.
 *
 * Checks (keys): title_length, description_length, focus_keyword, keyword_title, keyword_description, keyword_heading,
 * keyword_slug, keyword_first_paragraph, keyword_subheadings, keyword_image_alt, keyword_density, content_length,
 * heading_single_h1, heading_hierarchy, image_alt, internal_links, external_links, slug, readability_sentences,
 * readability_paragraphs, duplicate_title, duplicate_description, red_lines, diagnosis_claims, cornerstone.
 * Content checks are skipped when the input has no body (static pages render theirs from a template).
 *
 * Score = weighted share of earned credit (pass 1, warning ½, error 0; info not scored); a red-line or diagnosis
 * error caps it at 40.
 */
final class SeoAnalyzer
{
    public const DENSITY_MIN = 0.5;

    public const DENSITY_MAX = 2.5;

    /** Words of body text needed before density and readability are meaningful. */
    public const MIN_WORDS_FOR_RATIOS = 100;

    public const SENTENCE_AVG_MAX = 20;

    public const LONG_SENTENCE_WORDS = 25;

    /** Share of long sentences tolerated (percent). */
    public const LONG_SENTENCE_SHARE = 25;

    public const PARAGRAPH_MAX_WORDS = 150;

    public const SLUG_MAX_CHARS = 75;

    public const RED_LINE_SCORE_CAP = 40;

    private const SLUG_STOPWORDS = ['و', 'در', 'به', 'از', 'که', 'را', 'با', 'این', 'آن', 'برای', 'یا', 'تا',
        'a', 'an', 'the', 'and', 'or', 'of', 'to', 'in', 'on', 'for', 'with', 'is'];

    public function analyze(AnalysisInput $input): SeoAnalysis
    {
        $document = $input->contentHtml === null ? null : ContentDocument::parse($input->contentHtml);
        $keyword = trim($input->focusKeyword);

        $checks = [
            $this->titleLength($input->title),
            $this->descriptionLength($input->description),
        ];
        if ($input->duplicateTitle !== null) {
            $checks[] = $this->duplicate('duplicate_title', 'عنوان تکراری', $input->duplicateTitle, 'عنوان سئو');
        }
        if ($input->duplicateDescription !== null && trim($input->description) !== '') {
            $checks[] = $this->duplicate('duplicate_description', 'توضیح تکراری', $input->duplicateDescription, 'توضیح متا');
        }

        $checks = [...$checks, ...$this->keywordChecks($input, $keyword, $document)];

        if ($document !== null) {
            $checks[] = $this->contentLength($input, $document);
            $checks[] = $this->singleH1($document);
            $checks[] = $this->hierarchy($document);
            $checks[] = $this->imageAlt($document);
            $checks[] = $this->internalLinks($input, $document);
            $checks[] = $this->externalLinks($input, $document);
            $checks[] = $this->sentences($document);
            $checks[] = $this->paragraphs($document);
        }

        $checks[] = $this->slug($input->slug);

        $all = implode("\n", [$input->title, $input->heading, $input->description, $document === null ? '' : $document->text]);
        $checks[] = $this->redLines($all);
        $checks[] = $this->claims($all);

        if ($input->cornerstone) {
            $checks[] = new SeoCheck('cornerstone', 'محتوای ستون', Severity::Info,
                'این محتوا «ستون» علامت خورده است: حداقل '.$this->n($input->type->minWords(true)).' واژه و '
                .$this->n($input->type->minInternalLinks(true)).' پیوند داخلی سنجیده می‌شود.');
        }

        return new SeoAnalysis($this->score($checks), $this->sorted($checks));
    }

    private function titleLength(string $title): SeoCheck
    {
        $label = 'طول عنوان';
        $chars = TextWidth::chars($title);
        if ($chars === 0) {
            return new SeoCheck('title_length', $label, Severity::Error, 'عنوان صفحه خالی است.', 2);
        }

        $px = TextWidth::pixels($title, TextWidth::TITLE_FONT_PX);
        $size = $this->n($chars).' نویسه، حدود '.$this->n($px).' از '.$this->n(TextWidth::TITLE_MAX_PX).' پیکسل';
        [$min, $max] = TextWidth::TITLE_CHARS;

        return match (true) {
            $px > TextWidth::TITLE_MAX_PX => new SeoCheck('title_length', $label, Severity::Error, "عنوان در گوگل بریده می‌شود ({$size}). کوتاه‌ترش کنید.", 2),
            $chars < $min => new SeoCheck('title_length', $label, Severity::Warning, "عنوان کوتاه است ({$size}). پیشنهاد: {$this->n($min)} تا {$this->n($max)} نویسه.", 2),
            $chars > $max => new SeoCheck('title_length', $label, Severity::Warning, "عنوان کمی بلند است ({$size}). پیشنهاد: حداکثر {$this->n($max)} نویسه.", 2),
            default => new SeoCheck('title_length', $label, Severity::Pass, "طول عنوان مناسب است ({$size}).", 2),
        };
    }

    private function descriptionLength(string $description): SeoCheck
    {
        $label = 'طول توضیح متا';
        $chars = TextWidth::chars($description);
        if ($chars === 0) {
            return new SeoCheck('description_length', $label, Severity::Warning, 'توضیح متا خالی است؛ گوگل خودش تکه‌ای از متن را نشان می‌دهد.');
        }

        $px = TextWidth::pixels($description, TextWidth::DESCRIPTION_FONT_PX);
        $size = $this->n($chars).' نویسه، حدود '.$this->n($px).' از '.$this->n(TextWidth::DESCRIPTION_MAX_PX).' پیکسل';
        [$min, $max] = TextWidth::DESCRIPTION_CHARS;

        return match (true) {
            $px > TextWidth::DESCRIPTION_MAX_PX => new SeoCheck('description_length', $label, Severity::Warning, "توضیح در گوگل بریده می‌شود ({$size})."),
            $chars < $min => new SeoCheck('description_length', $label, Severity::Warning, "توضیح کوتاه است ({$size}). پیشنهاد: {$this->n($min)} تا {$this->n($max)} نویسه."),
            $chars > $max => new SeoCheck('description_length', $label, Severity::Warning, "توضیح بلند است ({$size}). پیشنهاد: حداکثر {$this->n($max)} نویسه."),
            default => new SeoCheck('description_length', $label, Severity::Pass, "طول توضیح مناسب است ({$size})."),
        };
    }

    private function duplicate(string $key, string $label, bool $duplicate, string $field): SeoCheck
    {
        return $duplicate
            ? new SeoCheck($key, $label, Severity::Error, "{$field} همین حالا برای صفحه دیگری از سایت ثبت شده است؛ متن یکتا بنویسید.")
            : new SeoCheck($key, $label, Severity::Pass, "{$field} در سایت یکتاست.");
    }

    /**
     * @return list<SeoCheck>
     */
    private function keywordChecks(AnalysisInput $input, string $keyword, ?ContentDocument $document): array
    {
        if (PersianText::pattern($keyword) === null) {
            return [new SeoCheck('focus_keyword', 'کلیدواژه کانونی', Severity::Warning, 'کلیدواژه کانونی تعیین نشده است؛ بدون آن بررسی‌های کلیدواژه انجام نمی‌شود.', 2)];
        }

        $q = "«{$keyword}»";
        $checks = [
            $this->presence('keyword_title', 'کلیدواژه در عنوان', PersianText::contains($keyword, $input->title),
                "{$q} در عنوان سئو آمده است.", "{$q} در عنوان سئو نیامده است؛ ترجیحاً نزدیک ابتدای عنوان بیاورید.", Severity::Error, 2),
            $this->presence('keyword_description', 'کلیدواژه در توضیح متا', PersianText::contains($keyword, $input->description),
                "{$q} در توضیح متا آمده است.", "{$q} در توضیح متا نیامده است."),
        ];

        if (trim($input->heading) !== '') {
            $checks[] = $this->presence('keyword_heading', 'کلیدواژه در تیتر اصلی (h1)', PersianText::contains($keyword, $input->heading),
                "{$q} در تیتر اصلی صفحه آمده است.", "{$q} در تیتر اصلی صفحه (h1) نیامده است.");
        }

        $checks[] = $this->keywordSlug($keyword, $input->slug);

        if ($document === null) {
            return $checks;
        }

        $first = $document->paragraphs[0] ?? '';
        $checks[] = $first === ''
            ? new SeoCheck('keyword_first_paragraph', 'کلیدواژه در پاراگراف اول', Severity::Warning, 'متن هنوز پاراگرافی ندارد.')
            : $this->presence('keyword_first_paragraph', 'کلیدواژه در پاراگراف اول', PersianText::contains($keyword, $first),
                "{$q} در پاراگراف اول آمده است.", "{$q} در پاراگراف اول نیامده است؛ خواننده و گوگل موضوع را از همان ابتدا می‌فهمند.");

        $subheadings = array_values(array_filter($document->headings, static fn (array $h): bool => $h['level'] >= 2));
        if ($subheadings === []) {
            $checks[] = new SeoCheck('keyword_subheadings', 'کلیدواژه در زیرتیترها',
                $document->wordCount >= 300 ? Severity::Warning : Severity::Info, 'متن زیرتیتر (h2/h3) ندارد.');
        } else {
            $hits = count(array_filter($subheadings, static fn (array $h): bool => PersianText::contains($keyword, $h['text'])));
            $checks[] = $hits > 0
                ? new SeoCheck('keyword_subheadings', 'کلیدواژه در زیرتیترها', Severity::Pass, "{$q} در {$this->n($hits)} زیرتیتر از {$this->n(count($subheadings))} آمده است.")
                : new SeoCheck('keyword_subheadings', 'کلیدواژه در زیرتیترها', Severity::Warning, "{$q} یا بخشی از آن در هیچ زیرتیتری نیامده است.");
        }

        $alts = array_values(array_filter($document->imageAlts, static fn (?string $alt): bool => $alt !== null));
        if ($document->imageAlts === []) {
            $checks[] = new SeoCheck('keyword_image_alt', 'کلیدواژه در متن جایگزین تصویر', Severity::Info, 'متن تصویری ندارد.');
        } else {
            $hit = array_filter($alts, static fn (string $alt): bool => PersianText::contains($keyword, $alt)) !== [];
            $checks[] = $this->presence('keyword_image_alt', 'کلیدواژه در متن جایگزین تصویر', $hit,
                "{$q} در متن جایگزین (alt) دست‌کم یک تصویر آمده است.", "{$q} در متن جایگزین (alt) هیچ تصویری نیامده است.");
        }

        $checks[] = $this->density($keyword, $document);

        return $checks;
    }

    private function presence(string $key, string $label, bool $ok, string $yes, string $no, Severity $missing = Severity::Warning, int $weight = 1): SeoCheck
    {
        return new SeoCheck($key, $label, $ok ? Severity::Pass : $missing, $ok ? $yes : $no, $weight);
    }

    private function keywordSlug(string $keyword, string $slug): SeoCheck
    {
        $label = 'کلیدواژه در نامک';
        $readable = $this->slugText($slug);
        if ($readable === '') {
            return new SeoCheck('keyword_slug', $label, Severity::Info, 'نامک هنوز ساخته نشده است.');
        }
        if (PersianText::contains($keyword, $readable)) {
            return new SeoCheck('keyword_slug', $label, Severity::Pass, "«{$keyword}» در نامک آمده است.");
        }
        if (PersianText::isLatin($readable) && ! PersianText::isLatin($keyword)) {
            return new SeoCheck('keyword_slug', $label, Severity::Info, 'نامک لاتین است؛ مطمئن شوید معادل کلیدواژه در آن آمده است.');
        }

        return new SeoCheck('keyword_slug', $label, Severity::Warning, "«{$keyword}» در نامک نیامده است.");
    }

    private function density(string $keyword, ContentDocument $document): SeoCheck
    {
        $label = 'تراکم کلیدواژه';
        if ($document->wordCount < self::MIN_WORDS_FOR_RATIOS) {
            return new SeoCheck('keyword_density', $label, Severity::Info, 'متن برای سنجش تراکم کلیدواژه هنوز کوتاه است.');
        }

        $occurrences = PersianText::count($keyword, $document->text);
        $keywordWords = max(1, PersianText::wordCount($keyword));
        $density = $occurrences * $keywordWords / $document->wordCount * 100;
        $text = $this->n($occurrences).' بار، '.PersianDigits::number($density, 1).'٪';

        return match (true) {
            $occurrences === 0 => new SeoCheck('keyword_density', $label, Severity::Warning, "«{$keyword}» در متن نیامده است."),
            $density < self::DENSITY_MIN => new SeoCheck('keyword_density', $label, Severity::Warning, "تراکم کم است ({$text}). پیشنهاد: ۰٫۵ تا ۲٫۵٪."),
            $density > self::DENSITY_MAX => new SeoCheck('keyword_density', $label, Severity::Warning, "تکرار زیاد است ({$text})؛ طبیعی‌تر بنویسید و از هم‌معنی‌ها کمک بگیرید."),
            default => new SeoCheck('keyword_density', $label, Severity::Pass, "تراکم کلیدواژه مناسب است ({$text})."),
        };
    }

    private function contentLength(AnalysisInput $input, ContentDocument $document): SeoCheck
    {
        $label = 'طول محتوا';
        $min = $input->type->minWords($input->cornerstone);
        $words = $document->wordCount;
        $text = $this->n($words).' واژه؛ پیشنهاد برای '.$input->type->label().': دست‌کم '.$this->n($min);

        return match (true) {
            $words === 0 => new SeoCheck('content_length', $label, $input->type === ContentType::Archive ? Severity::Info : Severity::Error, 'متن خالی است.', 2),
            $words < intdiv($min, 2) && $input->type !== ContentType::Archive => new SeoCheck('content_length', $label, Severity::Error, "متن خیلی کوتاه است ({$text}).", 2),
            $words < $min => new SeoCheck('content_length', $label, Severity::Warning, "متن کوتاه است ({$text}).", 2),
            default => new SeoCheck('content_length', $label, Severity::Pass, "طول متن مناسب است ({$this->n($words)} واژه).", 2),
        };
    }

    private function singleH1(ContentDocument $document): SeoCheck
    {
        $h1 = count(array_filter($document->headings, static fn (array $h): bool => $h['level'] === 1));

        return $h1 === 0
            ? new SeoCheck('heading_single_h1', 'یک تیتر اصلی', Severity::Pass, 'تیتر اصلی (h1) فقط عنوان صفحه است.')
            : new SeoCheck('heading_single_h1', 'یک تیتر اصلی', Severity::Error, 'متن '.$this->n($h1).' تیتر h1 دارد؛ عنوان صفحه خودش h1 است، داخل متن از تیتر ۲ و ۳ استفاده کنید.');
    }

    private function hierarchy(ContentDocument $document): SeoCheck
    {
        $label = 'ساختار تیترها';
        $previous = 1; // the page title
        $skips = [];
        foreach ($document->headings as $heading) {
            if ($heading['level'] > $previous + 1) {
                $skips[] = 'h'.$previous.'→h'.$heading['level'].($heading['text'] !== '' ? ' («'.mb_strimwidth($heading['text'], 0, 30, '…').'»)' : '');
            }
            $previous = max(1, $heading['level']);
        }

        if ($skips !== []) {
            return new SeoCheck('heading_hierarchy', $label, Severity::Warning, 'سطح تیترها جا افتاده است: '.implode('، ', $skips).'.');
        }
        if ($document->headings === [] && $document->wordCount >= 300) {
            return new SeoCheck('heading_hierarchy', $label, Severity::Warning, 'متن بلند است اما زیرتیتر ندارد؛ با تیتر ۲ بخش‌بندی کنید.');
        }

        return new SeoCheck('heading_hierarchy', $label, Severity::Pass, 'ترتیب تیترها درست است.');
    }

    private function imageAlt(ContentDocument $document): SeoCheck
    {
        $label = 'متن جایگزین تصاویر';
        $total = count($document->imageAlts);
        if ($total === 0) {
            return new SeoCheck('image_alt', $label, Severity::Info, 'متن تصویری ندارد.');
        }

        $missing = count(array_filter($document->imageAlts, static fn (?string $alt): bool => $alt === null));

        return $missing === 0
            ? new SeoCheck('image_alt', $label, Severity::Pass, 'همه '.$this->n($total).' تصویر متن جایگزین (alt) دارند.')
            : new SeoCheck('image_alt', $label, Severity::Error, $this->n($missing).' تصویر از '.$this->n($total).' متن جایگزین (alt) ندارد.');
    }

    private function internalLinks(AnalysisInput $input, ContentDocument $document): SeoCheck
    {
        $label = 'پیوندهای داخلی';
        $count = count(array_filter($document->links, fn (array $l): bool => $this->linkKind($l['href'], $input->ownHosts) === 'internal'));
        $min = $input->type->minInternalLinks($input->cornerstone);

        if ($min === 0) {
            return new SeoCheck('internal_links', $label, Severity::Info, $this->n($count).' پیوند داخلی.');
        }

        return $count >= $min
            ? new SeoCheck('internal_links', $label, Severity::Pass, $this->n($count).' پیوند داخلی به صفحه‌های دیگر سایت.')
            : new SeoCheck('internal_links', $label, Severity::Warning, $this->n($count).' پیوند داخلی؛ دست‌کم '.$this->n($min).' پیوند به مقاله‌ها یا صفحه‌های مرتبط بگذارید.');
    }

    private function externalLinks(AnalysisInput $input, ContentDocument $document): SeoCheck
    {
        $label = 'پیوندهای بیرونی';
        $external = array_values(array_filter($document->links, fn (array $l): bool => $this->linkKind($l['href'], $input->ownHosts) === 'external'));
        $count = count($external);
        if ($count === 0) {
            return new SeoCheck('external_links', $label, Severity::Info, $input->type === ContentType::Post
                ? 'پیوند بیرونی ندارد؛ برای محتوای سلامت، پیوند به منبع معتبر (در متن یا بخش منابع) اعتماد می‌سازد.'
                : 'پیوند بیرونی ندارد.');
        }

        $unsafe = count(array_filter($external, static fn (array $l): bool => $l['blank'] && array_intersect(['noopener', 'noreferrer'], $l['rel']) === []));
        $nofollow = count(array_filter($external, static fn (array $l): bool => array_intersect(['nofollow', 'sponsored', 'ugc'], $l['rel']) !== []));
        $summary = $this->n($count).' پیوند بیرونی'.($nofollow > 0 ? '، '.$this->n($nofollow).' با nofollow/sponsored' : '');

        return $unsafe > 0
            ? new SeoCheck('external_links', $label, Severity::Warning, "{$summary}؛ {$this->n($unsafe)} پیوند در زبانه جدید باز می‌شود و rel=\"noopener\" ندارد.")
            : new SeoCheck('external_links', $label, Severity::Pass, "{$summary}.");
    }

    private function sentences(ContentDocument $document): SeoCheck
    {
        $label = 'طول جمله‌ها';
        if ($document->wordCount < self::MIN_WORDS_FOR_RATIOS) {
            return new SeoCheck('readability_sentences', $label, Severity::Info, 'متن برای سنجش خوانایی هنوز کوتاه است.');
        }

        $lengths = [];
        foreach ($document->paragraphs as $paragraph) {
            foreach (PersianText::sentences($paragraph) as $sentence) {
                $lengths[] = PersianText::wordCount($sentence);
            }
        }
        $lengths = $lengths === [] ? [$document->wordCount] : $lengths;
        $average = array_sum($lengths) / count($lengths);
        $long = count(array_filter($lengths, static fn (int $n): bool => $n > self::LONG_SENTENCE_WORDS));
        $share = $long / count($lengths) * 100;
        $text = 'میانگین '.PersianDigits::number($average, 1).' واژه در جمله';

        return $average > self::SENTENCE_AVG_MAX || $share > self::LONG_SENTENCE_SHARE
            ? new SeoCheck('readability_sentences', $label, Severity::Warning, "جمله‌ها بلندند ({$text}، {$this->n((int) round($share))}٪ بیش از {$this->n(self::LONG_SENTENCE_WORDS)} واژه)؛ جمله‌های بلند را بشکنید.")
            : new SeoCheck('readability_sentences', $label, Severity::Pass, "جمله‌ها خوش‌خوان‌اند ({$text}).");
    }

    private function paragraphs(ContentDocument $document): SeoCheck
    {
        $label = 'طول پاراگراف‌ها';
        if ($document->wordCount < self::MIN_WORDS_FOR_RATIOS) {
            return new SeoCheck('readability_paragraphs', $label, Severity::Info, 'متن برای سنجش پاراگراف‌ها هنوز کوتاه است.');
        }

        $long = count(array_filter($document->paragraphs, static fn (string $p): bool => PersianText::wordCount($p) > self::PARAGRAPH_MAX_WORDS));

        return $long > 0
            ? new SeoCheck('readability_paragraphs', $label, Severity::Warning, $this->n($long).' پاراگراف بیش از '.$this->n(self::PARAGRAPH_MAX_WORDS).' واژه دارد؛ در موبایل خسته‌کننده است، کوتاه‌ترش کنید.')
            : new SeoCheck('readability_paragraphs', $label, Severity::Pass, 'طول پاراگراف‌ها مناسب است.');
    }

    private function slug(string $slug): SeoCheck
    {
        $label = 'نامک (slug)';
        $readable = $this->slugText($slug);
        if ($readable === '') {
            return new SeoCheck('slug', $label, Severity::Info, 'نامک خالی است و از عنوان ساخته می‌شود.');
        }

        $length = mb_strlen($readable);
        $stopwords = array_values(array_intersect(explode(' ', mb_strtolower($readable)), self::SLUG_STOPWORDS));
        $problems = [];
        if ($length > self::SLUG_MAX_CHARS) {
            $problems[] = 'بلند است ('.$this->n($length).' نویسه؛ حداکثر '.$this->n(self::SLUG_MAX_CHARS).')';
        }
        if ($stopwords !== []) {
            $problems[] = 'واژه‌های زائد دارد ('.implode('، ', array_unique($stopwords)).')';
        }

        return $problems === []
            ? new SeoCheck('slug', $label, Severity::Pass, 'نامک کوتاه و تمیز است.')
            : new SeoCheck('slug', $label, Severity::Warning, 'نامک '.implode(' و ', $problems).'.');
    }

    private function redLines(string $text): SeoCheck
    {
        $found = RedLines::words($text);

        return $found === []
            ? new SeoCheck('red_lines', 'واژه‌های ممنوع', Severity::Pass, 'واژه قطعی‌گو یا وعده تضمینی ندارد.', 2)
            : new SeoCheck('red_lines', 'واژه‌های ممنوع', Severity::Error, 'واژه‌های خط قرمز برند: «'.implode('»، «', $found).'». بدون قطعیت و وعده بنویسید.', 2);
    }

    private function claims(string $text): SeoCheck
    {
        $found = RedLines::claims($text);

        return $found === []
            ? new SeoCheck('diagnosis_claims', 'ادعای تشخیص یا درمان', Severity::Pass, 'ادعای تشخیص یا درمان ندارد.', 2)
            : new SeoCheck('diagnosis_claims', 'ادعای تشخیص یا درمان', Severity::Error, implode('، ', $found).'. ریتمی تشخیص نمی‌دهد؛ خواننده را به پزشک ارجاع دهید.', 2);
    }

    /**
     * internal | external | other (mailto:, tel:, #anchor, javascript:).
     *
     * @param  list<string>  $ownHosts
     */
    private function linkKind(string $href, array $ownHosts): string
    {
        if (str_starts_with($href, '#') || preg_match('/^(mailto|tel|sms|javascript|data):/i', $href) === 1) {
            return 'other';
        }

        $parts = parse_url($href);
        if ($parts === false) {
            return 'other';
        }
        if (! isset($parts['host'])) {
            return isset($parts['scheme']) ? 'other' : 'internal';
        }

        $host = strtolower($parts['host']);
        $own = array_map(static fn (string $h): string => strtolower(preg_replace('/^www\./i', '', $h) ?? $h), $ownHosts);

        return in_array(preg_replace('/^www\./', '', $host), $own, true) ? 'internal' : 'external';
    }

    private function slugText(string $slug): string
    {
        $slug = trim(rawurldecode(trim($slug)), '/');
        $slug = (string) preg_replace('/[-_\s]+/u', ' ', $slug);

        return trim($slug);
    }

    /**
     * @param  list<SeoCheck>  $checks
     */
    private function score(array $checks): int
    {
        $earned = 0.0;
        $total = 0;
        $capped = false;
        foreach ($checks as $check) {
            $credit = $check->severity->credit();
            if ($credit === null) {
                continue;
            }
            $earned += $credit * $check->weight;
            $total += $check->weight;
            $capped = $capped || ($check->severity === Severity::Error && in_array($check->key, ['red_lines', 'diagnosis_claims'], true));
        }

        $score = $total === 0 ? 0 : (int) round($earned / $total * 100);

        return $capped ? min($score, self::RED_LINE_SCORE_CAP) : $score;
    }

    /**
     * Errors first, then warnings, info, passes; original order inside each group.
     *
     * @param  list<SeoCheck>  $checks
     * @return list<SeoCheck>
     */
    private function sorted(array $checks): array
    {
        $order = array_keys($checks);
        usort($order, static fn (int $a, int $b): int => [$checks[$a]->severity->rank(), $a] <=> [$checks[$b]->severity->rank(), $b]);

        return array_map(static fn (int $i): SeoCheck => $checks[$i], $order);
    }

    private function n(int $value): string
    {
        return PersianDigits::toPersian($value);
    }
}

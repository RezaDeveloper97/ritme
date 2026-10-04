<?php

declare(strict_types=1);

namespace App\Domain\Seo\StaticPages;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Data\StaticPageDefaults;
use App\Domain\Seo\Support\DescriptionText;
use Illuminate\Contracts\Translation\Translator;

/**
 * The default (no admin override) title / description of every static page — the same lang strings and title mode
 * its controller uses, so the admin list can show "default vs override" without rendering pages. Kept honest by
 * tests/Feature/Admin/StaticPageSeoTest.php, which renders each indexable page and compares its <title>.
 */
final class StaticPageSeoDefaults
{
    /** FaqController::TITLE / ::DESCRIPTION (the FAQ page has no lang file). */
    public const FAQ_TITLE = 'سؤالات متداول — جواب سؤال‌های رایج درباره ریتمی';

    public const FAQ_DESCRIPTION = 'جواب سؤال‌های رایج درباره ریتمی: شروع کار، دقت پیش‌بینی‌ها و سلامت، حریم خصوصی و داده، پرداخت و اشتراک، خدمات و فروشگاه.';

    public function __construct(private readonly Translator $translator) {}

    public function for(StaticPage $page): StaticPageDefaults
    {
        return match ($page) {
            StaticPage::Home => $this->complete('home.seo.title', 'home.seo.description'),
            StaticPage::Cycle, StaticPage::Ttc, StaticPage::Pregnancy, StaticPage::Postpartum,
            StaticPage::Menopause, StaticPage::Teen => $this->templated(
                $this->text('stages/'.substr($page->value, strlen('stage.')).'.seo.title', $page->label()),
                $this->text('stages/'.substr($page->value, strlen('stage.')).'.seo.description'),
            ),
            StaticPage::Services => $this->complete('services.seo.title', 'services.seo.description'),
            StaticPage::Plus => $this->complete('plus.seo.title', 'plus.seo.description'),
            StaticPage::Tools => $this->complete('tools.seo.title', 'tools.seo.description'),
            StaticPage::About => $this->complete('about.seo.title', 'about.seo.description'),
            StaticPage::SocialResponsibility => $this->complete('social.seo.title', 'social.seo.description'),
            StaticPage::Privacy => $this->complete('privacy.seo.title', 'privacy.seo.description'),
            StaticPage::Terms => $this->complete('terms.seo.title', 'terms.seo.description'),
            StaticPage::Contact => $this->complete('contact.seo.title', 'contact.seo.description'),
            StaticPage::Faq => new StaticPageDefaults(self::FAQ_TITLE, true, self::FAQ_DESCRIPTION),
            // Listing controllers: title through the template, description trimmed like an excerpt.
            StaticPage::Blog => $this->templated(
                $this->text('blog.index.seo_title', $page->label()),
                DescriptionText::fromExcerpt($this->text('blog.index.seo_description')),
            ),
            StaticPage::Directory => $this->templated(
                $this->text('directory.index.seo_title', $page->label()),
                DescriptionText::fromExcerpt($this->text('directory.index.seo_description')),
            ),
            StaticPage::DirectoryBusiness => $this->complete('directory.business_page.seo.title', 'directory.business_page.seo.description'),
            StaticPage::DirectoryJoin => $this->complete('directory.join.seo.title', 'directory.join.seo.description'),
            StaticPage::Shop => $this->complete('shop.home.seo_title', 'shop.home.seo_description'),
            StaticPage::DirectoryJoinDone, StaticPage::ShopCart, StaticPage::ShopCheckout => $this->templated($page->label(), ''),
        };
    }

    private function complete(string $titleKey, string $descriptionKey): StaticPageDefaults
    {
        return new StaticPageDefaults($this->text($titleKey), true, $this->text($descriptionKey));
    }

    private function templated(string $title, string $description): StaticPageDefaults
    {
        return new StaticPageDefaults($title, false, $description);
    }

    private function text(string $key, string $fallback = ''): string
    {
        $value = $this->translator->get($key);

        return is_string($value) && $value !== $key ? trim($value) : $fallback;
    }
}

<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Contact\Actions\SubmitContactMessage;
use App\Domain\Contact\Enums\ContactTopic;
use App\Domain\Contact\Support\ContactRecipients;
use App\Domain\Contact\Support\FormTimer;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Schema\Enums\WebPageType;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\StaticPages\StaticPageSeo;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\SiteSettings;
use App\Http\Requests\ContactRequest;
use App\Support\Text\PersianDigits;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Response;

/**
 * /contact (design/html/contact.html) — pages/contact.blade.php: contact channels from the contact settings, the
 * emergency note, the contact form and the `contact` FAQ group (FaqServiceProvider composer → `$faq` + FAQPage).
 * JSON-LD: ContactPage, plus one Organization contactPoint per configured mailbox.
 *
 * Form flow (PRG): POST → ContactRequest → SubmitContactMessage → redirect to /contact#contact-form with a flash
 * message. Spam (honeypot, time trap) gets the very same redirect and stores nothing; `throttle:contact` limits
 * posts per IP. The page sends `Cache-Control: no-store`, so the full-page cache never stores it: every render
 * mints a fresh time-trap token (FormTimer) and the visitor's own CSRF token, and flash/error states are never cached.
 */
final class ContactController
{
    public const FLASH = 'contact_status';

    public function __construct(
        private readonly StaticPageSeo $staticSeo,
        private readonly SettingsRepository $settings,
        private readonly FormTimer $timer,
        private readonly ViewFactory $views,
        private readonly Config $config,
    ) {}

    /** SeoManager and SchemaGraph are request-scoped: injected per call, not into the (route-cached) controller. */
    public function show(SeoManager $seo, SchemaGraph $graph): Response
    {
        $settings = $this->settings->all();

        $this->staticSeo->apply($seo, StaticPage::Contact);
        $graph->pageType(WebPageType::ContactPage)->pageName(StaticPage::Contact->label());
        $this->contactPoints($graph, $settings);

        $content = $this->views->make('pages.contact', [
            'contact' => $settings->contact,
            'supportEmail' => ContactRecipients::valid($settings->contact->supportEmail),
            'partnershipEmail' => ContactRecipients::valid($settings->contact->partnershipEmail),
            'phoneHref' => self::telephone($settings->contact->phone),
            'emergencyNumber' => $settings->general->emergencyNumber,
            'topics' => ContactTopic::options(),
            'formToken' => $this->timer->issue(),
        ])->render();

        return new Response($content, 200, ['Content-Type' => 'text/html; charset=UTF-8', 'Cache-Control' => 'no-store, private']);
    }

    public function store(ContactRequest $request, SubmitContactMessage $submit): RedirectResponse
    {
        if (! $request->isSpam()) {
            $submit->handle($request->toData());
        }

        return redirect()->to(route('contact').'#contact-form')->with(self::FLASH, 'sent');
    }

    /**
     * The contact page lists the organization's mailboxes as contactPoints (only real addresses — empty settings and
     * «[…]» placeholders are skipped; with none configured the automatic Organization node stays as it is).
     */
    private function contactPoints(SchemaGraph $graph, SiteSettings $settings): void
    {
        $points = [];
        foreach ([
            'customer support' => [$settings->contact->supportEmail, $settings->contact->phone],
            'partnerships' => [$settings->contact->partnershipEmail, null],
            'data protection' => [$settings->legal->dataProtectionEmail, null],
        ] as $type => [$email, $phone]) {
            $email = ContactRecipients::valid($email);
            $phone = self::telephone($phone);
            if ($email === null && $phone === null) {
                continue;
            }
            $points[] = Node::clean([
                '@type' => 'ContactPoint',
                'contactType' => $type,
                'email' => $email,
                'telephone' => $phone,
                'areaServed' => 'IR',
                'availableLanguage' => ['fa'],
            ]);
        }

        if ($points !== []) {
            $graph->add([
                '@type' => 'Organization',
                '@id' => SchemaIds::organization(SchemaIds::root((string) $this->config->get('app.url'))),
                'contactPoint' => $points,
            ]);
        }
    }

    /** A dialable number (`+98…` / digits, 5–15 of them) from the phone setting, or null for empty / placeholder text. */
    private static function telephone(?string $phone): ?string
    {
        $digits = preg_replace('/[^\d+]/', '', PersianDigits::toLatin($phone)) ?? '';

        return preg_match('/^\+?\d{5,15}$/', $digits) === 1 ? $digits : null;
    }
}

<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

/**
 * The only settings groups the public layout (header/footer) needs; shared as `$siteSettings`.
 */
final readonly class LayoutSettings
{
    public function __construct(
        public GeneralSettings $general,
        public ContactSettings $contact,
        public SocialSettings $social,
        public AppLinksSettings $appLinks,
        public LegalSettings $legal,
    ) {}

    public static function from(SiteSettings $settings): self
    {
        return new self($settings->general, $settings->contact, $settings->social, $settings->appLinks, $settings->legal);
    }
}

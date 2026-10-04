<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Nodes;

use App\Domain\Seo\Data\SeoImage;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Settings\Data\SiteSettings;

/**
 * The publisher: name, logo, sameAs (social + organization settings) and contactPoint, all from settings.
 */
final class OrganizationNode
{
    /**
     * @return array<string, mixed>
     */
    public static function make(SiteSettings $settings, string $siteUrl, ?SeoImage $logo = null): array
    {
        $organization = $settings->organization;
        $contact = $organization->contactPoint;
        $telephone = self::real($contact->telephone ?? $settings->contact->phone);
        $email = self::real($contact->email ?? $settings->contact->supportEmail);
        $email = $email !== null && filter_var($email, FILTER_VALIDATE_EMAIL) !== false ? $email : null;

        return Node::clean([
            '@type' => 'Organization',
            '@id' => SchemaIds::organization($siteUrl),
            'name' => $settings->general->siteName,
            'alternateName' => $settings->general->alternateName,
            'legalName' => self::real($organization->legalName),
            'url' => SchemaIds::root($siteUrl),
            'logo' => $logo !== null ? Node::image($logo, SchemaIds::logo($siteUrl)) : null,
            'image' => $logo !== null ? Node::ref(SchemaIds::logo($siteUrl)) : null,
            'foundingDate' => $organization->foundingDate,
            'email' => $email,
            'sameAs' => Node::strings([...array_values($settings->social->filled()), ...$organization->sameAs]),
            'contactPoint' => $telephone !== null || $email !== null ? [
                '@type' => 'ContactPoint',
                'contactType' => $contact->contactType,
                'telephone' => $telephone,
                'email' => $email,
                'areaServed' => 'IR',
                'availableLanguage' => ['fa'],
            ] : null,
        ]);
    }

    /** Seeded design placeholders like «[شماره تماس]» are not facts and never go into structured data. */
    private static function real(?string $value): ?string
    {
        $value = $value !== null ? trim($value) : null;

        return $value === null || $value === '' || preg_match('/^\[.*\]$/u', $value) === 1 ? null : $value;
    }
}

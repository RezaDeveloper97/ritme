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
        $telephone = $contact->telephone ?? $settings->contact->phone;
        $email = $contact->email ?? $settings->contact->supportEmail;

        return Node::clean([
            '@type' => 'Organization',
            '@id' => SchemaIds::organization($siteUrl),
            'name' => $settings->general->siteName,
            'alternateName' => $settings->general->alternateName,
            'legalName' => $organization->legalName,
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
}

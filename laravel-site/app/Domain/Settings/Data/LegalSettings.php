<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;

/**
 * Trust seal (enamad) and data-protection contact. The enamad badge is either a plain code or an admin-pasted
 * HTML snippet; the renderer must keep only `enamadAllowedTags` from the snippet.
 */
final readonly class LegalSettings implements SettingsGroupData
{
    use CoercesSettingValues;

    /**
     * @param  list<string>  $enamadAllowedTags
     */
    public function __construct(
        public ?string $enamadCode,
        public ?string $enamadHtml,
        public array $enamadAllowedTags,
        public ?string $dataProtectionEmail,
    ) {}

    public static function group(): SettingGroup
    {
        return SettingGroup::Legal;
    }

    public static function fromArray(array $values): static
    {
        return new self(
            enamadCode: self::nullableString($values, 'enamad_code'),
            enamadHtml: self::nullableString($values, 'enamad_html'),
            enamadAllowedTags: array_values(array_filter(
                array_map(strtolower(...), self::stringList($values, 'enamad_allowed_tags', ['a', 'img'])),
                static fn (string $tag): bool => preg_match('/^[a-z][a-z0-9]*$/', $tag) === 1,
            )),
            dataProtectionEmail: self::nullableString($values, 'data_protection_email'),
        );
    }

    public function toArray(): array
    {
        return [
            'enamad_code' => $this->enamadCode,
            'enamad_html' => $this->enamadHtml,
            'enamad_allowed_tags' => $this->enamadAllowedTags,
            'data_protection_email' => $this->dataProtectionEmail,
        ];
    }
}

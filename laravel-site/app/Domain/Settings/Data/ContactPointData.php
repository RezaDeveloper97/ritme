<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

/**
 * schema.org ContactPoint for the Organization node.
 */
final readonly class ContactPointData
{
    use CoercesSettingValues;

    public function __construct(
        public string $contactType,
        public ?string $telephone,
        public ?string $email,
    ) {}

    /**
     * @param  array<string, mixed>  $values
     */
    public static function fromArray(array $values): self
    {
        return new self(
            contactType: self::string($values, 'contact_type', 'customer support'),
            telephone: self::nullableString($values, 'telephone'),
            email: self::nullableString($values, 'email'),
        );
    }

    /**
     * @return array{contact_type: string, telephone: ?string, email: ?string}
     */
    public function toArray(): array
    {
        return ['contact_type' => $this->contactType, 'telephone' => $this->telephone, 'email' => $this->email];
    }
}

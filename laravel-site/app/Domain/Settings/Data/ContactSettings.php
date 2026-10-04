<?php

declare(strict_types=1);

namespace App\Domain\Settings\Data;

use App\Domain\Settings\Enums\SettingGroup;

final readonly class ContactSettings implements SettingsGroupData
{
    use CoercesSettingValues;

    public function __construct(
        public ?string $supportEmail,
        public ?string $partnershipEmail,
        public ?string $phone,
        public ?string $workingHours,
        public ?string $responseTime,
        public ?string $address,
        public ?string $addressNote,
    ) {}

    public static function group(): SettingGroup
    {
        return SettingGroup::Contact;
    }

    public static function fromArray(array $values): static
    {
        return new self(
            supportEmail: self::nullableString($values, 'support_email'),
            partnershipEmail: self::nullableString($values, 'partnership_email'),
            phone: self::nullableString($values, 'phone'),
            workingHours: self::nullableString($values, 'working_hours'),
            responseTime: self::nullableString($values, 'response_time'),
            address: self::nullableString($values, 'address'),
            addressNote: self::nullableString($values, 'address_note'),
        );
    }

    public function toArray(): array
    {
        return [
            'support_email' => $this->supportEmail,
            'partnership_email' => $this->partnershipEmail,
            'phone' => $this->phone,
            'working_hours' => $this->workingHours,
            'response_time' => $this->responseTime,
            'address' => $this->address,
            'address_note' => $this->addressNote,
        ];
    }
}

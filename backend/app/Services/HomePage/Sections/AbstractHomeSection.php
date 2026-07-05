<?php

namespace App\Services\HomePage\Sections;

use App\Services\HomePage\Contracts\HomeSectionInterface;
use App\Services\HomePage\HomeContext;

/**
 * Convenience base for sections: shared helpers + a sensible default
 * supports() (visible in all modes). Concrete sections override what they need.
 */
abstract class AbstractHomeSection implements HomeSectionInterface
{
    public function supports(HomeContext $context): bool
    {
        return true;
    }

    /**
     * Resolve a bilingual ["fa"=>..,"en"=>..] value, gracefully falling back.
     */
    protected function pick(mixed $value, string $locale): mixed
    {
        if (!is_array($value)) {
            return $value;
        }

        return $value[$locale] ?? $value['fa'] ?? $value['en'] ?? null;
    }

    /**
     * Build a standard section action descriptor.
     *
     * @return array{key: string, label: string}
     */
    protected function action(string $key, string $label): array
    {
        return ['key' => $key, 'label' => $label];
    }
}

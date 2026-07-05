<?php

namespace App\Services\MessageSystem\Contracts;

use App\Services\MessageSystem\Core\MessageContext;
use App\Services\MessageSystem\Core\MessageResult;

interface MessageEngineInterface
{
    /**
     * Get the message mode this engine handles
     */
    public function getMode(): string;

    /**
     * Check if this engine can handle the given context
     */
    public function canHandle(MessageContext $context): bool;

    /**
     * Generate messages for the given context
     */
    public function generateMessages(MessageContext $context): MessageResult;

    /**
     * Get base message for Layer 1
     */
    public function getBaseMessage(MessageContext $context): array;

    /**
     * Get override message for Layer 2 (based on symptoms/conditions)
     */
    public function getOverrideMessage(MessageContext $context): ?array;

    /**
     * Get context-specific enums
     */
    public function getEnums(string $locale = 'en'): array;
}

<?php

namespace App\Services\MessageSystem\Core;

use App\Services\MessageSystem\Enums\MessageMode;

/**
 * Data Transfer Object for message generation result
 * Contains all layers of messages and supplementary content
 */
class MessageResult
{
    public function __construct(
        // Meta Information
        public readonly MessageMode $mode,
        public readonly string $date,
        public readonly string $userGoal,
        public readonly string $subscriptionType,

        // Context Info (varies by mode)
        public readonly array $contextInfo,

        // Layer 1 & 2: Base + Override Messages
        public readonly array $primaryMessage,

        // Layer 3: Correlations
        public readonly array $correlations,

        // Layer 4: Patterns (Premium only)
        public readonly array $patterns,

        // Supplementary Modules
        public readonly array $nutrition,
        public readonly array $sleep,
        public readonly array $exercise,

        // Additional Tips
        public readonly array $tips = [],
    ) {}

    /**
     * Convert to API response array
     */
    public function toArray(): array
    {
        return [
            'mode' => $this->mode->value,
            'date' => $this->date,
            'user_goal' => $this->userGoal,
            'subscription_type' => $this->subscriptionType,
            'context_info' => $this->contextInfo,
            'primary_message' => $this->primaryMessage,
            'correlations' => $this->correlations,
            'patterns' => $this->patterns,
            'supplements' => [
                'nutrition' => $this->nutrition,
                'sleep' => $this->sleep,
                'exercise' => $this->exercise,
            ],
            'tips' => $this->tips,
        ];
    }

    /**
     * Create an empty/error result
     */
    public static function empty(MessageMode $mode, string $date, string $message): self
    {
        return new self(
            mode: $mode,
            date: $date,
            userGoal: 'non_ttc',
            subscriptionType: 'free',
            contextInfo: ['error' => $message],
            primaryMessage: [],
            correlations: [],
            patterns: [],
            nutrition: [],
            sleep: [],
            exercise: [],
        );
    }
}

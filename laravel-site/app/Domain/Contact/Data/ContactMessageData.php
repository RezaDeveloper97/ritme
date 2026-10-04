<?php

declare(strict_types=1);

namespace App\Domain\Contact\Data;

use App\Domain\Contact\Enums\ContactTopic;
use App\Domain\Contact\Support\ReplyChannel;

/**
 * A validated contact form submission (built by the delivery layer after validation).
 */
final readonly class ContactMessageData
{
    public function __construct(
        public ContactTopic $topic,
        public string $name,
        public ReplyChannel $channel,
        public string $message,
    ) {}
}

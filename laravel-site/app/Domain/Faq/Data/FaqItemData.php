<?php

declare(strict_types=1);

namespace App\Domain\Faq\Data;

use App\Domain\Seo\Schema\Data\FaqItem;

/**
 * A published question for views. `answer` is sanitised HTML (safe to print unescaped).
 */
final readonly class FaqItemData
{
    public function __construct(
        public int $id,
        public string $question,
        public string $answer,
    ) {}

    public function toSchema(): FaqItem
    {
        return new FaqItem($this->question, $this->answer);
    }

    /**
     * @return array{id: int, question: string, answer: string}
     */
    public function toArray(): array
    {
        return ['id' => $this->id, 'question' => $this->question, 'answer' => $this->answer];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self((int) $data['id'], (string) $data['question'], (string) $data['answer']);
    }
}

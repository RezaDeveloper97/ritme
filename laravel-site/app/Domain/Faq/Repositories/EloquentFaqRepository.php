<?php

declare(strict_types=1);

namespace App\Domain\Faq\Repositories;

use App\Domain\Faq\Contracts\FaqRepository;
use App\Domain\Faq\Data\FaqGroupData;
use App\Domain\Faq\Data\FaqItemData;
use App\Domain\Faq\Models\FaqGroup;
use App\Domain\Faq\Models\FaqItem;
use Illuminate\Database\Eloquent\Relations\Relation;

final class EloquentFaqRepository implements FaqRepository
{
    public function group(string $slug): ?FaqGroupData
    {
        foreach ($this->all() as $group) {
            if ($group->slug === $slug) {
                return $group;
            }
        }

        return null;
    }

    public function listed(): array
    {
        return array_values(array_filter($this->all(), static fn (FaqGroupData $group): bool => $group->listed && ! $group->isEmpty()));
    }

    /**
     * Every group with its published items (two queries; groups are few).
     *
     * @return list<FaqGroupData>
     */
    public function all(): array
    {
        return array_values(FaqGroup::query()
            ->with(['items' => static function (Relation $query): void {
                $query->where('is_published', true)->orderBy('sort_order')->orderBy('id');
            }])
            ->orderBy('sort_order')
            ->orderBy('id')
            ->get()
            ->map(static fn (FaqGroup $group): FaqGroupData => new FaqGroupData(
                id: $group->id,
                slug: $group->slug,
                title: $group->title,
                listed: $group->is_listed,
                items: array_values($group->items
                    ->map(static fn (FaqItem $item): FaqItemData => new FaqItemData($item->id, $item->question, $item->answer))
                    ->all()),
            ))
            ->all());
    }
}

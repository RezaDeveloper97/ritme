<?php

namespace App\Services\HomePage\Sections;

use App\Enums\TaskCategory;
use App\Models\TaskTemplate;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 6 — "وظایف امروز": the daily task checklist with completion progress.
 * Tasks come from active templates (phase-aware); completion is per user/day.
 */
class TasksSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'tasks';
    }

    public function order(): int
    {
        return 60;
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $templates = TaskTemplate::query()
            ->active()
            ->forPhase($context->phase())
            ->get();

        if ($templates->isEmpty()) {
            return null;
        }

        $completedIds = $context->user->taskCompletions()
            ->whereDate('completion_date', $context->date)
            ->whereIn('task_template_id', $templates->pluck('id'))
            ->pluck('task_template_id')
            ->all();
        $completedIds = array_flip($completedIds);

        $items = [];
        $completed = 0;
        foreach ($templates as $template) {
            $isDone = isset($completedIds[$template->id]);
            if ($isDone) {
                $completed++;
            }

            $category = TaskCategory::tryFrom($template->category);

            $items[] = [
                'id' => $template->id,
                'key' => $template->key,
                'title' => $template->localized('title', $context->locale),
                'description' => $template->localized('description', $context->locale),
                'category' => $template->category,
                'category_label' => $category?->label($context->locale),
                'icon' => $template->icon ?? $category?->icon() ?? 'clipboard-check',
                'is_completed' => $isDone,
            ];
        }

        $total = count($items);

        return new HomeSection(
            key: $this->key(),
            type: 'tasks',
            title: $context->t('وظایف امروز', "Today's tasks"),
            data: [
                'items' => $items,
                'progress' => [
                    'completed' => $completed,
                    'total' => $total,
                    'percent' => $total > 0 ? (int) round($completed / $total * 100) : 0,
                ],
            ],
            order: $this->order(),
            subtitle: "{$completed}/{$total}",
        );
    }
}

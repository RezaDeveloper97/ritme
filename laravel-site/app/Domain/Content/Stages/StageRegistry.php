<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Content\Enums\StaticPage;
use Illuminate\Contracts\Container\Container;

/**
 * Finds the StageDefinition of a life stage by convention: `App\Domain\Content\Stages\<Studly slug>` (Cycle, Ttc,
 * Pregnancy, Postpartum, Menopause, Teen). A stage without a class has no page yet.
 */
final class StageRegistry
{
    /** @var array<string, StageDefinition|null> */
    private array $resolved = [];

    public function __construct(private readonly Container $container) {}

    public function find(LifeStage $stage): ?StageDefinition
    {
        if (! array_key_exists($stage->value, $this->resolved)) {
            $class = self::classFor($stage);
            $definition = class_exists($class) && is_subclass_of($class, StageDefinition::class) ? $this->container->make($class) : null;
            $this->resolved[$stage->value] = $definition instanceof StageDefinition ? $definition : null;
        }

        return $this->resolved[$stage->value];
    }

    public function forPage(?StaticPage $page): ?StageDefinition
    {
        if ($page === null || ! str_starts_with($page->value, 'stage.')) {
            return null;
        }

        $stage = LifeStage::tryFrom(substr($page->value, strlen('stage.')));

        return $stage === null ? null : $this->find($stage);
    }

    /**
     * @return list<StageDefinition>
     */
    public function all(): array
    {
        return array_values(array_filter(array_map($this->find(...), LifeStage::cases())));
    }

    /**
     * @return class-string
     */
    public static function classFor(LifeStage $stage): string
    {
        /** @var class-string */
        return __NAMESPACE__.'\\'.str_replace(' ', '', ucwords(str_replace('-', ' ', $stage->value)));
    }
}

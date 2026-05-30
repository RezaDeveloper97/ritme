<?php

namespace App\Services\HomePage;

use App\Services\HomePage\Contracts\HomeSectionInterface;
use App\Services\HomePage\Sections\AffirmationSection;
use App\Services\HomePage\Sections\ArticlesSection;
use App\Services\HomePage\Sections\ChallengeSection;
use App\Services\HomePage\Sections\CyclePredictionSection;
use App\Services\HomePage\Sections\CycleSummarySection;
use App\Services\HomePage\Sections\DoctorReminderSection;
use App\Services\HomePage\Sections\HeaderSection;
use App\Services\HomePage\Sections\MedicationReminderSection;
use App\Services\HomePage\Sections\MyCyclesSection;
use App\Services\HomePage\Sections\NextPeriodSection;
use App\Services\HomePage\Sections\RecommendationsSection;
use App\Services\HomePage\Sections\SmartTipSection;
use App\Services\HomePage\Sections\StatusChartsSection;
use App\Services\HomePage\Sections\TasksSection;
use App\Services\HomePage\Sections\VitalsSection;
use App\Services\HomePage\Sections\WeeklySummarySection;
use App\Services\HomePage\Sections\WeekCalendarSection;
use Illuminate\Support\Facades\Log;

/**
 * Orchestrates the home page: holds the ordered registry of section builders,
 * and assembles the full page or a single section.
 *
 * Fault isolation: a section that throws is logged and skipped — one broken
 * widget never takes down the whole page.
 */
class HomePageService
{
    /** @var HomeSectionInterface[] */
    private array $sections;

    public function __construct()
    {
        // Registration order is irrelevant; output is sorted by ->order().
        $this->sections = [
            new HeaderSection(),
            new WeekCalendarSection(),
            new NextPeriodSection(),
            new CyclePredictionSection(),
            new RecommendationsSection(),
            new TasksSection(),
            new ChallengeSection(),
            new DoctorReminderSection(),
            new MedicationReminderSection(),
            new SmartTipSection(),
            new AffirmationSection(),
            new WeeklySummarySection(),
            new VitalsSection(),
            new StatusChartsSection(),
            new ArticlesSection(),
            new MyCyclesSection(),
            new CycleSummarySection(),
        ];
    }

    /**
     * Build every supported section, sorted by display order.
     *
     * @return array<int, array>
     */
    public function build(HomeContext $context): array
    {
        $built = [];

        foreach ($this->sections as $section) {
            if (!$section->supports($context)) {
                continue;
            }

            $result = $this->safeBuild($section, $context);

            if ($result !== null) {
                $built[] = $result->toArray();
            }
        }

        usort($built, fn ($a, $b) => $a['order'] <=> $b['order']);

        return $built;
    }

    /**
     * Build a single section by key. Returns null if unknown/unsupported/empty.
     */
    public function buildSection(string $key, HomeContext $context): ?array
    {
        $section = $this->find($key);

        if (!$section || !$section->supports($context)) {
            return null;
        }

        return $this->safeBuild($section, $context)?->toArray();
    }

    public function hasSection(string $key): bool
    {
        return $this->find($key) !== null;
    }

    /**
     * @return array<int, string>
     */
    public function availableKeys(): array
    {
        return array_map(fn (HomeSectionInterface $s) => $s->key(), $this->sections);
    }

    private function find(string $key): ?HomeSectionInterface
    {
        foreach ($this->sections as $section) {
            if ($section->key() === $key) {
                return $section;
            }
        }

        return null;
    }

    private function safeBuild(HomeSectionInterface $section, HomeContext $context): ?HomeSection
    {
        try {
            return $section->build($context);
        } catch (\Throwable $e) {
            Log::error('HomePage section failed to build', [
                'section' => $section->key(),
                'user_id' => $context->user->id,
                'error' => $e->getMessage(),
            ]);

            return null;
        }
    }
}

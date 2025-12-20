<?php

namespace App\Services\MessageSystem\Core;

use App\Enums\SubscriptionType;
use App\Enums\UserGoal;
use App\Models\DailyHealthLog;
use App\Models\User;
use App\Services\HealthDataEngine;
use App\Services\MessageSystem\Contracts\MessageEngineInterface;
use App\Services\MessageSystem\Engines\CycleMessageEngine;
use App\Services\MessageSystem\Engines\PregnancyMessageEngine;
use App\Services\MessageSystem\Enums\MessageMode;
use App\Services\MessageSystem\Layers\CorrelationLayer;
use App\Services\MessageSystem\Layers\PatternLayer;
use App\Services\MessageSystem\Modules\ExerciseModule;
use App\Services\MessageSystem\Modules\NutritionModule;
use App\Services\MessageSystem\Modules\SleepModule;
use App\Services\PregnancyEngine\PregnancyCalculationService;
use Carbon\Carbon;

/**
 * Main entry point for the Message System
 * Coordinates between different engines and layers
 */
class MessageManager
{
    private array $engines = [];

    public function __construct(
        private readonly User $user,
        private readonly string $locale = 'fa',
    ) {
        $this->registerEngines();
    }

    /**
     * Register available message engines
     */
    private function registerEngines(): void
    {
        $this->engines = [
            MessageMode::CYCLE->value => new CycleMessageEngine($this->user, $this->locale),
            MessageMode::PREGNANCY->value => new PregnancyMessageEngine($this->user, $this->locale),
        ];
    }

    /**
     * Detect current mode for user
     */
    public function detectMode(): MessageMode
    {
        // Check if user has active pregnancy mode
        $pregnancyProfile = $this->user->pregnancyProfile;
        if ($pregnancyProfile && $pregnancyProfile->pregnancy_mode) {
            return MessageMode::PREGNANCY;
        }

        // Default to cycle mode
        return MessageMode::CYCLE;
    }

    /**
     * Get the appropriate engine for the current mode
     */
    public function getEngine(?MessageMode $mode = null): ?MessageEngineInterface
    {
        $mode = $mode ?? $this->detectMode();
        return $this->engines[$mode->value] ?? null;
    }

    /**
     * Build context for message generation
     */
    public function buildContext(Carbon $date, ?MessageMode $forceMode = null): MessageContext
    {
        $mode = $forceMode ?? $this->detectMode();
        $profile = $this->user->profile;

        // Get user preferences
        $userGoal = $profile?->user_goal ?? UserGoal::NON_TTC->value;
        $subscriptionType = $profile?->subscription_type ?? SubscriptionType::FREE->value;

        // Get daily health log
        $dailyLog = DailyHealthLog::where('user_id', $this->user->id)
            ->whereDate('log_date', $date)
            ->first();

        // Get recent logs for pattern detection (last 90 days)
        $recentLogs = DailyHealthLog::where('user_id', $this->user->id)
            ->where('log_date', '>=', $date->copy()->subDays(90))
            ->where('log_date', '<=', $date)
            ->orderBy('log_date', 'desc')
            ->get()
            ->toArray();

        // Extract symptoms from daily log
        $symptoms = $this->extractSymptoms($dailyLog);

        // Build mode-specific context
        if ($mode === MessageMode::PREGNANCY) {
            return $this->buildPregnancyContext($date, $mode, $userGoal, $subscriptionType, $dailyLog, $symptoms, $recentLogs);
        }

        return $this->buildCycleContext($date, $mode, $userGoal, $subscriptionType, $dailyLog, $symptoms, $recentLogs);
    }

    /**
     * Build cycle-specific context
     */
    private function buildCycleContext(
        Carbon $date,
        MessageMode $mode,
        string $userGoal,
        string $subscriptionType,
        ?object $dailyLog,
        array $symptoms,
        array $recentLogs
    ): MessageContext {
        $healthEngine = new HealthDataEngine($this->user, $this->locale);
        $cycleData = $healthEngine->calculateForDate($date);

        return new MessageContext(
            user: $this->user,
            locale: $this->locale,
            date: $date,
            mode: $mode,
            userGoal: $userGoal,
            subscriptionType: $subscriptionType,
            cyclePhase: $cycleData['phase'] ?? null,
            cycleSubphase: $cycleData['subphase'] ?? null,
            cycleDay: $cycleData['cycle_day'] ?? null,
            cycleLength: $cycleData['cycle_length_used'] ?? null,
            isFertileWindow: $cycleData['is_fertile_window'] ?? false,
            isPmsWindow: $cycleData['is_pms_window'] ?? false,
            estimatedOvulationDay: $cycleData['estimated_ovulation_day'] ?? null,
            dailyLog: $dailyLog,
            symptoms: $symptoms,
            recentLogs: $recentLogs,
        );
    }

    /**
     * Build pregnancy-specific context
     */
    private function buildPregnancyContext(
        Carbon $date,
        MessageMode $mode,
        string $userGoal,
        string $subscriptionType,
        ?object $dailyLog,
        array $symptoms,
        array $recentLogs
    ): MessageContext {
        $pregnancyService = new PregnancyCalculationService($this->user, $this->locale);
        $status = $pregnancyService->getPregnancyStatus();

        $gestationalAge = $status['gestational_age'] ?? null;

        return new MessageContext(
            user: $this->user,
            locale: $this->locale,
            date: $date,
            mode: $mode,
            userGoal: $userGoal,
            subscriptionType: $subscriptionType,
            pregnancyWeek: $gestationalAge['weeks'] ?? null,
            pregnancyDay: $gestationalAge['days'] ?? null,
            trimester: $status['trimester'] ?? null,
            dueDate: isset($status['estimated_due_date']['date'])
                ? Carbon::parse($status['estimated_due_date']['date'])
                : null,
            dailyLog: $dailyLog,
            symptoms: $symptoms,
            recentLogs: $recentLogs,
        );
    }

    /**
     * Generate messages for a specific date
     */
    public function generateMessages(Carbon $date, ?MessageMode $forceMode = null): MessageResult
    {
        $context = $this->buildContext($date, $forceMode);
        $engine = $this->getEngine($context->mode);

        if (!$engine) {
            return MessageResult::empty(
                $context->mode,
                $date->toDateString(),
                $this->locale === 'fa' ? 'موتور پیام برای این حالت یافت نشد' : 'Message engine not found for this mode'
            );
        }

        // Get base message from engine (Layer 1 & 2)
        $baseMessage = $engine->getBaseMessage($context);
        $overrideMessage = $engine->getOverrideMessage($context);

        // Merge override into base if exists
        $primaryMessage = $overrideMessage
            ? array_merge($baseMessage, $overrideMessage, ['has_override' => true])
            : array_merge($baseMessage, ['has_override' => false]);

        // Layer 3: Correlations
        $correlationLayer = new CorrelationLayer($this->user, $this->locale);
        $correlations = $correlationLayer->analyze($context);

        // Filter premium-only correlations for free users
        if (!$context->isPremium()) {
            $correlations = array_filter($correlations, fn($c) => !($c['is_premium_only'] ?? false));
            $correlations = array_values($correlations);
        }

        // Layer 4: Patterns (Premium only)
        $patterns = [];
        if ($context->isPremium()) {
            $patternLayer = new PatternLayer($this->user, $this->locale);
            $patterns = $patternLayer->analyze($context);
        }

        // Supplementary Modules
        $nutritionModule = new NutritionModule($this->locale);
        $sleepModule = new SleepModule($this->locale);
        $exerciseModule = new ExerciseModule($this->locale);

        return new MessageResult(
            mode: $context->mode,
            date: $date->toDateString(),
            userGoal: $context->userGoal,
            subscriptionType: $context->subscriptionType,
            contextInfo: $this->buildContextInfo($context),
            primaryMessage: $primaryMessage,
            correlations: $correlations,
            patterns: $patterns,
            nutrition: $nutritionModule->getTips($context),
            sleep: $sleepModule->getTips($context),
            exercise: $exerciseModule->getTips($context),
        );
    }

    /**
     * Build context info for response
     */
    private function buildContextInfo(MessageContext $context): array
    {
        if ($context->isCycleMode()) {
            return [
                'phase' => $context->cyclePhase,
                'phase_label' => $context->cyclePhase
                    ? \App\Enums\CyclePhase::from($context->cyclePhase)->label($this->locale)
                    : null,
                'subphase' => $context->cycleSubphase,
                'subphase_label' => $context->cycleSubphase
                    ? \App\Enums\CycleSubphase::from($context->cycleSubphase)->label($this->locale)
                    : null,
                'cycle_day' => $context->cycleDay,
                'cycle_length' => $context->cycleLength,
                'is_fertile_window' => $context->isFertileWindow,
                'is_pms_window' => $context->isPmsWindow,
                'estimated_ovulation_day' => $context->estimatedOvulationDay,
            ];
        }

        if ($context->isPregnancyMode()) {
            return [
                'week' => $context->pregnancyWeek,
                'day' => $context->pregnancyDay,
                'trimester' => $context->trimester,
                'gestational_age' => $context->getGestationalAgeString(),
                'due_date' => $context->dueDate?->toDateString(),
            ];
        }

        return [];
    }

    /**
     * Extract symptoms from daily log
     */
    private function extractSymptoms(?object $dailyLog): array
    {
        if (!$dailyLog) {
            return [];
        }

        $symptoms = [];

        // Pain symptoms
        if ($dailyLog->has_cramps) $symptoms[] = 'cramps';
        if ($dailyLog->has_headache) $symptoms[] = 'headache';
        if ($dailyLog->has_backache) $symptoms[] = 'backache';
        if ($dailyLog->has_breast_tenderness) $symptoms[] = 'breast_tenderness';

        // Mood symptoms
        if ($dailyLog->mood === 'sad' || $dailyLog->mood === 'depressed') $symptoms[] = 'mood_sad';
        if ($dailyLog->mood === 'anxious' || $dailyLog->mood === 'stressed') $symptoms[] = 'mood_anxious';
        if ($dailyLog->mood === 'angry' || $dailyLog->mood === 'irritable') $symptoms[] = 'mood_angry';

        // Flow
        if ($dailyLog->flow_intensity === 'heavy') $symptoms[] = 'heavy_flow';
        if ($dailyLog->flow_intensity === 'light') $symptoms[] = 'light_flow';

        // Energy & Sleep
        if ($dailyLog->energy_level === 'low' || $dailyLog->energy_level === 'very_low') $symptoms[] = 'low_energy';
        if ($dailyLog->sleep_quality === 'poor' || $dailyLog->sleep_quality === 'very_poor') $symptoms[] = 'poor_sleep';

        // Other
        if ($dailyLog->has_bloating) $symptoms[] = 'bloating';
        if ($dailyLog->has_nausea) $symptoms[] = 'nausea';
        if ($dailyLog->has_fatigue) $symptoms[] = 'fatigue';
        if ($dailyLog->has_acne) $symptoms[] = 'acne';

        return $symptoms;
    }

    /**
     * Get all available enums for API
     */
    public function getEnums(): array
    {
        $enums = [
            'modes' => collect(MessageMode::cases())->map(fn($m) => [
                'value' => $m->value,
                'label' => $m->label($this->locale),
            ])->values()->toArray(),
            'user_goals' => collect(UserGoal::cases())->map(fn($g) => [
                'value' => $g->value,
                'label' => $g->label($this->locale),
            ])->values()->toArray(),
            'subscription_types' => collect(SubscriptionType::cases())->map(fn($s) => [
                'value' => $s->value,
                'label' => $s->label($this->locale),
            ])->values()->toArray(),
        ];

        // Add engine-specific enums
        foreach ($this->engines as $mode => $engine) {
            $enums[$mode] = $engine->getEnums($this->locale);
        }

        return $enums;
    }
}

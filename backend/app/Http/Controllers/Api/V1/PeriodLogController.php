<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\CalculationStatus;
use App\Enums\DataQualityFlag;
use App\Enums\DataSource;
use App\Http\Controllers\Concerns\ResolvesLocale;
use App\Http\Controllers\Controller;
use App\Jobs\CalculateCycleDataJob;
use App\Models\CycleHistory;
use Carbon\Carbon;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Validator;

/**
 * Period logging — lets the user mark the start and the end of a menstrual
 * period from the home screen. Starting a period re-anchors the cycle (updates
 * the profile LMP and creates a CycleHistory record); ending it closes that
 * record with an end date and bleeding length. Kept separate from
 * ProfileController so the "log a period" action is a first-class endpoint
 * rather than a side effect of a profile update.
 */
class PeriodLogController extends Controller
{
    use ResolvesLocale;

    /** A period longer than this is unusual — allowed, but flagged and warned (§16). */
    private const LONG_PERIOD_DAYS = 10;

    /** Cycle gaps outside this range read as irregular outliers (§8, §16). */
    private const MIN_PLAUSIBLE_CYCLE_DAYS = 21;

    private const MAX_PLAUSIBLE_CYCLE_DAYS = 45;

    /** A new start closer than this to the previous one gets a stronger warning (§16). */
    private const CLOSE_START_DAYS = 14;

    /** An open period older than this is no longer "ongoing" and won't block a new start (§4). */
    private const HARD_CAP_DAYS = 12;

    /**
     * @OA\Get(
     *     path="/cycle/period/status",
     *     summary="Current period logging state",
     *     description="Returns whether the user currently has an ongoing (open) period, so the client can show Start vs End.",
     *     tags={"Cycle"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Response(
     *         response=200,
     *         description="Period status",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="active", type="boolean", example=true),
     *                 @OA\Property(property="period_start_date", type="string", format="date", nullable=true, example="2024-12-01"),
     *                 @OA\Property(property="period_end_date", type="string", format="date", nullable=true, example=null)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function status(Request $request): JsonResponse
    {
        $user = $request->user();
        $ongoing = $this->ongoingPeriod($user->id);
        $latest = CycleHistory::where('user_id', $user->id)
            ->orderBy('period_start_date', 'desc')
            ->first();

        return response()->json([
            'success' => true,
            'data' => [
                'active' => (bool) $ongoing,
                'period_start_date' => $latest?->period_start_date?->toDateString(),
                'period_end_date' => $latest?->period_end_date?->toDateString(),
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/cycle/period/start",
     *     summary="Mark a period as started",
     *     description="Records the start of a menstrual period (defaults to today), re-anchors the cycle and triggers recalculation.",
     *     tags={"Cycle"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\RequestBody(
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="date", type="string", format="date", nullable=true, example="2024-12-01", description="Period start date; defaults to today")
     *         )
     *     ),
     *
     *     @OA\Response(response=200, description="Period start recorded"),
     *     @OA\Response(response=422, description="Validation error"),
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function start(Request $request): JsonResponse
    {
        $locale = $this->resolveLocale($request);
        app()->setLocale($locale);

        $validator = Validator::make($request->all(), [
            'date' => 'nullable|date|before_or_equal:today',
        ]);

        if ($validator->fails()) {
            return response()->json([
                'success' => false,
                'message' => $validator->errors()->first(),
                'errors' => $validator->errors(),
            ], 422);
        }

        $user = $request->user();
        $startDate = $request->filled('date')
            ? Carbon::parse($request->input('date'))->startOfDay()
            : now()->startOfDay();

        // Reuse the record for this exact day if it already exists (idempotent
        // re-tap / estimate promotion), otherwise create a fresh one. Avoid
        // updateOrCreate on the date column — SQLite's date cast makes it unreliable.
        $period = CycleHistory::where('user_id', $user->id)
            ->whereDate('period_start_date', $startDate->toDateString())
            ->first();

        // Block a new start while a genuinely-ongoing period is still open (spec §3/§16.6):
        // the user must close the previous bleed first rather than stack two open periods.
        if ($blocker = $this->blockingOpenPeriod($user->id, $startDate)) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa'
                    ? 'پایان پریود قبلی هنوز ثبت نشده است. ابتدا پایان آن را مشخص کن.'
                    : 'Your previous period has no end date yet. Please log its end first.',
                'code' => 'previous_period_open',
                'data' => ['open_period_start' => $blocker->period_start_date?->toDateString()],
            ], 422);
        }

        // Reject a start that lands inside another logged period's range (spec §16.2).
        // The same-day record (if any) is the reuse target, not an overlap.
        if ($this->overlapsExistingPeriod($user->id, $startDate, $startDate, $period?->id)) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa'
                    ? 'این تاریخ با یک پریود ثبت‌شدهٔ دیگر همپوشانی دارد. لطفاً یکی از بازه‌ها را ویرایش کن.'
                    : 'This date overlaps another logged period. Please edit one of the ranges.',
                'code' => 'period_overlap',
            ], 422);
        }

        if ($period) {
            // Logging a start promotes an onboarding estimate on this day into a real,
            // user-confirmed period (spec §1) and reopens it so End can close it.
            $period->update([
                'period_end_date' => null,
                'bleeding_length' => null,
                'is_confirmed' => true,
                'is_estimated' => false,
                'source' => DataSource::USER_LOGGED->value,
            ]);
        } else {
            // Cycle length = gap from the previous period start, if any.
            $previous = CycleHistory::where('user_id', $user->id)
                ->whereDate('period_start_date', '<', $startDate->toDateString())
                ->orderBy('period_start_date', 'desc')
                ->first();

            $cycleLength = $previous
                ? Carbon::parse($previous->period_start_date)->diffInDays($startDate)
                : null;

            $period = CycleHistory::create([
                'user_id' => $user->id,
                'period_start_date' => $startDate,
                'cycle_length' => $cycleLength,
                'is_confirmed' => true,
                'source' => DataSource::USER_LOGGED->value,
                'data_quality_flags' => $this->qualityFlags($cycleLength, null),
            ]);
        }

        // Real logged data supersedes the onboarding seed — drop any remaining
        // estimates (other than the record we just wrote) so they can't linger.
        CycleHistory::where('user_id', $user->id)
            ->where('is_estimated', true)
            ->where('id', '!=', $period->id)
            ->delete();

        // Re-anchor the cycle engine on this period start and recalculate.
        $profile = $user->profile;
        if ($profile) {
            $profile->update(['last_period_start' => $startDate->toDateString()]);
            $this->triggerRecalculation($user, $profile, $locale);
        }

        return response()->json([
            'success' => true,
            'message' => __('profile.updated'),
            'data' => [
                'active' => true,
                'period_start_date' => $period->period_start_date?->toDateString(),
                'period_end_date' => $period->period_end_date?->toDateString(),
                'warnings' => $this->periodWarnings($period->cycle_length, $period->bleeding_length, $locale),
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/cycle/period",
     *     summary="Log a complete period range",
     *     description="Creates a logged period from an explicit start (and optional end) date in one call. Unlike start+end, this never depends on which period is currently ongoing, so a client reconciling several ranges at once can't close the wrong one.",
     *     tags={"Cycle"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\RequestBody(
     *         required=true,
     *
     *         @OA\JsonContent(
     *             required={"start_date"},
     *
     *             @OA\Property(property="start_date", type="string", format="date", example="2026-03-21"),
     *             @OA\Property(property="end_date", type="string", format="date", nullable=true, example="2026-03-25", description="Omit/null to leave the period ongoing")
     *         )
     *     ),
     *
     *     @OA\Response(response=200, description="Period created"),
     *     @OA\Response(response=422, description="Validation error / overlap"),
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function store(Request $request): JsonResponse
    {
        $locale = $this->resolveLocale($request);
        app()->setLocale($locale);

        $validator = Validator::make($request->all(), [
            'start_date' => 'required|date|before_or_equal:today',
            'end_date' => 'nullable|date|after_or_equal:start_date|before_or_equal:today',
        ]);

        if ($validator->fails()) {
            return response()->json([
                'success' => false,
                'message' => $validator->errors()->first(),
                'errors' => $validator->errors(),
            ], 422);
        }

        $user = $request->user();
        $startDate = Carbon::parse($request->input('start_date'))->startOfDay();
        $endDate = $request->filled('end_date')
            ? Carbon::parse($request->input('end_date'))->startOfDay()
            : null;

        // Same-day record → reuse it (idempotent re-save, and promotes an
        // onboarding estimate into a real logged period, like start() does).
        $period = CycleHistory::where('user_id', $user->id)
            ->whereDate('period_start_date', $startDate->toDateString())
            ->first();

        if ($this->overlapsExistingPeriod($user->id, $startDate, $endDate ?? $startDate, $period?->id)) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa'
                    ? 'این بازه با یک پریود ثبت‌شدهٔ دیگر همپوشانی دارد. لطفاً یکی از بازه‌ها را ویرایش کن.'
                    : 'This range overlaps another logged period. Please edit one of the ranges.',
                'code' => 'period_overlap',
            ], 422);
        }

        $attributes = [
            'period_start_date' => $startDate,
            'period_end_date' => $endDate?->toDateString(),
            'bleeding_length' => $endDate ? $startDate->diffInDays($endDate) + 1 : null,
            'is_confirmed' => true,
            'is_estimated' => false,
            'source' => DataSource::USER_LOGGED->value,
        ];

        if ($period) {
            $period->update($attributes);
        } else {
            $period = CycleHistory::create($attributes + ['user_id' => $user->id]);
        }

        // Real logged data supersedes the onboarding seed (same rule as start()).
        CycleHistory::where('user_id', $user->id)
            ->where('is_estimated', true)
            ->where('id', '!=', $period->id)
            ->delete();

        $this->recomputeCycleLengths($user->id);
        $this->reanchor($user, $locale);

        $period->refresh();
        $period->update([
            'data_quality_flags' => $this->qualityFlags($period->cycle_length, $period->bleeding_length),
        ]);

        return response()->json([
            'success' => true,
            'message' => __('profile.updated'),
            'data' => [
                'id' => $period->id,
                'active' => $period->period_end_date === null,
                'period_start_date' => $period->period_start_date?->toDateString(),
                'period_end_date' => $period->period_end_date?->toDateString(),
                'warnings' => $this->periodWarnings($period->cycle_length, $period->bleeding_length, $locale),
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/cycle/period/end",
     *     summary="Mark the ongoing period as ended",
     *     description="Closes the current ongoing period with an end date (defaults to today) and records the bleeding length.",
     *     tags={"Cycle"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\RequestBody(
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="date", type="string", format="date", nullable=true, example="2024-12-05", description="Period end date; defaults to today")
     *         )
     *     ),
     *
     *     @OA\Response(response=200, description="Period end recorded"),
     *     @OA\Response(response=422, description="Validation error / no ongoing period"),
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function end(Request $request): JsonResponse
    {
        $locale = $this->resolveLocale($request);
        app()->setLocale($locale);

        $validator = Validator::make($request->all(), [
            'date' => 'nullable|date|before_or_equal:today',
        ]);

        if ($validator->fails()) {
            return response()->json([
                'success' => false,
                'message' => $validator->errors()->first(),
                'errors' => $validator->errors(),
            ], 422);
        }

        $user = $request->user();
        $period = $this->ongoingPeriod($user->id);

        if (! $period) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa'
                    ? 'پریود فعالی برای پایان دادن وجود ندارد.'
                    : 'There is no ongoing period to end.',
            ], 422);
        }

        $endDate = $request->filled('date')
            ? Carbon::parse($request->input('date'))->startOfDay()
            : now()->startOfDay();

        $startDate = Carbon::parse($period->period_start_date)->startOfDay();

        if ($endDate->lt($startDate)) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa'
                    ? 'تاریخ پایان پریود نمی‌تواند قبل از تاریخ شروع باشد.'
                    : 'The period end date cannot be before its start date.',
            ], 422);
        }

        $bleedingLength = $startDate->diffInDays($endDate) + 1;
        $period->update([
            'period_end_date' => $endDate->toDateString(),
            'bleeding_length' => $bleedingLength,
            'data_quality_flags' => $this->qualityFlags($period->cycle_length, $bleedingLength),
        ]);

        return response()->json([
            'success' => true,
            'message' => __('profile.updated'),
            'data' => [
                'active' => false,
                'period_start_date' => $period->period_start_date?->toDateString(),
                'period_end_date' => $period->period_end_date?->toDateString(),
                'warnings' => $this->periodWarnings($period->cycle_length, $bleedingLength, $locale),
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/cycle/period/history",
     *     summary="List logged periods",
     *     description="All periods the user has logged (newest first), so the client can render and edit real logged ranges as opposed to predictions.",
     *     tags={"Cycle"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Response(
     *         response=200,
     *         description="Logged periods",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="periods", type="array",
     *
     *                     @OA\Items(type="object",
     *
     *                         @OA\Property(property="id", type="integer", example=12),
     *                         @OA\Property(property="period_start_date", type="string", format="date", example="2024-12-01"),
     *                         @OA\Property(property="period_end_date", type="string", format="date", nullable=true, example="2024-12-05")
     *                     )
     *                 )
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function history(Request $request): JsonResponse
    {
        // Estimates (the onboarding seed) are not user-managed periods — they render as
        // predictions and are excluded here so the client only lists real logged ranges.
        $periods = CycleHistory::where('user_id', $request->user()->id)
            ->where('is_estimated', false)
            ->orderBy('period_start_date', 'desc')
            ->get()
            ->map(fn (CycleHistory $p) => [
                'id' => $p->id,
                'period_start_date' => $p->period_start_date?->toDateString(),
                'period_end_date' => $p->period_end_date?->toDateString(),
                'source' => $p->source,
            ]);

        return response()->json([
            'success' => true,
            'data' => ['periods' => $periods],
        ]);
    }

    /**
     * @OA\Put(
     *     path="/cycle/period/{period}",
     *     summary="Edit a logged period",
     *     description="Moves a logged period's start and/or end date, then re-anchors the cycle and recalculates predictions.",
     *     tags={"Cycle"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(name="period", in="path", required=true, @OA\Schema(type="integer")),
     *
     *     @OA\RequestBody(
     *         required=true,
     *
     *         @OA\JsonContent(
     *             required={"start_date"},
     *
     *             @OA\Property(property="start_date", type="string", format="date", example="2024-12-02"),
     *             @OA\Property(property="end_date", type="string", format="date", nullable=true, example="2024-12-06", description="Omit/null to leave the period ongoing")
     *         )
     *     ),
     *
     *     @OA\Response(response=200, description="Period updated"),
     *     @OA\Response(response=404, description="Period not found"),
     *     @OA\Response(response=422, description="Validation error"),
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function update(Request $request, int $period): JsonResponse
    {
        $locale = $this->resolveLocale($request);
        app()->setLocale($locale);

        $validator = Validator::make($request->all(), [
            'start_date' => 'required|date|before_or_equal:today',
            'end_date' => 'nullable|date|after_or_equal:start_date|before_or_equal:today',
        ]);

        if ($validator->fails()) {
            return response()->json([
                'success' => false,
                'message' => $validator->errors()->first(),
                'errors' => $validator->errors(),
            ], 422);
        }

        $user = $request->user();
        $record = CycleHistory::where('user_id', $user->id)->find($period);

        if (! $record) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'پریود مورد نظر پیدا نشد.' : 'Period not found.',
            ], 404);
        }

        $startDate = Carbon::parse($request->input('start_date'))->startOfDay();
        $endDate = $request->filled('end_date')
            ? Carbon::parse($request->input('end_date'))->startOfDay()
            : null;

        // The edited range must not collide with another logged period (spec §16.2).
        if ($this->overlapsExistingPeriod($user->id, $startDate, $endDate ?? $startDate, $record->id)) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa'
                    ? 'این بازه با یک پریود ثبت‌شدهٔ دیگر همپوشانی دارد. لطفاً یکی از بازه‌ها را ویرایش کن.'
                    : 'This range overlaps another logged period. Please edit one of the ranges.',
                'code' => 'period_overlap',
            ], 422);
        }

        $record->update([
            'period_start_date' => $startDate,
            'period_end_date' => $endDate?->toDateString(),
            'bleeding_length' => $endDate ? $startDate->diffInDays($endDate) + 1 : null,
        ]);

        $this->recomputeCycleLengths($user->id);
        $this->reanchor($user, $locale);

        // Recompute quality flags against the re-anchored cycle length and edited duration.
        $record->refresh();
        $record->update([
            'data_quality_flags' => $this->qualityFlags($record->cycle_length, $record->bleeding_length),
        ]);

        return response()->json([
            'success' => true,
            'message' => __('profile.updated'),
            'data' => [
                'id' => $record->id,
                'period_start_date' => $record->period_start_date?->toDateString(),
                'period_end_date' => $record->period_end_date?->toDateString(),
                'warnings' => $this->periodWarnings($record->cycle_length, $record->bleeding_length, $locale),
            ],
        ]);
    }

    /**
     * @OA\Delete(
     *     path="/cycle/period/{period}",
     *     summary="Delete a logged period",
     *     description="Removes a logged period, re-anchors the cycle on the most recent remaining period and recalculates predictions.",
     *     tags={"Cycle"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(name="period", in="path", required=true, @OA\Schema(type="integer")),
     *
     *     @OA\Response(response=200, description="Period deleted"),
     *     @OA\Response(response=404, description="Period not found"),
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function destroy(Request $request, int $period): JsonResponse
    {
        $locale = $this->resolveLocale($request);
        app()->setLocale($locale);

        $user = $request->user();
        $record = CycleHistory::where('user_id', $user->id)->find($period);

        if (! $record) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'پریود مورد نظر پیدا نشد.' : 'Period not found.',
            ], 404);
        }

        $record->delete();

        $this->recomputeCycleLengths($user->id);
        $this->reanchor($user, $locale);

        return response()->json([
            'success' => true,
            'message' => __('profile.updated'),
            'data' => ['deleted' => true],
        ]);
    }

    /**
     * Recompute each record's cycle_length (gap to the previous period start)
     * after an edit or delete shuffled the timeline, so variability and the
     * effective cycle length stay truthful.
     */
    private function recomputeCycleLengths(int $userId): void
    {
        $records = CycleHistory::where('user_id', $userId)
            ->orderBy('period_start_date')
            ->get();

        $previousStart = null;
        foreach ($records as $record) {
            $start = Carbon::parse($record->period_start_date)->startOfDay();
            $length = $previousStart?->diffInDays($start);
            if ($record->cycle_length !== $length) {
                $record->update(['cycle_length' => $length]);
            }
            $previousStart = $start;
        }
    }

    /**
     * Point the engine's anchor (profile LMP) at the most recent remaining
     * logged period — or clear it when none are left — then recalculate.
     */
    private function reanchor($user, string $locale): void
    {
        $profile = $user->profile;
        if (! $profile) {
            return;
        }

        $latest = CycleHistory::where('user_id', $user->id)
            ->orderBy('period_start_date', 'desc')
            ->first();

        $profile->update([
            'last_period_start' => $latest?->period_start_date?->toDateString(),
        ]);

        $this->triggerRecalculation($user, $profile, $locale);
    }

    /**
     * The user's currently-ongoing period: the most recent record that still has no
     * end date. No age window — a long-open period stays endable so the End action
     * agrees with the daily card's "log period end" prompt (spec §4).
     */
    private function ongoingPeriod(int $userId): ?CycleHistory
    {
        return CycleHistory::where('user_id', $userId)
            ->whereNull('period_end_date')
            ->orderBy('period_start_date', 'desc')
            ->first();
    }

    /**
     * A genuinely-ongoing period that should block a new Start on a different day
     * (spec §3/§16.6): a real, open (null-end) record whose start is recent enough
     * (within the hard cap of today) that the user could still be bleeding. Old
     * start-only records — the correction/backfill flow — are deliberately NOT
     * blocking, and estimates (which carry an end) never are.
     *
     * Only an open period that started BEFORE the attempted start blocks it: that
     * is the genuine "stacking a newer bleed on an unclosed older one" case. A new
     * start that lands *earlier* than the open period is a historical backfill —
     * the open period is still the most recent ongoing one, so allowing it doesn't
     * create two competing ongoing periods (previously this wrongly 422'd backfills).
     */
    private function blockingOpenPeriod(int $userId, Carbon $startDate): ?CycleHistory
    {
        return CycleHistory::where('user_id', $userId)
            ->whereNull('period_end_date')
            ->where('is_estimated', false)
            ->whereDate('period_start_date', '<', $startDate->toDateString())
            ->whereDate('period_start_date', '>=', now()->subDays(self::HARD_CAP_DAYS)->toDateString())
            ->orderBy('period_start_date', 'desc')
            ->first();
    }

    /**
     * Whether [start, end] intersects another logged period's actual range (spec §16.2).
     * Open periods and estimates count only as their single start day — their assumed
     * bleed length is not treated as occupied, so the correction flow (starts logged a
     * few days apart) and the onboarding seed don't produce phantom overlaps.
     */
    private function overlapsExistingPeriod(int $userId, Carbon $start, Carbon $end, ?int $excludeId): bool
    {
        $others = CycleHistory::where('user_id', $userId)
            ->where('is_estimated', false)
            ->when($excludeId !== null, fn ($q) => $q->where('id', '!=', $excludeId))
            ->get();

        foreach ($others as $other) {
            $otherStart = Carbon::parse($other->period_start_date)->startOfDay();
            $otherEnd = $other->period_end_date
                ? Carbon::parse($other->period_end_date)->startOfDay()
                : $otherStart;

            // Standard interval overlap: startA <= endB AND startB <= endA.
            if ($start->lte($otherEnd) && $otherStart->lte($end)) {
                return true;
            }
        }

        return false;
    }

    /**
     * Non-fatal data-quality flags implied by a period's own bleeding length and the
     * gap to the previous period (§8, §16). Stored on the record so predictions can
     * hold outliers out of the medians while keeping them visible in history.
     *
     * @return array<string>
     */
    private function qualityFlags(?int $cycleLength, ?int $bleedingLength): array
    {
        $flags = [];

        if ($bleedingLength !== null && $bleedingLength > self::LONG_PERIOD_DAYS) {
            $flags[] = DataQualityFlag::LONG_PERIOD->value;
        }
        if ($bleedingLength === 1) {
            $flags[] = DataQualityFlag::SHORT_PERIOD->value;
        }
        if ($cycleLength !== null && ($cycleLength < self::MIN_PLAUSIBLE_CYCLE_DAYS || $cycleLength > self::MAX_PLAUSIBLE_CYCLE_DAYS)) {
            $flags[] = DataQualityFlag::IRREGULAR_CYCLE->value;
        }

        return $flags;
    }

    /**
     * Non-blocking warnings for unusual but allowed data (§16): logging is never
     * blocked — the user is simply informed and the record is flagged.
     *
     * @return array<array{type: string, message: string}>
     */
    private function periodWarnings(?int $cycleLength, ?int $bleedingLength, string $locale): array
    {
        $warnings = [];

        if ($bleedingLength !== null && $bleedingLength > self::LONG_PERIOD_DAYS) {
            $warnings[] = [
                'type' => DataQualityFlag::LONG_PERIOD->value,
                'message' => $locale === 'fa'
                    ? 'این بازه طولانی‌تر از معمول است. اگر مطمئنی، می‌توانی آن را ثبت کنی.'
                    : 'This range is longer than usual. You can still log it if you are sure.',
            ];
        }

        if ($bleedingLength === 1) {
            $warnings[] = [
                'type' => DataQualityFlag::SHORT_PERIOD->value,
                'message' => $locale === 'fa'
                    ? 'آیا این خون‌ریزی واقعاً پریود بود یا لکه‌بینی؟ می‌توانی همچنان ثبتش کنی.'
                    : 'Was this really a period or spotting? You can still log it.',
            ];
        }

        if ($cycleLength !== null && $cycleLength > 0 && $cycleLength < self::CLOSE_START_DAYS) {
            $warnings[] = [
                'type' => 'close_to_previous',
                'message' => $locale === 'fa'
                    ? 'این تاریخ خیلی نزدیک به پریود قبلی است. مطمئنی این یک پریود جدید است و ادامهٔ خون‌ریزی قبلی نیست؟'
                    : 'This is very close to your previous period. Are you sure it is a new period and not continued bleeding?',
            ];
        } elseif ($cycleLength !== null && ($cycleLength < self::MIN_PLAUSIBLE_CYCLE_DAYS || $cycleLength > self::MAX_PLAUSIBLE_CYCLE_DAYS)) {
            $warnings[] = [
                'type' => DataQualityFlag::IRREGULAR_CYCLE->value,
                'message' => $locale === 'fa'
                    ? 'این فاصله با الگوی معمول چرخه‌ها متفاوت است و ممکن است روی دقت پیش‌بینی اثر بگذارد.'
                    : 'This gap differs from your usual cycle pattern and may affect prediction accuracy.',
            ];
        }

        return $warnings;
    }

    /**
     * Kick off background recalculation of cycle data after the LMP moved.
     * Mirrors ProfileController's trigger so the engine stays in sync.
     */
    private function triggerRecalculation($user, $profile, string $locale): void
    {
        if ($profile->calculation_status === CalculationStatus::PROCESSING->value) {
            return;
        }

        $newVersion = ($profile->calculation_version ?? 0) + 1;

        $profile->update([
            'calculation_status' => CalculationStatus::PROCESSING->value,
        ]);

        CalculateCycleDataJob::dispatch($user->id, $newVersion, $locale);
    }
}

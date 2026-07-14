<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\CalculationStatus;
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

    /**
     * How many days after a period start we still consider it "ongoing" (i.e.
     * the End-period action applies to it). Guards against an old un-closed
     * record staying active forever.
     */
    private const ONGOING_WINDOW_DAYS = 15;

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
        // re-tap), otherwise create a fresh one. Avoid updateOrCreate on the
        // date column — SQLite's date cast makes its match unreliable.
        $period = CycleHistory::where('user_id', $user->id)
            ->whereDate('period_start_date', $startDate->toDateString())
            ->first();

        if ($period) {
            // Starting a period means it's ongoing again — clear any end that a
            // previous "end period" wrote for this same day, so the record is a
            // genuinely open period the End action can later close.
            if ($period->period_end_date !== null || $period->bleeding_length !== null) {
                $period->update([
                    'period_end_date' => null,
                    'bleeding_length' => null,
                ]);
            }
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
            ]);
        }

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

        $period->update([
            'period_end_date' => $endDate->toDateString(),
            'bleeding_length' => $startDate->diffInDays($endDate) + 1,
        ]);

        return response()->json([
            'success' => true,
            'message' => __('profile.updated'),
            'data' => [
                'active' => false,
                'period_start_date' => $period->period_start_date?->toDateString(),
                'period_end_date' => $period->period_end_date?->toDateString(),
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
        $periods = CycleHistory::where('user_id', $request->user()->id)
            ->orderBy('period_start_date', 'desc')
            ->get()
            ->map(fn (CycleHistory $p) => [
                'id' => $p->id,
                'period_start_date' => $p->period_start_date?->toDateString(),
                'period_end_date' => $p->period_end_date?->toDateString(),
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

        $record->update([
            'period_start_date' => $startDate,
            'period_end_date' => $endDate?->toDateString(),
            'bleeding_length' => $endDate ? $startDate->diffInDays($endDate) + 1 : null,
        ]);

        $this->recomputeCycleLengths($user->id);
        $this->reanchor($user, $locale);

        return response()->json([
            'success' => true,
            'message' => __('profile.updated'),
            'data' => [
                'id' => $record->id,
                'period_start_date' => $record->period_start_date?->toDateString(),
                'period_end_date' => $record->period_end_date?->toDateString(),
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
     * The user's currently-ongoing period: the most recent record that has no
     * end date yet and started within the ongoing window.
     */
    private function ongoingPeriod(int $userId): ?CycleHistory
    {
        return CycleHistory::where('user_id', $userId)
            ->whereNull('period_end_date')
            ->whereDate('period_start_date', '>=', now()->subDays(self::ONGOING_WINDOW_DAYS)->toDateString())
            ->orderBy('period_start_date', 'desc')
            ->first();
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

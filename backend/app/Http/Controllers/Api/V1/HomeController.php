<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Concerns\ResolvesLocale;
use App\Http\Controllers\Controller;
use App\Models\Challenge;
use App\Models\TaskTemplate;
use App\Models\UserNotification;
use App\Models\UserTaskCompletion;
use App\Services\Challenges\DailyChallengeService;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomePageService;
use App\Services\MessageSystem\Enums\MessageMode;
use Carbon\Carbon;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class HomeController extends Controller
{
    use ResolvesLocale;

    public function __construct(
        private readonly HomePageService $homePage,
        private readonly DailyChallengeService $challenges,
    ) {}

    /**
     * @OA\Get(
     *     path="/home",
     *     summary="Get the full home page",
     *     description="Aggregates every home-page section into a single, ordered, server-driven feed. Each item is a self-describing section ({key, type, title, order, data}); the frontend renders by `type` and iterates by `order`. Sections with nothing to show are omitted.",
     *     tags={"Home"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="date",
     *         in="query",
     *         description="Reference date (YYYY-MM-DD). Defaults to today.",
     *         required=false,
     *
     *         @OA\Schema(type="string", format="date", example="2026-05-30")
     *     ),
     *
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         description="Language for text responses (en, fa)",
     *         required=false,
     *
     *         @OA\Schema(type="string", default="fa", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Home page assembled successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="mode", type="string", example="cycle"),
     *                 @OA\Property(property="mode_label", type="string", example="چرخه قاعدگی"),
     *                 @OA\Property(property="date", type="string", format="date", example="2026-05-30"),
     *                 @OA\Property(property="locale", type="string", example="fa"),
     *                 @OA\Property(property="profile_complete", type="boolean", example=true),
     *                 @OA\Property(property="sections", type="array", @OA\Items(ref="#/components/schemas/HomeSection"))
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function index(Request $request): JsonResponse
    {
        $context = $this->buildContext($request);

        return response()->json([
            'success' => true,
            'data' => [
                'mode' => $context->mode->value,
                'mode_label' => $context->mode->label($context->locale),
                'date' => $context->date->toDateString(),
                'locale' => $context->locale,
                'profile_complete' => $context->hasCycleData(),
                'sections' => $this->homePage->build($context),
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/home/sections/{section}",
     *     summary="Get a single home-page section",
     *     description="Returns one section by its key for lazy-loading or independent refresh (e.g. tasks, vitals, articles). Use this to refresh a single widget without rebuilding the whole page.",
     *     tags={"Home"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="section",
     *         in="path",
     *         required=true,
     *         description="Section key (e.g. header, week_calendar, next_period, cycle_prediction, recommendations, tasks, challenge, doctor_reminder, medication_reminder, smart_tip, affirmation, weekly_summary, vitals, status_charts, articles, my_cycles, cycle_summary)",
     *
     *         @OA\Schema(type="string", example="tasks")
     *     ),
     *
     *     @OA\Parameter(
     *         name="date",
     *         in="query",
     *         required=false,
     *
     *         @OA\Schema(type="string", format="date")
     *     ),
     *
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         required=false,
     *
     *         @OA\Schema(type="string", default="fa", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Section retrieved (data may be null if the section has nothing to show)",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="section", ref="#/components/schemas/HomeSection", nullable=true)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=404,
     *         description="Unknown section key",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=false),
     *             @OA\Property(property="message", type="string"),
     *             @OA\Property(property="available_sections", type="array", @OA\Items(type="string"))
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function section(Request $request, string $section): JsonResponse
    {
        if (! $this->homePage->hasSection($section)) {
            $locale = $this->resolveLocale($request);

            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'بخش موردنظر یافت نشد' : 'Unknown section',
                'available_sections' => $this->homePage->availableKeys(),
            ], 404);
        }

        $context = $this->buildContext($request);

        return response()->json([
            'success' => true,
            'data' => [
                'section' => $this->homePage->buildSection($section, $context),
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/home/tasks/{task}/toggle",
     *     summary="Toggle today's completion of a task",
     *     description="Marks the given daily task as completed/uncompleted for today and returns updated checklist progress.",
     *     tags={"Home"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(name="task", in="path", required=true, description="Task template id", @OA\Schema(type="integer")),
     *     @OA\Parameter(name="Accept-Language", in="header", required=false, @OA\Schema(type="string", default="fa", enum={"en","fa"})),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Task toggled",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="task_id", type="integer", example=3),
     *                 @OA\Property(property="is_completed", type="boolean", example=true),
     *                 @OA\Property(property="progress", type="object",
     *                     @OA\Property(property="completed", type="integer", example=3),
     *                     @OA\Property(property="total", type="integer", example=4),
     *                     @OA\Property(property="percent", type="integer", example=75)
     *                 )
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Task not found")
     * )
     */
    public function toggleTask(Request $request, TaskTemplate $task): JsonResponse
    {
        $user = $request->user();
        $today = Carbon::today();

        $existing = UserTaskCompletion::query()
            ->where('user_id', $user->id)
            ->where('task_template_id', $task->id)
            ->whereDate('completion_date', $today)
            ->first();

        if ($existing) {
            $existing->delete();
            $isCompleted = false;
        } else {
            UserTaskCompletion::create([
                'user_id' => $user->id,
                'task_template_id' => $task->id,
                'completion_date' => $today,
                'completed_at' => now(),
            ]);
            $isCompleted = true;
        }

        return response()->json([
            'success' => true,
            'data' => [
                'task_id' => $task->id,
                'is_completed' => $isCompleted,
                'progress' => $this->taskProgress($user->id, $today, $request),
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/home/challenges/{challenge}/toggle",
     *     summary="Toggle today's completion of a challenge",
     *     description="Marks the given daily challenge as completed/uncompleted for today.",
     *     tags={"Home"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(name="challenge", in="path", required=true, description="Challenge id", @OA\Schema(type="integer")),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Challenge toggled",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="challenge_id", type="integer", example=2),
     *                 @OA\Property(property="is_completed", type="boolean", example=true)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Challenge not found")
     * )
     */
    public function toggleChallenge(Request $request, Challenge $challenge): JsonResponse
    {
        $isCompleted = $this->challenges->toggle($request->user(), $challenge, Carbon::today());

        return response()->json([
            'success' => true,
            'data' => [
                'challenge_id' => $challenge->id,
                'is_completed' => $isCompleted,
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/home/notifications",
     *     summary="List the user's notifications",
     *     description="Paginated list of in-app notifications (newest first) plus the unread count for the header bell.",
     *     tags={"Home"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(name="per_page", in="query", required=false, @OA\Schema(type="integer", default=20)),
     *     @OA\Parameter(name="Accept-Language", in="header", required=false, @OA\Schema(type="string", default="fa", enum={"en","fa"})),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Notifications retrieved",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="unread_count", type="integer", example=2),
     *                 @OA\Property(property="items", type="array", @OA\Items(type="object")),
     *                 @OA\Property(property="pagination", type="object")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function notifications(Request $request): JsonResponse
    {
        $user = $request->user();
        $locale = $this->resolveLocale($request);
        $perPage = (int) $request->query('per_page', 20);

        $paginator = $user->appNotifications()
            ->orderByDesc('created_at')
            ->paginate(max(1, min($perPage, 100)));

        $items = collect($paginator->items())->map(fn (UserNotification $n) => [
            'id' => $n->id,
            'type' => $n->type,
            'title' => $n->localized('title', $locale),
            'body' => $n->localized('body', $locale),
            'action_url' => $n->action_url,
            'is_read' => $n->isRead(),
            'read_at' => $n->read_at?->toIso8601String(),
            'created_at' => $n->created_at?->toIso8601String(),
        ])->all();

        return response()->json([
            'success' => true,
            'data' => [
                'unread_count' => $user->appNotifications()->unread()->count(),
                'items' => $items,
                'pagination' => [
                    'current_page' => $paginator->currentPage(),
                    'last_page' => $paginator->lastPage(),
                    'per_page' => $paginator->perPage(),
                    'total' => $paginator->total(),
                ],
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/home/notifications/{notification}/read",
     *     summary="Mark a notification as read",
     *     tags={"Home"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(name="notification", in="path", required=true, @OA\Schema(type="integer")),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Marked as read",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="unread_count", type="integer", example=1)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Notification not found")
     * )
     */
    public function markNotificationRead(Request $request, UserNotification $notification): JsonResponse
    {
        $user = $request->user();

        abort_if($notification->user_id !== $user->id, 404);

        if (! $notification->isRead()) {
            $notification->update(['read_at' => now()]);
        }

        return response()->json([
            'success' => true,
            'data' => [
                'unread_count' => $user->appNotifications()->unread()->count(),
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/home/notifications/read-all",
     *     summary="Mark all notifications as read",
     *     tags={"Home"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Response(
     *         response=200,
     *         description="All marked as read",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="marked", type="integer", example=2)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function markAllNotificationsRead(Request $request): JsonResponse
    {
        $user = $request->user();

        $marked = $user->appNotifications()->unread()->update(['read_at' => now()]);

        return response()->json([
            'success' => true,
            'data' => [
                'marked' => $marked,
            ],
        ]);
    }

    /**
     * Assemble the request-scoped HomeContext (locale, date, mode, profiles).
     */
    private function buildContext(Request $request): HomeContext
    {
        // Reject malformed dates with a clean 422 instead of letting
        // Carbon::parse() throw and surface as a 500.
        $request->validate(['date' => 'nullable|date']);

        $user = $request->user();
        $locale = $this->resolveLocale($request);

        $date = $request->query('date')
            ? Carbon::parse($request->query('date'))->startOfDay()
            : Carbon::today();

        $pregnancyProfile = $user->pregnancyProfile;
        $mode = ($pregnancyProfile && $pregnancyProfile->pregnancy_mode)
            ? MessageMode::PREGNANCY
            : MessageMode::CYCLE;

        return new HomeContext(
            user: $user,
            profile: $user->profile,
            pregnancyProfile: $pregnancyProfile,
            date: $date,
            locale: $locale,
            mode: $mode,
        );
    }

    /**
     * Current checklist progress for the given user/day.
     *
     * @return array{completed: int, total: int, percent: int}
     */
    private function taskProgress(int $userId, Carbon $date, Request $request): array
    {
        $context = $this->buildContext($request);
        $phase = $context->phase();

        $templateIds = TaskTemplate::query()
            ->active()
            ->forPhase($phase)
            ->pluck('id');

        $total = $templateIds->count();
        $completed = UserTaskCompletion::query()
            ->where('user_id', $userId)
            ->whereIn('task_template_id', $templateIds)
            ->whereDate('completion_date', $date)
            ->count();

        return [
            'completed' => $completed,
            'total' => $total,
            'percent' => $total > 0 ? (int) round($completed / $total * 100) : 0,
        ];
    }

    /**
     * Resolve locale from Accept-Language header (supports "fa-IR" etc).
     */
}

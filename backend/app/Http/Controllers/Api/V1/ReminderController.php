<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\ReminderType;
use App\Http\Controllers\Concerns\ResolvesLocale;
use App\Http\Controllers\Controller;
use App\Models\Reminder;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Validator;
use Illuminate\Validation\Rule;

/**
 * @OA\Tag(
 *     name="Reminders",
 *     description="User reminders (doctor, medication, appointment, custom)"
 * )
 */
class ReminderController extends Controller
{
    use ResolvesLocale;

    /**
     * @OA\Get(
     *     path="/reminders/enums",
     *     summary="Get reminder enums",
     *     description="Available reminder types and recurrence options for building the reminder form.",
     *     tags={"Reminders"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(name="Accept-Language", in="header", required=false, @OA\Schema(type="string", default="fa", enum={"en","fa"})),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Enums retrieved",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="types", type="array", @OA\Items(type="object")),
     *                 @OA\Property(property="recurrences", type="array", @OA\Items(type="object"))
     *             )
     *         )
     *     ),
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function enums(Request $request): JsonResponse
    {
        $locale = $this->resolveLocale($request);

        $recurrenceLabels = [
            'none' => ['fa' => 'یک‌بار', 'en' => 'One-off'],
            'daily' => ['fa' => 'روزانه', 'en' => 'Daily'],
            'weekly' => ['fa' => 'هفتگی', 'en' => 'Weekly'],
            'monthly' => ['fa' => 'ماهانه', 'en' => 'Monthly'],
        ];

        return response()->json([
            'success' => true,
            'data' => [
                'types' => array_map(fn (ReminderType $t) => [
                    'value' => $t->value,
                    'label' => $t->label($locale),
                    'icon' => $t->icon(),
                ], ReminderType::cases()),
                'recurrences' => array_map(fn (string $r) => [
                    'value' => $r,
                    'label' => $recurrenceLabels[$r][$locale],
                ], array_keys($recurrenceLabels)),
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/reminders",
     *     summary="List reminders",
     *     description="All of the authenticated user's reminders, newest first. Optionally filter by type.",
     *     tags={"Reminders"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(name="type", in="query", required=false, @OA\Schema(type="string", enum={"doctor","medication","appointment","custom"})),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Reminders retrieved",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="array", @OA\Items(ref="#/components/schemas/Reminder"))
     *         )
     *     ),
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function index(Request $request): JsonResponse
    {
        $query = $request->user()->reminders()->orderByDesc('created_at');

        if ($type = $request->query('type')) {
            $query->ofType($type);
        }

        return response()->json([
            'success' => true,
            'data' => $query->get(),
        ]);
    }

    /**
     * @OA\Post(
     *     path="/reminders",
     *     summary="Create a reminder",
     *     tags={"Reminders"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\RequestBody(
     *         required=true,
     *         @OA\JsonContent(
     *             required={"type","title"},
     *             @OA\Property(property="type", type="string", enum={"doctor","medication","appointment","custom"}),
     *             @OA\Property(property="title", type="string", example="ویتامین D"),
     *             @OA\Property(property="subtitle", type="string", nullable=true, example="یک عدد بعد از ناهار"),
     *             @OA\Property(property="notes", type="string", nullable=true),
     *             @OA\Property(property="scheduled_at", type="string", format="date-time", nullable=true),
     *             @OA\Property(property="recurrence", type="string", enum={"none","daily","weekly","monthly"}, default="none"),
     *             @OA\Property(property="recurrence_time", type="string", nullable=true, example="16:00"),
     *             @OA\Property(property="starts_on", type="string", format="date", nullable=true),
     *             @OA\Property(property="ends_on", type="string", format="date", nullable=true)
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=201,
     *         description="Reminder created",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", ref="#/components/schemas/Reminder")
     *         )
     *     ),
     *     @OA\Response(response=422, description="Validation error"),
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function store(Request $request): JsonResponse
    {
        $validator = Validator::make($request->all(), $this->rules(required: true));

        if ($validator->fails()) {
            return $this->validationError($validator, $request);
        }

        $reminder = $request->user()->reminders()->create(
            $validator->validated() + ['is_active' => true]
        );

        return response()->json([
            'success' => true,
            'data' => $reminder,
        ], 201);
    }

    /**
     * @OA\Put(
     *     path="/reminders/{id}",
     *     summary="Update a reminder",
     *     description="Update any subset of the reminder's fields, including toggling is_active.",
     *     tags={"Reminders"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(name="id", in="path", required=true, @OA\Schema(type="integer")),
     *
     *     @OA\RequestBody(
     *         required=true,
     *         @OA\JsonContent(
     *             @OA\Property(property="type", type="string", enum={"doctor","medication","appointment","custom"}),
     *             @OA\Property(property="title", type="string"),
     *             @OA\Property(property="subtitle", type="string", nullable=true),
     *             @OA\Property(property="notes", type="string", nullable=true),
     *             @OA\Property(property="scheduled_at", type="string", format="date-time", nullable=true),
     *             @OA\Property(property="recurrence", type="string", enum={"none","daily","weekly","monthly"}),
     *             @OA\Property(property="recurrence_time", type="string", nullable=true),
     *             @OA\Property(property="starts_on", type="string", format="date", nullable=true),
     *             @OA\Property(property="ends_on", type="string", format="date", nullable=true),
     *             @OA\Property(property="is_active", type="boolean")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Reminder updated",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", ref="#/components/schemas/Reminder")
     *         )
     *     ),
     *     @OA\Response(response=404, description="Not found"),
     *     @OA\Response(response=422, description="Validation error"),
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function update(Request $request, int $id): JsonResponse
    {
        $reminder = $request->user()->reminders()->find($id);

        if (!$reminder) {
            return $this->notFound($request);
        }

        $validator = Validator::make($request->all(), $this->rules(required: false));

        if ($validator->fails()) {
            return $this->validationError($validator, $request);
        }

        $reminder->update($validator->validated());

        return response()->json([
            'success' => true,
            'data' => $reminder->fresh(),
        ]);
    }

    /**
     * @OA\Delete(
     *     path="/reminders/{id}",
     *     summary="Delete a reminder",
     *     tags={"Reminders"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(name="id", in="path", required=true, @OA\Schema(type="integer")),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Reminder deleted",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true)
     *         )
     *     ),
     *     @OA\Response(response=404, description="Not found"),
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function destroy(Request $request, int $id): JsonResponse
    {
        $reminder = $request->user()->reminders()->find($id);

        if (!$reminder) {
            return $this->notFound($request);
        }

        $reminder->delete();

        return response()->json(['success' => true]);
    }

    /** Shared validation rules; `required` toggles create vs partial update. */
    private function rules(bool $required): array
    {
        $presence = $required ? 'required' : 'sometimes';

        return [
            'type' => [$presence, Rule::in(ReminderType::values())],
            'title' => [$presence, 'string', 'max:255'],
            'subtitle' => ['nullable', 'string', 'max:255'],
            'notes' => ['nullable', 'string', 'max:2000'],
            'scheduled_at' => ['nullable', 'date'],
            'recurrence' => ['sometimes', Rule::in(['none', 'daily', 'weekly', 'monthly'])],
            'recurrence_time' => ['nullable', 'date_format:H:i'],
            'starts_on' => ['nullable', 'date'],
            'ends_on' => ['nullable', 'date', 'after_or_equal:starts_on'],
            'is_active' => ['sometimes', 'boolean'],
        ];
    }

    private function validationError($validator, Request $request): JsonResponse
    {
        $locale = $this->resolveLocale($request);

        return response()->json([
            'success' => false,
            'message' => $locale === 'fa' ? 'اطلاعات واردشده نامعتبر است' : 'Validation failed',
            'errors' => $validator->errors(),
        ], 422);
    }

    private function notFound(Request $request): JsonResponse
    {
        $locale = $this->resolveLocale($request);

        return response()->json([
            'success' => false,
            'message' => $locale === 'fa' ? 'یادآور پیدا نشد' : 'Reminder not found',
        ], 404);
    }
}

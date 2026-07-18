<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\CycleSubphase;
use App\Http\Controllers\Controller;
use App\Models\PhaseContent;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * Serves the DB-driven educational content for the "Phase Details" screen the
 * user opens from the daily cycle card. The client passes the current
 * cycle_view.subphase value and gets back the nine localized sections for it.
 */
class PhaseContentController extends Controller
{
    /**
     * @OA\Get(
     *     path="/cycle/phase-content/{phase}",
     *     summary="Get educational content for a cycle sub-phase",
     *     description="Returns the nine localized content sections for a given fine-grained cycle sub-phase (the cycle_view.subphase value), for the Phase Details screen.",
     *     tags={"Cycle"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="phase",
     *         in="path",
     *         description="Sub-phase key (e.g. menstruation, high_fertility, pms_possible)",
     *         required=true,
     *
     *         @OA\Schema(type="string", enum={"menstruation","early_follicular","mid_follicular","fertile_rising","high_fertility","ovulation_likely","post_ovulation","early_luteal","mid_luteal","late_luteal","pms_possible","period_expected"})
     *     ),
     *
     *     @OA\Parameter(
     *         name="locale",
     *         in="query",
     *         description="Language for content (fa, en). Takes precedence over Accept-Language.",
     *         required=false,
     *
     *         @OA\Schema(type="string", default="fa", enum={"fa","en"})
     *     ),
     *
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         required=false,
     *
     *         @OA\Schema(type="string", default="fa", enum={"fa","en"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Phase content retrieved successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="phase", type="string", example="high_fertility"),
     *                 @OA\Property(property="phase_label", type="string", example="باروری بالا"),
     *                 @OA\Property(property="sections", type="object")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=401, description="Unauthenticated"),
     *     @OA\Response(response=404, description="Content not found for this phase"),
     *     @OA\Response(response=422, description="Invalid phase key")
     * )
     */
    public function show(Request $request, string $phase): JsonResponse
    {
        // Browsers forbid JS from overriding Accept-Language, so the SPA passes
        // the active locale as an explicit `?locale=` query param which wins.
        $locale = $request->query('locale', $request->header('Accept-Language', 'fa'));
        $locale = in_array($locale, ['fa', 'en'], true) ? $locale : 'fa';

        $subphase = CycleSubphase::tryFrom($phase);
        if ($subphase === null) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'فاز نامعتبر است' : 'Invalid phase key',
            ], 422);
        }

        $content = PhaseContent::getByPhase($phase);
        if ($content === null) {
            return response()->json([
                'success' => false,
                'message' => $locale === 'fa' ? 'محتوای این فاز یافت نشد' : 'Content not found for this phase',
            ], 404);
        }

        // Only include sections that actually have copy for this locale, so the
        // client can hide missing sections without extra guards.
        $sections = [];
        foreach (PhaseContent::SECTIONS as $field) {
            $value = $content->getLocalizedContent($field, $locale);
            if (is_string($value) && trim($value) !== '') {
                $sections[$field] = $value;
            }
        }

        return response()->json([
            'success' => true,
            'data' => [
                'phase' => $subphase->value,
                'phase_label' => $subphase->label($locale),
                'sections' => $sections,
            ],
        ]);
    }
}

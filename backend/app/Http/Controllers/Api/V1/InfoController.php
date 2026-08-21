<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Concerns\ResolvesLocale;
use App\Http\Controllers\Controller;
use App\Models\InfoSection;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * The admin-managed text screens: راهنما و پشتیبانی, حریم خصوصی, قوانین,
 * درباره ما. All four are the same shape — a list of boxes — so one endpoint
 * serves them, keyed by group.
 */
class InfoController extends Controller
{
    use ResolvesLocale;

    /**
     * @OA\Get(
     *     path="/info/{group}",
     *     summary="Boxes of an in-app text screen",
     *     description="The admin-managed boxes of one text screen, active ones only, in display order and resolved to a single locale. Public: help and policy have to be readable before signing in.",
     *     tags={"Info"},
     *
     *     @OA\Parameter(
     *         name="group",
     *         in="path",
     *         required=true,
     *         description="Which screen to fetch.",
     *
     *         @OA\Schema(type="string", enum={"help","privacy","terms","about"})
     *     ),
     *
     *     @OA\Parameter(
     *         name="locale",
     *         in="query",
     *         required=false,
     *         description="Overrides Accept-Language.",
     *
     *         @OA\Schema(type="string", enum={"en","fa"})
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
     *         description="Sections of the requested screen",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="group", type="string", example="help"),
     *                 @OA\Property(property="sections", type="array", @OA\Items(ref="#/components/schemas/InfoSection"))
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(response=404, description="Unknown group")
     * )
     */
    public function show(Request $request, string $group): JsonResponse
    {
        abort_unless(InfoSection::isGroup($group), 404);

        return $this->sections($request, $group);
    }

    /**
     * @OA\Get(
     *     path="/privacy",
     *     summary="Privacy-policy sections (legacy alias)",
     *     description="Deprecated alias of /info/privacy, kept for app versions shipped before the text screens were generalised.",
     *     tags={"Info"},
     *     deprecated=true,
     *
     *     @OA\Parameter(name="locale", in="query", required=false, @OA\Schema(type="string", enum={"en","fa"})),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Privacy sections",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="sections", type="array", @OA\Items(ref="#/components/schemas/InfoSection"))
     *             )
     *         )
     *     )
     * )
     */
    public function privacy(Request $request): JsonResponse
    {
        return $this->sections($request, InfoSection::GROUP_PRIVACY);
    }

    private function sections(Request $request, string $group): JsonResponse
    {
        // Browsers drop a JS-set Accept-Language header, so the web client sends
        // `?locale=` instead; when present it wins over the header.
        $query = $request->query('locale');
        $locale = in_array($query, ['fa', 'en'], true)
            ? $query
            : $this->resolveLocale($request);

        $sections = InfoSection::inGroup($group)->active()->ordered()->get()
            ->map(fn (InfoSection $section) => $section->toLocalizedArray($locale))
            ->values();

        return response()->json([
            'success' => true,
            'data' => ['group' => $group, 'sections' => $sections],
        ]);
    }
}

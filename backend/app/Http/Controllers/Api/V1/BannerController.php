<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\BannerPosition;
use App\Http\Controllers\Concerns\ResolvesLocale;
use App\Http\Controllers\Controller;
use App\Models\Banner;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class BannerController extends Controller
{
    use ResolvesLocale;

    /**
     * @OA\Get(
     *     path="/banners",
     *     summary="Active home-page banners grouped by slot",
     *     description="Returns the currently active, in-window banners for each home-page slot, ordered for display. Slots with no active banner come back as empty arrays, so the client can render a slideshow per slot or nothing at all.",
     *     tags={"Banners"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Parameter(
     *         name="position",
     *         in="query",
     *         required=false,
     *         description="Restrict to a single slot (home_top, home_middle, home_bottom).",
     *         @OA\Schema(type="string", enum={"home_top","home_middle","home_bottom"})
     *     ),
     *     @OA\Parameter(
     *         name="Accept-Language",
     *         in="header",
     *         required=false,
     *         @OA\Schema(type="string", default="fa", enum={"en","fa"})
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Banners grouped by slot",
     *         @OA\JsonContent(
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="positions", type="object",
     *                     @OA\Property(property="home_top", type="array", @OA\Items(ref="#/components/schemas/Banner")),
     *                     @OA\Property(property="home_middle", type="array", @OA\Items(ref="#/components/schemas/Banner")),
     *                     @OA\Property(property="home_bottom", type="array", @OA\Items(ref="#/components/schemas/Banner"))
     *                 )
     *             )
     *         )
     *     ),
     *     @OA\Response(response=401, description="Unauthenticated")
     * )
     */
    public function index(Request $request): JsonResponse
    {
        $locale = $this->resolveLocale($request);

        $only = $request->query('position');
        $positions = ($only && in_array($only, BannerPosition::values(), true))
            ? [BannerPosition::from($only)]
            : BannerPosition::cases();

        $active = Banner::active()
            ->whereIn('position', array_map(fn (BannerPosition $p) => $p->value, $positions))
            ->get()
            ->groupBy('position');

        $grouped = [];
        foreach ($positions as $position) {
            $grouped[$position->value] = $active->get($position->value, collect())
                ->map(fn (Banner $banner) => $this->present($banner, $locale))
                ->values();
        }

        return response()->json([
            'success' => true,
            'data' => [
                'positions' => $grouped,
            ],
        ]);
    }

    /**
     * Shape a banner for the client — only fields the UI renders.
     *
     * @return array<string, mixed>
     */
    private function present(Banner $banner, string $locale): array
    {
        return [
            'id' => $banner->id,
            'title' => $banner->localized('title', $locale),
            'image_url' => $banner->image_url,
            'position' => $banner->position,
            'link_url' => $banner->link_url,
            'link_type' => $banner->link_type,
        ];
    }
}

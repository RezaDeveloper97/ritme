<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Services\Language\LanguageRegistry;
use App\Services\Language\TranslationStore;
use Illuminate\Http\JsonResponse;

/**
 * The locales the app ships and the UI strings for each one.
 *
 * Both endpoints are public and unauthenticated on purpose: the client has to
 * pick a language and render its interface before anyone signs in. They are
 * also the mechanism that makes a language added in the admin panel appear in
 * the app without redeploying the frontend — the client reads this list at
 * runtime rather than baking locales in at build time.
 */
class LanguageController extends Controller
{
    public function __construct(
        private readonly LanguageRegistry $registry,
        private readonly TranslationStore $translations,
    ) {}

    /**
     * @OA\Get(
     *     path="/languages",
     *     summary="Languages the app supports",
     *     description="Active locales in display order, each with its endonym and text direction. The client builds its language picker and <html dir> from this. Public.",
     *     tags={"Languages"},
     *
     *     @OA\Response(
     *         response=200,
     *         description="Supported languages",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="default", type="string", example="fa"),
     *                 @OA\Property(property="languages", type="array", @OA\Items(ref="#/components/schemas/Language"))
     *             )
     *         )
     *     )
     * )
     */
    public function index(): JsonResponse
    {
        return response()->json([
            'success' => true,
            'data' => [
                'default' => $this->registry->defaultCode(),
                'languages' => array_map(
                    fn (array $language): array => [
                        'code' => $language['code'],
                        'name' => $language['name'],
                        'english_name' => $language['english_name'],
                        'direction' => $language['direction'],
                        'is_default' => $language['is_default'],
                    ],
                    $this->registry->all()
                ),
            ],
        ]);
    }

    /**
     * @OA\Get(
     *     path="/languages/{code}/messages",
     *     summary="UI string bundle for one language",
     *     description="Every interface string for the locale, grouped by namespace. Keys the locale has not translated yet are filled in from the default language, so the response is always complete. Public.",
     *     tags={"Languages"},
     *
     *     @OA\Parameter(
     *         name="code",
     *         in="path",
     *         required=true,
     *         description="Language code from /languages. An unsupported code falls back to the default language.",
     *
     *         @OA\Schema(type="string", example="fa")
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="Message bundle",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="locale", type="string", example="fa"),
     *                 @OA\Property(property="direction", type="string", example="rtl"),
     *                 @OA\Property(property="messages", type="object", description="namespace => nested ICU messages")
     *             )
     *         )
     *     )
     * )
     */
    public function messages(string $code): JsonResponse
    {
        // Never 404: an old client asking for a locale that has since been
        // removed should keep rendering, in the default language.
        $locale = $this->registry->resolve($code);

        return response()->json([
            'success' => true,
            'data' => [
                'locale' => $locale,
                'direction' => $this->registry->direction($locale)->value,
                'messages' => $this->translations->bundle($locale),
            ],
        ]);
    }
}

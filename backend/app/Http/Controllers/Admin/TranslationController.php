<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Language;
use App\Services\Language\LanguageRegistry;
use App\Services\Language\TranslationStore;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\View\View;

/**
 * Editor for a locale's UI strings — the JSON bundles the app renders its
 * interface from (buttons, labels, empty states), one namespace at a time.
 *
 * Content stored in the database (articles, challenges, banners…) is edited on
 * its own page; this is only for the fixed interface copy that ships with the
 * frontend. A newly created language starts as a copy of the default locale,
 * so this page is where an admin turns that copy into an actual translation.
 */
class TranslationController extends Controller
{
    public function __construct(
        private readonly LanguageRegistry $registry,
        private readonly TranslationStore $store,
    ) {}

    public function index(Request $request, Language $language): View
    {
        $namespaces = $this->store->namespaces();
        $namespace = $this->namespace($request, $namespaces);
        $defaultCode = $this->registry->defaultCode();

        // What the admin edits, and what the default locale says for the same
        // key — shown side by side so they can translate without guessing.
        $current = $this->store->flatten($this->store->rawNamespace($language->code, $namespace));
        $reference = $this->store->flatten($this->store->rawNamespace($defaultCode, $namespace));

        // Keys the reference has but this locale doesn't yet — they render via
        // fallback today, and must still be listed so they can be translated.
        $rows = collect($reference)
            ->keys()
            ->merge(array_keys($current))
            ->unique()
            ->sort()
            ->map(fn (string $key): array => [
                'key' => $key,
                'value' => $current[$key] ?? '',
                'reference' => $reference[$key] ?? '',
            ])
            ->values();

        return view('admin.translations.index', [
            'language' => $language,
            'namespaces' => $namespaces,
            'namespace' => $namespace,
            'rows' => $rows,
            'defaultCode' => $defaultCode,
            'defaultName' => $this->registry->name($defaultCode),
            'isDefaultLocale' => $language->code === $defaultCode,
        ]);
    }

    public function update(Request $request, Language $language): RedirectResponse
    {
        $namespaces = $this->store->namespaces();
        $namespace = $this->namespace($request, $namespaces);

        // Rows travel as an indexed list rather than key => value: a message
        // key contains dots ("health.days"), which PHP's form parser would
        // otherwise turn into nested input names.
        $data = $request->validate([
            'rows' => ['nullable', 'array'],
            'rows.*.key' => ['required', 'string', 'max:255'],
            'rows.*.value' => ['nullable', 'string'],
        ]);

        $flat = [];

        foreach ($data['rows'] ?? [] as $row) {
            $flat[$row['key']] = (string) ($row['value'] ?? '');
        }

        $this->store->writeNamespace($language->code, $namespace, $this->store->unflatten($flat));

        return redirect()
            ->route('admin.languages.translations.index', ['language' => $language, 'namespace' => $namespace])
            ->with('status', "ترجمه‌های «{$namespace}» ذخیره شد.");
    }

    /**
     * The namespace being edited; anything unrecognised falls back to the first.
     *
     * @param  array<int, string>  $namespaces
     */
    private function namespace(Request $request, array $namespaces): string
    {
        $requested = (string) $request->input('namespace', '');

        return in_array($requested, $namespaces, true)
            ? $requested
            : ($namespaces[0] ?? 'common');
    }
}

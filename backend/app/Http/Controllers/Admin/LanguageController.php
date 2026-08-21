<?php

namespace App\Http\Controllers\Admin;

use App\Enums\TextDirection;
use App\Http\Controllers\Controller;
use App\Models\Language;
use App\Services\Language\LanguageProvisioner;
use App\Services\Language\LanguageRegistry;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;
use Illuminate\View\View;

/**
 * The languages the product ships — /admin/languages.
 *
 * This page is the switch behind every other multi-language behaviour: adding
 * a row here grows every content form in the panel with an input for the new
 * locale, generates the locale's UI-string files (so the frontend can serve
 * it immediately), and makes it selectable in the app's language picker.
 */
class LanguageController extends Controller
{
    public function __construct(
        private readonly LanguageRegistry $registry,
        private readonly LanguageProvisioner $provisioner,
    ) {}

    public function index(): View
    {
        return view('admin.languages.index', [
            'languages' => Language::query()->ordered()->get(),
        ]);
    }

    public function create(): View
    {
        return view('admin.languages.form', $this->formData(new Language([
            'direction' => TextDirection::LTR,
            'is_active' => true,
            'sort_order' => ((int) Language::max('sort_order')) + 10,
        ])));
    }

    public function store(Request $request): RedirectResponse
    {
        $data = $this->validated($request);
        $copyFrom = $this->copySource($request);

        $language = Language::create($data);

        $this->applyDefault($language);
        $this->registry->flush();

        // Generate the locale's files AFTER the registry knows about it, so
        // TranslationStore sees the new code as a supported locale.
        $generated = $this->provisioner->provision($language, $copyFrom);

        return redirect()
            ->route('admin.languages.index')
            ->with('status', sprintf(
                'زبان «%s» ساخته شد؛ %d فایل ترجمه از «%s» ساخته شد.',
                $language->name,
                $generated['messages'],
                $this->registry->name($copyFrom),
            ));
    }

    public function edit(Language $language): View
    {
        return view('admin.languages.form', $this->formData($language));
    }

    public function update(Request $request, Language $language): RedirectResponse
    {
        $language->update($this->validated($request, $language));

        $this->applyDefault($language);
        $this->registry->flush();

        return redirect()
            ->route('admin.languages.index')
            ->with('status', 'زبان به‌روزرسانی شد.');
    }

    public function destroy(Language $language): RedirectResponse
    {
        // The default locale is every other locale's fallback — deleting it
        // would leave half-translated content with nothing to fall back to.
        if ($language->is_default) {
            return back()->with('error', 'زبان پیش‌فرض حذف نمی‌شود. ابتدا زبان دیگری را پیش‌فرض کنید.');
        }

        $code = $language->code;
        $language->delete();

        $this->provisioner->deprovision($code);
        $this->registry->flush();

        return back()->with('status', 'زبان و فایل‌های ترجمه‌اش حذف شد.');
    }

    public function toggle(Language $language): RedirectResponse
    {
        if ($language->is_default && $language->is_active) {
            return back()->with('error', 'زبان پیش‌فرض غیرفعال نمی‌شود.');
        }

        $language->update(['is_active' => ! $language->is_active]);
        $this->registry->flush();

        return back()->with('status', 'وضعیت زبان تغییر کرد.');
    }

    /** Re-generate a locale's translation files from another locale. */
    public function regenerate(Request $request, Language $language): RedirectResponse
    {
        $source = $this->registry->isSupported($request->input('copy_from'))
            ? Language::normalizeCode($request->string('copy_from')->value())
            : $this->registry->defaultCode();

        $result = $this->provisioner->provision($language, $source);

        return back()->with('status', "{$result['messages']} فایل ترجمه دوباره ساخته شد.");
    }

    /** @return array<string, mixed> */
    private function formData(Language $language): array
    {
        return [
            'language' => $language,
            'directions' => TextDirection::cases(),
            'sources' => Language::query()->active()->ordered()->get(),
            'defaultCode' => $this->registry->defaultCode(),
        ];
    }

    /**
     * Which locale a new language copies its text from. Defaults to the
     * product default so the app is fully usable in the new language at once.
     */
    private function copySource(Request $request): string
    {
        $requested = $request->input('copy_from');

        return $this->registry->isSupported($requested)
            ? Language::normalizeCode((string) $requested)
            : $this->registry->defaultCode();
    }

    /**
     * Keep exactly one default. Marking a language default demotes the others;
     * a default language is implicitly active (users must be able to reach it).
     */
    private function applyDefault(Language $language): void
    {
        if (! $language->is_default) {
            // Never leave the table without a default — if this row was the
            // default and got unticked, the first active row takes over.
            if (! Language::query()->where('is_default', true)->exists()) {
                Language::query()->active()->ordered()->first()?->update(['is_default' => true]);
            }

            return;
        }

        Language::query()
            ->where('id', '!=', $language->id)
            ->where('is_default', true)
            ->update(['is_default' => false]);

        if (! $language->is_active) {
            $language->update(['is_active' => true]);
        }
    }

    /** @return array<string, mixed> */
    private function validated(Request $request, ?Language $language = null): array
    {
        $data = $request->validate([
            // Locale codes end up in URLs (/pt-br/home), JSON content keys and
            // Accept-Language, so keep them to the BCP-47 shape.
            'code' => [
                'required', 'string', 'max:12', 'regex:/^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})?$/',
                Rule::unique('languages', 'code')->ignore($language?->id),
            ],
            'name' => ['required', 'string', 'max:60'],
            'english_name' => ['required', 'string', 'max:60'],
            'direction' => ['required', Rule::enum(TextDirection::class)],
            'sort_order' => ['nullable', 'integer'],
        ], [
            'code.regex' => 'کد زبان باید مانند fa، en یا pt-BR باشد.',
            'code.unique' => 'این کد زبان قبلاً ثبت شده است.',
        ]);

        $data['code'] = Language::normalizeCode($data['code']);
        $data['sort_order'] = $data['sort_order'] ?? 0;
        $data['is_active'] = $request->boolean('is_active');
        $data['is_default'] = $request->boolean('is_default');

        return $data;
    }
}

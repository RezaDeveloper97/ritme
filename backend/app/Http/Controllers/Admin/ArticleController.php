<?php

namespace App\Http\Controllers\Admin;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Http\Controllers\Controller;
use App\Models\Article;
use App\Services\Media\ImageOptimizer;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Storage;
use Illuminate\Validation\Rule;
use Illuminate\View\View;

class ArticleController extends Controller
{
    /** Where optimized cover images land on the "public" disk. */
    private const IMAGE_DIRECTORY = 'articles';

    public function __construct(private readonly ImageOptimizer $images) {}

    public function index(): View
    {
        $articles = Article::orderBy('sort_order')->orderByDesc('id')->paginate(20);

        return view('admin.articles.index', compact('articles'));
    }

    public function create(): View
    {
        return view('admin.articles.form', $this->formData(new Article(['is_published' => true])));
    }

    public function store(Request $request): RedirectResponse
    {
        $data = $this->validated($request);

        if ($request->hasFile('image')) {
            $data['image_path'] = $this->images->store($request->file('image'), self::IMAGE_DIRECTORY);
        }

        Article::create($data);

        return redirect()->route('admin.articles.index')->with('status', 'مقاله ایجاد شد.');
    }

    public function edit(Article $article): View
    {
        return view('admin.articles.form', $this->formData($article));
    }

    public function update(Request $request, Article $article): RedirectResponse
    {
        $data = $this->validated($request, $article);
        $previousImage = $article->image_path;

        if ($request->hasFile('image')) {
            $data['image_path'] = $this->images->store($request->file('image'), self::IMAGE_DIRECTORY);
        } elseif ($request->boolean('remove_image')) {
            $data['image_path'] = null;
        }

        $article->update($data);

        if (array_key_exists('image_path', $data) && $previousImage !== $article->image_path) {
            $this->deleteImage($previousImage);
        }

        return redirect()->route('admin.articles.index')->with('status', 'مقاله به‌روزرسانی شد.');
    }

    public function destroy(Article $article): RedirectResponse
    {
        $this->deleteImage($article->image_path);
        $article->delete();

        return back()->with('status', 'مقاله حذف شد.');
    }

    public function toggle(Article $article): RedirectResponse
    {
        $article->update([
            'is_published' => ! $article->is_published,
            'published_at' => $article->is_published ? $article->published_at : now(),
        ]);

        return back()->with('status', 'وضعیت انتشار تغییر کرد.');
    }

    /**
     * Shared create/edit view data.
     */
    private function formData(Article $article): array
    {
        return [
            'article' => $article,
            'phases' => $this->phaseOptions($article->cycle_phases ?? []),
            'maxImageWidth' => ImageOptimizer::MAX_WIDTH,
        ];
    }

    /**
     * Phase options for the picker.
     *
     * The list is exactly what the "محتوای فازهای چرخه" page (/admin/phase-contents)
     * offers — both read CycleSubphase::options(), so adding or removing a phase
     * there changes this form too, with nothing to keep in sync by hand.
     *
     * A row saved before this page moved to sub-phases holds a main-phase key
     * (e.g. "follicular"); those values are appended as their own options so
     * editing such an article doesn't silently clear its phases.
     *
     * @param  array<int, string>  $current
     * @return array<int, array{value: string, label: string}>
     */
    private function phaseOptions(array $current): array
    {
        $options = CycleSubphase::options();
        $known = CycleSubphase::values();

        foreach (array_unique($current) as $phase) {
            if (! in_array($phase, $known, true)) {
                $options[] = [
                    'value' => $phase,
                    'label' => (CyclePhase::labelFor($phase) ?? $phase).' (قدیمی)',
                ];
            }
        }

        return $options;
    }

    /**
     * Phase keys a row may be saved with: the current sub-phase list, plus the
     * legacy main-phase keys so old rows stay editable.
     *
     * @return array<int, string>
     */
    private function allowedPhases(): array
    {
        return array_values(array_unique([...CycleSubphase::values(), ...CyclePhase::values()]));
    }

    private function validated(Request $request, ?Article $article = null): array
    {
        $data = $request->validate([
            'slug' => ['required', 'string', 'max:255', Rule::unique('articles', 'slug')->ignore($article)],
            'title' => ['required', 'array'],
            'title.fa' => ['required', 'string'],
            'title.en' => ['nullable', 'string'],
            'excerpt' => ['nullable', 'array'],
            'body' => ['nullable', 'array'],
            'cycle_phases' => ['nullable', 'array'],
            'cycle_phases.*' => [Rule::in($this->allowedPhases())],
            'category' => ['nullable', 'string', 'max:255'],
            'read_time_minutes' => ['nullable', 'integer', 'min:1', 'max:120'],
            // Images only — the mimes list is enforced on the decoded file, not
            // on the client-supplied name.
            'image' => ['nullable', 'image', 'mimes:jpeg,jpg,png,webp', 'max:8192'],
            'image_url' => ['nullable', 'string', 'max:1000'],
            'sort_order' => ['nullable', 'integer'],
        ], [
            'image.image' => 'فایل انتخاب‌شده باید یک تصویر باشد.',
            'image.mimes' => 'فرمت تصویر باید JPG، PNG یا WebP باشد.',
            'image.max' => 'حجم تصویر نباید بیشتر از ۸ مگابایت باشد.',
        ]);

        $data['is_published'] = $request->boolean('is_published');
        $data['sort_order'] = $data['sort_order'] ?? 0;
        // "No phase ticked" means general, and that is stored as null — an empty
        // array would read as "tagged, but with nothing" to every query below.
        $data['cycle_phases'] = array_values(array_unique($data['cycle_phases'] ?? [])) ?: null;

        if ($data['is_published'] && ! ($article?->published_at)) {
            $data['published_at'] = now();
        }

        // The upload is stored separately; never mass-assign the file itself.
        unset($data['image']);

        return $data;
    }

    private function deleteImage(?string $path): void
    {
        if ($path && Storage::disk('public')->exists($path)) {
            Storage::disk('public')->delete($path);
        }
    }
}

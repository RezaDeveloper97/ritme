<?php

namespace App\Http\Controllers\Admin;

use App\Enums\CyclePhase;
use App\Http\Controllers\Controller;
use App\Models\Article;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;
use Illuminate\View\View;

class ArticleController extends Controller
{
    public function index(): View
    {
        $articles = Article::orderBy('sort_order')->orderByDesc('id')->paginate(20);

        return view('admin.articles.index', compact('articles'));
    }

    public function create(): View
    {
        return view('admin.articles.form', [
            'article' => new Article(['is_published' => true]),
            'phases' => CyclePhase::cases(),
        ]);
    }

    public function store(Request $request): RedirectResponse
    {
        Article::create($this->validated($request));

        return redirect()->route('admin.articles.index')->with('status', 'مقاله ایجاد شد.');
    }

    public function edit(Article $article): View
    {
        return view('admin.articles.form', [
            'article' => $article,
            'phases' => CyclePhase::cases(),
        ]);
    }

    public function update(Request $request, Article $article): RedirectResponse
    {
        $article->update($this->validated($request, $article));

        return redirect()->route('admin.articles.index')->with('status', 'مقاله به‌روزرسانی شد.');
    }

    public function destroy(Article $article): RedirectResponse
    {
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

    private function validated(Request $request, ?Article $article = null): array
    {
        $data = $request->validate([
            'slug' => ['required', 'string', 'max:255', Rule::unique('articles', 'slug')->ignore($article)],
            'title' => ['required', 'array'],
            'title.fa' => ['required', 'string'],
            'title.en' => ['nullable', 'string'],
            'excerpt' => ['nullable', 'array'],
            'body' => ['nullable', 'array'],
            'cycle_phase' => ['nullable', Rule::in(array_column(CyclePhase::cases(), 'value'))],
            'category' => ['nullable', 'string', 'max:255'],
            'read_time_minutes' => ['nullable', 'integer', 'min:1', 'max:120'],
            'image_url' => ['nullable', 'string', 'max:1000'],
            'sort_order' => ['nullable', 'integer'],
        ]);

        $data['is_published'] = $request->boolean('is_published');
        $data['sort_order'] = $data['sort_order'] ?? 0;

        if ($data['is_published'] && ! ($article?->published_at)) {
            $data['published_at'] = now();
        }

        return $data;
    }
}

<?php

namespace App\Http\Controllers\Admin;

use App\Enums\BannerLinkType;
use App\Enums\BannerPosition;
use App\Http\Controllers\Controller;
use App\Models\Banner;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Storage;
use Illuminate\Validation\Rule;
use Illuminate\View\View;

class BannerController extends Controller
{
    /**
     * Recommended banner image — a 2:1 slide that stays crisp on retina phones.
     * Surfaced in the form and enforced (min side) at upload time.
     */
    private const RECOMMENDED_WIDTH = 1080;
    private const RECOMMENDED_HEIGHT = 540;

    public function index(): View
    {
        $banners = Banner::orderBy('position')
            ->orderBy('sort_order')
            ->orderByDesc('id')
            ->paginate(20);

        return view('admin.banners.index', [
            'banners' => $banners,
            'positions' => BannerPosition::cases(),
        ]);
    }

    public function create(): View
    {
        return view('admin.banners.form', $this->formData(new Banner([
            'is_active' => true,
            'position' => BannerPosition::HOME_TOP->value,
        ])));
    }

    public function store(Request $request): RedirectResponse
    {
        $data = $this->validated($request, null);
        $data['image_path'] = $this->storeImage($request);

        Banner::create($data);

        return redirect()->route('admin.banners.index')->with('status', 'بنر ایجاد شد.');
    }

    public function edit(Banner $banner): View
    {
        return view('admin.banners.form', $this->formData($banner));
    }

    public function update(Request $request, Banner $banner): RedirectResponse
    {
        $data = $this->validated($request, $banner);

        if ($request->hasFile('image')) {
            $data['image_path'] = $this->storeImage($request);
            $this->deleteImage($banner->image_path);
        }

        $banner->update($data);

        return redirect()->route('admin.banners.index')->with('status', 'بنر به‌روزرسانی شد.');
    }

    public function destroy(Banner $banner): RedirectResponse
    {
        $this->deleteImage($banner->image_path);
        $banner->delete();

        return back()->with('status', 'بنر حذف شد.');
    }

    public function toggle(Banner $banner): RedirectResponse
    {
        $banner->update(['is_active' => ! $banner->is_active]);

        return back()->with('status', 'وضعیت بنر تغییر کرد.');
    }

    /**
     * Shared create/edit view data.
     */
    private function formData(Banner $banner): array
    {
        return [
            'banner' => $banner,
            'positions' => BannerPosition::cases(),
            'linkTypes' => BannerLinkType::cases(),
            'recommended' => self::RECOMMENDED_WIDTH.'×'.self::RECOMMENDED_HEIGHT,
        ];
    }

    private function validated(Request $request, ?Banner $banner): array
    {
        $data = $request->validate([
            'title' => ['nullable', 'array'],
            'title.fa' => ['nullable', 'string', 'max:255'],
            'title.en' => ['nullable', 'string', 'max:255'],
            // Required on create, optional on edit (keep the existing image).
            'image' => [
                $banner ? 'nullable' : 'required',
                'image',
                'mimes:jpeg,jpg,png,webp',
                'max:4096', // KB
                'dimensions:min_width=800,min_height=400',
            ],
            'position' => ['required', Rule::in(BannerPosition::values())],
            'link_type' => ['nullable', Rule::in(BannerLinkType::values())],
            'link_url' => [
                'nullable',
                'required_with:link_type',
                'string',
                'max:1000',
                // External links must be absolute URLs; internal ones are app paths.
                Rule::when(
                    $request->input('link_type') === BannerLinkType::EXTERNAL->value,
                    ['url'],
                ),
            ],
            'starts_at' => ['nullable', 'date'],
            'ends_at' => ['nullable', 'date', 'after_or_equal:starts_at'],
            'sort_order' => ['nullable', 'integer'],
        ]);

        $data['is_active'] = $request->boolean('is_active');
        $data['sort_order'] = $data['sort_order'] ?? 0;

        // Drop link fields entirely when no URL was provided.
        if (empty($data['link_url'])) {
            $data['link_url'] = null;
            $data['link_type'] = null;
        }

        // `image` is handled separately (file upload); never mass-assign it.
        unset($data['image']);

        return $data;
    }

    private function storeImage(Request $request): string
    {
        return $request->file('image')->store('banners', 'public');
    }

    private function deleteImage(?string $path): void
    {
        if ($path && Storage::disk('public')->exists($path)) {
            Storage::disk('public')->delete($path);
        }
    }
}

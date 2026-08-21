<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\InfoSection;
use App\Support\Translatable;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\View\View;

/**
 * CRUD for the boxes that make up the app's text screens — راهنما و پشتیبانی,
 * حریم خصوصی, قوانین, درباره ما.
 *
 * One screen at a time: `?group=` picks which one is being edited, and the
 * chosen group follows the admin through create/edit/delete so they land back
 * on the tab they came from.
 */
class InfoSectionController extends Controller
{
    public function index(Request $request): View
    {
        $group = $this->group($request);

        return view('admin.info-sections.index', [
            'group' => $group,
            'sections' => InfoSection::inGroup($group)->ordered()->paginate(20)->withQueryString(),
        ]);
    }

    public function create(Request $request): View
    {
        $group = $this->group($request);

        return view('admin.info-sections.form', [
            'group' => $group,
            'section' => new InfoSection([
                'group' => $group,
                'is_active' => true,
                // Slot a new box after everything in this screen, leaving a gap
                // so it can be nudged around later without renumbering.
                'sort_order' => ((int) InfoSection::inGroup($group)->max('sort_order')) + 10,
            ]),
        ]);
    }

    public function store(Request $request): RedirectResponse
    {
        $section = InfoSection::create($this->validated($request));

        return $this->backToIndex($section->group, 'باکس ایجاد شد.');
    }

    public function edit(InfoSection $infoSection): View
    {
        return view('admin.info-sections.form', [
            'group' => $infoSection->group,
            'section' => $infoSection,
        ]);
    }

    public function update(Request $request, InfoSection $infoSection): RedirectResponse
    {
        $infoSection->update($this->validated($request));

        return $this->backToIndex($infoSection->group, 'باکس به‌روزرسانی شد.');
    }

    public function destroy(InfoSection $infoSection): RedirectResponse
    {
        $infoSection->delete();

        return back()->with('status', 'باکس حذف شد.');
    }

    public function toggle(InfoSection $infoSection): RedirectResponse
    {
        $infoSection->update(['is_active' => ! $infoSection->is_active]);

        return back()->with('status', 'وضعیت تغییر کرد.');
    }

    /** The screen being edited; anything unrecognised falls back to the first tab. */
    private function group(Request $request): string
    {
        $group = $request->query('group');

        return InfoSection::isGroup($group) ? $group : InfoSection::GROUPS[0];
    }

    private function backToIndex(string $group, string $status): RedirectResponse
    {
        return redirect()
            ->route('admin.info-sections.index', ['group' => $group])
            ->with('status', $status);
    }

    /** @return array<string, mixed> */
    private function validated(Request $request): array
    {
        $data = $request->validate([
            'group' => ['required', 'string', 'in:'.implode(',', InfoSection::GROUPS)],
            ...Translatable::rulesFor([
                'heading' => ['required' => true, 'extra' => ['max:200']],
                'body' => ['required' => true],
                'link_label' => ['required' => false, 'extra' => ['max:60']],
            ]),
            // mailto: and tel: matter as much as https here — a support box is
            // usually an email or a phone number, which `url` would reject.
            'link_url' => ['nullable', 'string', 'max:500', 'regex:/^(https?:\/\/|mailto:|tel:)/i'],
            'sort_order' => ['nullable', 'integer'],
        ], [
            'link_url.regex' => 'لینک باید با https:// یا mailto: یا tel: شروع شود.',
        ]);

        $data['is_active'] = $request->boolean('is_active');
        $data['sort_order'] = $data['sort_order'] ?? 0;
        // A cleared or absent link has to become an explicit null, not a
        // missing key: on update, a missing key would leave the old button in
        // place. An untouched translatable input also arrives as one empty
        // string per language, which must not read as "blank button".
        $data['link_label'] = Translatable::clean($data['link_label'] ?? null);
        $data['link_url'] = ($data['link_url'] ?? null) ?: null;

        return $data;
    }
}

<?php

namespace App\Http\Controllers\Admin;

use App\Enums\CyclePhase;
use App\Enums\TaskCategory;
use App\Http\Controllers\Controller;
use App\Models\TaskTemplate;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;
use Illuminate\View\View;

class TaskTemplateController extends Controller
{
    public function index(): View
    {
        $tasks = TaskTemplate::orderBy('sort_order')->orderByDesc('id')->paginate(20);

        return view('admin.task-templates.index', compact('tasks'));
    }

    public function create(): View
    {
        return view('admin.task-templates.form', [
            'task' => new TaskTemplate(['is_active' => true]),
            'phases' => CyclePhase::cases(),
            'categories' => TaskCategory::cases(),
        ]);
    }

    public function store(Request $request): RedirectResponse
    {
        TaskTemplate::create($this->validated($request));

        return redirect()->route('admin.task-templates.index')->with('status', 'کار ایجاد شد.');
    }

    public function edit(TaskTemplate $taskTemplate): View
    {
        return view('admin.task-templates.form', [
            'task' => $taskTemplate,
            'phases' => CyclePhase::cases(),
            'categories' => TaskCategory::cases(),
        ]);
    }

    public function update(Request $request, TaskTemplate $taskTemplate): RedirectResponse
    {
        $taskTemplate->update($this->validated($request, $taskTemplate));

        return redirect()->route('admin.task-templates.index')->with('status', 'کار به‌روزرسانی شد.');
    }

    public function destroy(TaskTemplate $taskTemplate): RedirectResponse
    {
        $taskTemplate->delete();

        return back()->with('status', 'کار حذف شد.');
    }

    public function toggle(TaskTemplate $taskTemplate): RedirectResponse
    {
        $taskTemplate->update(['is_active' => ! $taskTemplate->is_active]);

        return back()->with('status', 'وضعیت تغییر کرد.');
    }

    private function validated(Request $request, ?TaskTemplate $task = null): array
    {
        $data = $request->validate([
            'key' => ['required', 'string', 'max:255', Rule::unique('task_templates', 'key')->ignore($task)],
            'title' => ['required', 'array'],
            'title.fa' => ['required', 'string'],
            'title.en' => ['nullable', 'string'],
            'description' => ['nullable', 'array'],
            'category' => ['required', Rule::in(array_column(TaskCategory::cases(), 'value'))],
            'icon' => ['nullable', 'string', 'max:255'],
            'cycle_phase' => ['nullable', Rule::in(array_column(CyclePhase::cases(), 'value'))],
            'sort_order' => ['nullable', 'integer'],
        ]);

        $data['is_active'] = $request->boolean('is_active');
        $data['sort_order'] = $data['sort_order'] ?? 0;

        return $data;
    }
}

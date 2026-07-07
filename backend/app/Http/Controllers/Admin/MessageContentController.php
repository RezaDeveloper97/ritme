<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\MessageContent;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\View\View;

class MessageContentController extends Controller
{
    /** Human-readable Persian names for the message groups. */
    public const GROUP_LABELS = [
        'cycle_base_non_ttc' => 'پیام پایه سیکل (بدون قصد بارداری)',
        'cycle_base_ttc' => 'پیام پایه سیکل (اقدام به بارداری)',
        'cycle_override' => 'پیام علائم سیکل',
        'pregnancy_base' => 'پیام پایه بارداری',
        'pregnancy_trimester' => 'پیام سه‌ماهه بارداری',
        'pregnancy_week' => 'پیام هفتگی بارداری',
        'pregnancy_override' => 'پیام علائم بارداری',
        'nutrition_cycle' => 'تغذیه (سیکل)',
        'nutrition_ttc' => 'تغذیه (اقدام به بارداری)',
        'nutrition_symptom' => 'تغذیه (بر اساس علائم)',
        'nutrition_trimester' => 'تغذیه (سه‌ماهه)',
        'sleep_cycle' => 'خواب (سیکل)',
        'sleep_trimester' => 'خواب (سه‌ماهه)',
        'exercise_cycle' => 'ورزش (سیکل)',
        'exercise_trimester' => 'ورزش (سه‌ماهه)',
        'exercise_symptom' => 'ورزش (بر اساس علائم)',
        'correlation_cycle' => 'همبستگی‌ها (سیکل)',
        'correlation_pregnancy' => 'همبستگی‌ها (بارداری)',
        'correlation_common' => 'همبستگی‌های عمومی',
        'pattern' => 'الگوها',
    ];

    public function index(Request $request): View
    {
        $groups = MessageContent::query()->distinct()->orderBy('group')->pluck('group');

        $group = $request->query('group');
        $locale = $request->query('locale');
        $status = $request->query('status'); // approved | pending

        $items = MessageContent::query()
            ->when($group, fn ($q) => $q->where('group', $group))
            ->when($locale, fn ($q) => $q->where('locale', $locale))
            ->when($status === 'approved', fn ($q) => $q->where('is_approved', true))
            ->when($status === 'pending', fn ($q) => $q->where('is_approved', false))
            ->orderBy('group')->orderBy('item_key')->orderBy('locale')
            ->paginate(30)
            ->withQueryString();

        return view('admin.messages.index', [
            'items' => $items,
            'groups' => $groups,
            'labels' => self::GROUP_LABELS,
            'group' => $group,
            'locale' => $locale,
            'status' => $status,
        ]);
    }

    public function edit(MessageContent $message): View
    {
        return view('admin.messages.edit', [
            'message' => $message,
            'labels' => self::GROUP_LABELS,
        ]);
    }

    public function update(Request $request, MessageContent $message): RedirectResponse
    {
        $input = $request->input('payload', []);
        $newPayload = [];

        // Preserve the original structure: scalar fields stay strings, list
        // fields (originally arrays) are split from newline-separated textareas.
        foreach ($message->payload as $key => $original) {
            $value = $input[$key] ?? null;

            if (is_array($original)) {
                $lines = preg_split('/\r\n|\r|\n/', (string) $value);
                $newPayload[$key] = array_values(array_filter(array_map('trim', $lines), fn ($l) => $l !== ''));
            } else {
                $newPayload[$key] = is_string($value) ? $value : $original;
            }
        }

        $message->update([
            'payload' => $newPayload,
            'label' => $request->input('label', $message->label),
        ]);

        return redirect()->route('admin.messages.index', ['group' => $message->group])
            ->with('status', 'پیام به‌روزرسانی شد.');
    }

    public function approve(MessageContent $message): RedirectResponse
    {
        $message->update(['is_approved' => ! $message->is_approved]);

        return back()->with('status', $message->is_approved ? 'پیام تأیید شد.' : 'تأیید پیام لغو شد.');
    }

    public function toggle(MessageContent $message): RedirectResponse
    {
        $message->update(['is_active' => ! $message->is_active]);

        return back()->with('status', 'وضعیت پیام تغییر کرد.');
    }
}

<?php

namespace App\Http\Controllers\Admin;

use App\Enums\SubscriptionType;
use App\Enums\UserGoal;
use App\Http\Controllers\Controller;
use App\Models\User;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\View\View;

class UserController extends Controller
{
    public function index(Request $request): View
    {
        $search = trim((string) $request->query('q', ''));
        $status = $request->query('status'); // all | active | blocked

        $users = User::query()
            ->with('profile')
            ->when($search !== '', function ($query) use ($search) {
                $query->where(function ($q) use ($search) {
                    $q->where('name', 'like', "%{$search}%")
                        ->orWhere('mobile', 'like', "%{$search}%")
                        ->orWhere('email', 'like', "%{$search}%");
                });
            })
            ->when($status === 'blocked', fn ($q) => $q->whereNotNull('blocked_at'))
            ->when($status === 'active', fn ($q) => $q->whereNull('blocked_at'))
            ->latest()
            ->paginate(20)
            ->withQueryString();

        return view('admin.users.index', compact('users', 'search', 'status'));
    }

    public function show(User $user): View
    {
        $user->load('profile');

        $stats = [
            'health_logs' => $user->dailyHealthLogs()->count(),
            'reminders' => $user->reminders()->count(),
            'notifications' => $user->appNotifications()->count(),
            'cycle_calculations' => $user->cycleCalculations()->count(),
        ];

        $goals = SubscriptionType::cases();
        $userGoals = UserGoal::cases();

        return view('admin.users.show', compact('user', 'stats', 'goals', 'userGoals'));
    }

    public function update(Request $request, User $user): RedirectResponse
    {
        $validated = $request->validate([
            'name' => ['nullable', 'string', 'max:255'],
            'subscription_type' => ['required', 'in:' . implode(',', SubscriptionType::values())],
            'user_goal' => ['required', 'in:' . implode(',', UserGoal::values())],
        ]);

        $user->update(['name' => $validated['name']]);

        // Subscription & goal live on the profile; create it on demand.
        $user->profile()->updateOrCreate(
            ['user_id' => $user->id],
            [
                'subscription_type' => $validated['subscription_type'],
                'user_goal' => $validated['user_goal'],
            ]
        );

        return back()->with('status', 'اطلاعات کاربر به‌روزرسانی شد.');
    }

    public function block(User $user): RedirectResponse
    {
        $user->forceFill(['blocked_at' => now()])->save();

        // Kill active API sessions so the block takes effect immediately.
        $user->tokens()->each(fn ($token) => $token->revoke());

        return back()->with('status', 'کاربر مسدود شد.');
    }

    public function unblock(User $user): RedirectResponse
    {
        $user->forceFill(['blocked_at' => null])->save();

        return back()->with('status', 'مسدودیت کاربر برداشته شد.');
    }

    public function destroy(User $user): RedirectResponse
    {
        $user->delete();

        return redirect()->route('admin.users.index')->with('status', 'کاربر حذف شد.');
    }
}

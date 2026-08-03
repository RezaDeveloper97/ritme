<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Challenge;
use App\Models\UserChallengeCompletion;
use Carbon\Carbon;
use Illuminate\Http\Request;
use Illuminate\View\View;

/**
 * "چه کسانی چالش‌ها را انجام داده‌اند" — a read-only report over
 * user_challenge_completions, filterable by challenge, user and date range.
 */
class ChallengeCompletionController extends Controller
{
    public function index(Request $request): View
    {
        $request->validate([
            'challenge_id' => ['nullable', 'integer', 'exists:challenges,id'],
            'from' => ['nullable', 'date'],
            'to' => ['nullable', 'date'],
        ]);

        $challengeId = $request->query('challenge_id');
        $search = trim((string) $request->query('q', ''));
        $from = $request->query('from') ? Carbon::parse($request->query('from')) : null;
        $to = $request->query('to') ? Carbon::parse($request->query('to')) : null;

        // A fresh, identically-filtered query per aggregate (builders are stateful).
        $filtered = fn () => UserChallengeCompletion::query()
            ->when($challengeId, fn ($q) => $q->where('challenge_id', $challengeId))
            ->when($from, fn ($q) => $q->whereDate('completion_date', '>=', $from))
            ->when($to, fn ($q) => $q->whereDate('completion_date', '<=', $to))
            ->when($search !== '', fn ($q) => $q->whereHas(
                'user',
                fn ($u) => $u->where('name', 'like', "%{$search}%")->orWhere('mobile', 'like', "%{$search}%"),
            ));

        $completions = $filtered()
            ->with(['user:id,name,mobile', 'challenge:id,title'])
            ->orderByDesc('completion_date')
            ->orderByDesc('id')
            ->paginate(30)
            ->withQueryString();

        $stats = [
            'total' => $filtered()->count(),
            'users' => $filtered()->distinct()->count('user_id'),
            'today' => UserChallengeCompletion::query()->whereDate('completion_date', Carbon::today())->count(),
        ];

        // Per-challenge leaderboard: how often each challenge actually gets done.
        $perChallenge = $filtered()
            ->selectRaw('challenge_id, count(*) as completions, count(distinct user_id) as users')
            ->groupBy('challenge_id')
            ->orderByDesc('completions')
            ->with('challenge:id,title')
            ->get();

        return view('admin.challenges.completions', [
            'completions' => $completions,
            'stats' => $stats,
            'perChallenge' => $perChallenge,
            'challenges' => Challenge::orderBy('sort_order')->orderBy('id')->get(['id', 'title']),
            'challengeId' => $challengeId,
            'search' => $search,
            'from' => $request->query('from'),
            'to' => $request->query('to'),
        ]);
    }
}

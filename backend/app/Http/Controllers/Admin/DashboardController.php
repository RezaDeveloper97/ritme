<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Affirmation;
use App\Models\Article;
use App\Models\Challenge;
use App\Models\MessageContent;
use App\Models\TaskTemplate;
use App\Models\User;
use Illuminate\View\View;

class DashboardController extends Controller
{
    public function index(): View
    {
        $stats = [
            'users' => User::count(),
            'users_blocked' => User::whereNotNull('blocked_at')->count(),
            'articles' => Article::count(),
            'affirmations' => Affirmation::count(),
            'challenges' => Challenge::count(),
            'task_templates' => TaskTemplate::count(),
            'messages' => MessageContent::count(),
            'messages_pending' => MessageContent::where('is_approved', false)->count(),
        ];

        $recentUsers = User::latest()->take(8)->get();

        return view('admin.dashboard', compact('stats', 'recentUsers'));
    }
}

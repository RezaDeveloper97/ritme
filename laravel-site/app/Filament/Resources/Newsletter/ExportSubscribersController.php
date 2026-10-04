<?php

declare(strict_types=1);

namespace App\Filament\Resources\Newsletter;

use App\Domain\Newsletter\Actions\ExportSubscribers;
use App\Domain\Newsletter\Enums\SubscriptionStatus;
use App\Domain\Newsletter\Models\Subscriber;
use Filament\Facades\Filament;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Gate;
use Symfony\Component\HttpFoundation\StreamedResponse;

/**
 * GET {admin}/newsletter/subscribers/export?status=&search= — streamed CSV download (registered by
 * SubscriberResource::registerRoutes() inside the panel's auth middleware). A plain route instead of a Livewire
 * action download, because Livewire buffers downloads in memory. Access: SubscriberPolicy::export().
 */
final class ExportSubscribersController
{
    public function __invoke(Request $request, ExportSubscribers $export): StreamedResponse
    {
        $user = Filament::auth()->user();
        Gate::forUser($user)->authorize('export', Subscriber::class);

        $status = SubscriptionStatus::tryFrom((string) $request->query('status', ''));
        $search = $request->query('search');
        $search = is_string($search) ? mb_substr($search, 0, 191) : null;

        return response()->streamDownload(static function () use ($export, $status, $search, $user): void {
            $out = fopen('php://output', 'wb');
            if ($out === false) {
                return;
            }

            $export->handle($out, $status, $search, $user);
            fclose($out);
        }, ExportSubscribers::filename(), [
            'Content-Type' => 'text/csv; charset=UTF-8',
            'Cache-Control' => 'private, no-store',
            'X-Robots-Tag' => 'noindex, nofollow',
        ]);
    }
}

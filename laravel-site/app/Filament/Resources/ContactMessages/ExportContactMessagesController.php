<?php

declare(strict_types=1);

namespace App\Filament\Resources\ContactMessages;

use App\Domain\Contact\Actions\ExportContactMessages;
use App\Domain\Contact\Enums\ContactMessageStatus;
use App\Domain\Contact\Enums\ContactTopic;
use App\Domain\Contact\Models\ContactMessage;
use Filament\Facades\Filament;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Gate;
use Symfony\Component\HttpFoundation\StreamedResponse;

/**
 * GET {admin}/contact-messages/export?status=&topic=&search= — streamed CSV download (registered by
 * ContactMessageResource::registerRoutes() inside the panel's auth middleware). Access: ContactMessagePolicy::export().
 */
final class ExportContactMessagesController
{
    public function __invoke(Request $request, ExportContactMessages $export): StreamedResponse
    {
        $user = Filament::auth()->user();
        Gate::forUser($user)->authorize('export', ContactMessage::class);

        $status = ContactMessageStatus::tryFrom((string) $request->query('status', ''));
        $topic = ContactTopic::tryFrom((string) $request->query('topic', ''));
        $search = $request->query('search');
        $search = is_string($search) ? mb_substr($search, 0, 191) : null;

        return response()->streamDownload(static function () use ($export, $status, $topic, $search, $user): void {
            $out = fopen('php://output', 'wb');
            if ($out === false) {
                return;
            }

            $export->handle($out, $status, $topic, $search, $user);
            fclose($out);
        }, ExportContactMessages::filename(), [
            'Content-Type' => 'text/csv; charset=UTF-8',
            'Cache-Control' => 'private, no-store',
            'X-Robots-Tag' => 'noindex, nofollow',
        ]);
    }
}

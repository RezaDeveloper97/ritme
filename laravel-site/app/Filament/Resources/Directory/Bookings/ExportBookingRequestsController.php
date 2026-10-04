<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Bookings;

use App\Domain\Directory\Booking\Actions\ExportBookingRequests;
use App\Domain\Directory\Booking\Enums\BookingStatus;
use App\Domain\Directory\Booking\Models\BookingRequest;
use Filament\Facades\Filament;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Gate;
use Symfony\Component\HttpFoundation\StreamedResponse;

/**
 * GET {admin}/directory/bookings/export?status=&place=&from=&until=&search= — streamed CSV download (registered by
 * BookingRequestResource::registerRoutes() inside the panel's auth middleware). Access: BookingRequestPolicy::export().
 */
final class ExportBookingRequestsController
{
    public function __invoke(Request $request, ExportBookingRequests $export): StreamedResponse
    {
        $user = Filament::auth()->user();
        Gate::forUser($user)->authorize('export', BookingRequest::class);

        $status = BookingStatus::tryFrom((string) $request->query('status', ''));
        $place = $request->query('place');
        $placeId = is_string($place) && ctype_digit($place) ? (int) $place : null;
        $from = self::day($request->query('from'));
        $until = self::day($request->query('until'));
        $search = $request->query('search');
        $search = is_string($search) ? mb_substr($search, 0, 191) : null;

        return response()->streamDownload(static function () use ($export, $status, $placeId, $from, $until, $search, $user): void {
            $out = fopen('php://output', 'wb');
            if ($out === false) {
                return;
            }

            $export->handle($out, $status, $placeId, $from, $until, $search, $user);
            fclose($out);
        }, ExportBookingRequests::filename(), [
            'Content-Type' => 'text/csv; charset=UTF-8',
            'Cache-Control' => 'private, no-store',
            'X-Robots-Tag' => 'noindex, nofollow',
        ]);
    }

    private static function day(mixed $value): ?string
    {
        return is_string($value) && preg_match('/^\d{4}-\d{2}-\d{2}/', $value) === 1 ? substr($value, 0, 10) : null;
    }
}

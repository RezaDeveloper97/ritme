<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\JoinRequests;

use App\Domain\Directory\Join\Actions\SubmitJoinRequest;
use App\Domain\Directory\Join\Models\JoinRequest;
use App\Domain\Media\Models\Media;
use Illuminate\Contracts\Auth\Access\Gate;
use Illuminate\Contracts\Filesystem\Factory as Filesystems;
use Illuminate\Filesystem\FilesystemAdapter;
use Symfony\Component\HttpFoundation\StreamedResponse;

/**
 * L9-04b (F17): streams a not-yet-public upload (join-request photos on the private `pending` disk) to an admin.
 * Registered in AdminPanelProvider behind the panel's auth middleware (login, inactivity timeout, MFA for the
 * configured roles) and authorised here: whoever reviews join requests or manages the media library. The pending
 * disk's url() points at this route, so MediaPresenter / MediaData previews work unchanged. Images only, private +
 * no-store, nosniff, sandboxed, noindex; anything else is a 404.
 */
final class PendingMediaController
{
    public const ROUTE = 'pending-media';

    /** Same layout StoreMedia writes: `Y/m/<10 chars>/<file>` (variants may sit next to the original). */
    public const PATH_PATTERN = '[0-9]{4}/[0-9]{2}/[a-z0-9]{10}/[A-Za-z0-9][A-Za-z0-9._-]{0,250}';

    public function __invoke(string $path, Gate $gate, Filesystems $filesystems): StreamedResponse
    {
        abort_unless($gate->allows('viewAny', JoinRequest::class) || $gate->allows('viewAny', Media::class), 403);
        abort_if(str_contains($path, '..'), 404);

        $disk = $filesystems->disk(SubmitJoinRequest::PHOTO_DISK);
        abort_unless($disk instanceof FilesystemAdapter && $disk->fileExists($path), 404);

        $mime = (string) $disk->mimeType($path);
        abort_unless(in_array($mime, ['image/jpeg', 'image/png', 'image/webp', 'image/avif', 'image/gif'], true), 404);

        return $disk->response($path, basename($path), [
            'Content-Type' => $mime,
            'Cache-Control' => 'private, no-store',
            'X-Content-Type-Options' => 'nosniff',
            'Content-Security-Policy' => "default-src 'none'; sandbox",
            'X-Robots-Tag' => 'noindex, nofollow',
        ], 'inline');
    }
}

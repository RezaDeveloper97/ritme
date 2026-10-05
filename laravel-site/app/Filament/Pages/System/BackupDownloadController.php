<?php

declare(strict_types=1);

namespace App\Filament\Pages\System;

use Filament\Facades\Filament;
use Spatie\Backup\BackupDestination\BackupDestination;
use Symfony\Component\HttpFoundation\StreamedResponse;

/**
 * L10-02: streams one backup zip to a super-admin. Registered by Backups::routes() inside the panel's authenticated
 * route group (login, inactivity timeout, MFA). Only a configured backup disk and a file that the destination lists
 * as one of its backups can be fetched (no path parameters, no traversal). Every download is written to the
 * activity log, because the archive holds all personal data of the site.
 */
final class BackupDownloadController
{
    /** spatie/laravel-backup names files `<prefix>Y-m-d-H-i-s.zip`. */
    public const FILE_PATTERN = '[A-Za-z0-9][A-Za-z0-9._-]{0,120}\.zip';

    public function __invoke(string $disk, string $file): StreamedResponse
    {
        abort_unless(Backups::canAccess(), 403);
        abort_unless(in_array($disk, array_map('strval', (array) config('backup.backup.destination.disks', [])), true), 404);
        abort_unless(preg_match('#^'.self::FILE_PATTERN.'$#', $file) === 1, 404);

        $destination = BackupDestination::create($disk, (string) config('backup.backup.name'));
        $backup = $destination->backups()->first(static fn ($backup): bool => basename($backup->path()) === $file);
        abort_if($backup === null || ! $backup->exists(), 404);

        activity('backups')
            ->causedBy(Filament::auth()->user())
            ->event('downloaded')
            ->withProperties(['disk' => $disk, 'file' => $file, 'bytes' => (int) $backup->sizeInBytes()])
            ->log('backups.downloaded');

        $stream = $backup->stream();

        return new StreamedResponse(static function () use ($stream): void {
            if (is_resource($stream)) {
                fpassthru($stream);
                fclose($stream);
            }
        }, 200, [
            'Content-Type' => 'application/zip',
            'Content-Disposition' => 'attachment; filename="'.$file.'"',
            'Content-Length' => (string) (int) $backup->sizeInBytes(),
            'Cache-Control' => 'private, no-store',
            'X-Content-Type-Options' => 'nosniff',
            'X-Robots-Tag' => 'noindex, nofollow',
        ]);
    }
}

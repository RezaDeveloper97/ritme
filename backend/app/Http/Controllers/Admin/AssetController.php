<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use Illuminate\Support\Facades\Response;
use Symfony\Component\HttpFoundation\BinaryFileResponse;

/**
 * Serves the bundled CKEditor 4 build through the "/admin" prefix.
 *
 * The panel is reachable behind a single reverse-proxy rule ("/admin -> backend",
 * see routes/admin.php), so its assets can't live under /assets — and the
 * production server has no international connectivity, which rules out a CDN.
 * The editor is therefore vendored in public/assets/ckeditor and streamed from
 * here, with the path constrained to that directory and to known asset types.
 */
class AssetController extends Controller
{
    private const CKEDITOR_ROOT = 'assets/ckeditor';

    /** Extensions CKEditor actually requests, mapped to their content type. */
    private const MIME_TYPES = [
        'js' => 'application/javascript',
        'css' => 'text/css',
        'png' => 'image/png',
        'gif' => 'image/gif',
        'jpg' => 'image/jpeg',
        'jpeg' => 'image/jpeg',
        'svg' => 'image/svg+xml',
        'webp' => 'image/webp',
        'ico' => 'image/x-icon',
        'json' => 'application/json',
        'md' => 'text/plain; charset=UTF-8',
        'txt' => 'text/plain; charset=UTF-8',
    ];

    public function ckeditor(string $path): BinaryFileResponse
    {
        $root = realpath(public_path(self::CKEDITOR_ROOT));
        $file = $root === false ? false : realpath($root.DIRECTORY_SEPARATOR.$path);

        // Reject anything that escaped the editor directory (../ traversal),
        // isn't a regular file, or isn't a known asset type.
        abort_if(
            $root === false
            || $file === false
            || ! str_starts_with($file, $root.DIRECTORY_SEPARATOR)
            || ! is_file($file),
            404,
        );

        $extension = strtolower(pathinfo($file, PATHINFO_EXTENSION));
        abort_unless(isset(self::MIME_TYPES[$extension]), 404);

        return Response::file($file, [
            'Content-Type' => self::MIME_TYPES[$extension],
            'Cache-Control' => 'public, max-age=604800',
        ]);
    }
}

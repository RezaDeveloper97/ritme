<?php

declare(strict_types=1);

namespace App\Http\Controllers\Pwa;

use App\Domain\Pwa\Manifest\WebManifest;
use Illuminate\Http\Request;
use Illuminate\Http\Response;

/** `/manifest.webmanifest` (L8-01): cached JSON from PwaSettings, one-hour browser cache + ETag revalidation. */
final class ManifestController
{
    public function __invoke(Request $request, WebManifest $manifest): Response
    {
        $json = $manifest->json();

        $response = new Response($json, 200, [
            'Content-Type' => WebManifest::CONTENT_TYPE.'; charset=UTF-8',
            'Cache-Control' => 'public, max-age=3600',
        ]);
        $response->setEtag(substr(hash('xxh128', $json), 0, 16));
        $response->isNotModified($request);

        return $response;
    }
}

<?php

declare(strict_types=1);

namespace App\Http\Controllers\Seo;

use App\Domain\Seo\Sitemap\RobotsTxt;
use Illuminate\Http\Response;

/** `/robots.txt` (dynamic, environment-aware; see RobotsTxt). */
final class RobotsTxtController
{
    public function __invoke(RobotsTxt $robots): Response
    {
        return new Response($robots->render(), 200, [
            'Content-Type' => 'text/plain; charset=UTF-8',
            'Cache-Control' => 'public, max-age=3600',
        ]);
    }
}

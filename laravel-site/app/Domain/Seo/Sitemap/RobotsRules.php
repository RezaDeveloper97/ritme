<?php

declare(strict_types=1);

namespace App\Domain\Seo\Sitemap;

/**
 * The crawl rules of the production robots.txt (without the `Sitemap:` line, which RobotsTxt appends).
 * DefaultRobotsRules is bound now; L7-04 rebinds this to the admin-edited rules (its observer must bump `sitemap`).
 */
interface RobotsRules
{
    public function rules(): string;
}

<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Crawl;

/**
 * One in-process GET: status (0 = the render threw), Location of a redirect, content type, body (HTML only), time
 * taken and the matched route (name + parameters) for fix links.
 */
final readonly class FetchResult
{
    /**
     * @param  array<string, string>  $routeParameters
     */
    public function __construct(
        public string $path,
        public int $status,
        public string $contentType = '',
        public ?string $location = null,
        public string $html = '',
        public int $milliseconds = 0,
        public ?string $routeName = null,
        public array $routeParameters = [],
        public ?string $error = null,
    ) {}

    public function isHtml(): bool
    {
        return $this->status === 200 && str_contains(strtolower($this->contentType), 'text/html');
    }

    public function isRedirect(): bool
    {
        return $this->status >= 300 && $this->status < 400 && $this->location !== null;
    }

    public function isBroken(): bool
    {
        return $this->status === 0 || $this->status >= 400;
    }
}

<?php

declare(strict_types=1);

namespace App\Domain\Media\Support;

use finfo;

/**
 * Detects the real mime type from file contents (never trusts the client name or header). SVGs are often reported
 * as text/plain or text/xml, so those are re-checked for an <svg> root.
 */
final class MimeSniffer
{
    public function sniff(string $path): string
    {
        $mime = (string) (new finfo(FILEINFO_MIME_TYPE))->file($path);

        if (in_array($mime, ['text/plain', 'text/xml', 'application/xml', 'text/html'], true) && $this->looksLikeSvg($path)) {
            return 'image/svg+xml';
        }

        return $mime;
    }

    private function looksLikeSvg(string $path): bool
    {
        $head = (string) file_get_contents($path, false, null, 0, 4096);

        return preg_match('/<svg[\s>]/i', $head) === 1;
    }
}

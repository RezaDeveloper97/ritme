<?php

declare(strict_types=1);

namespace App\Support\Html;

/**
 * Adds `rel="noopener"` to links that leave the site (absolute http/https URL on a host that is not ours); existing
 * rel tokens (nofollow, sponsored …) are kept. Internal, relative, mailto: and tel: links are left untouched.
 */
final class ExternalLinks
{
    /**
     * @param  list<string>  $ownHosts
     */
    public static function apply(string $html, array $ownHosts): string
    {
        if (trim($html) === '' || stripos($html, '<a') === false) {
            return $html;
        }

        $own = array_map(strtolower(...), $ownHosts);
        $fragment = HtmlFragment::load($html);

        foreach ($fragment->elements('a') as $link) {
            $parts = parse_url(trim($link->getAttribute('href')));
            if ($parts === false || ! isset($parts['host'])) {
                continue;
            }

            $scheme = strtolower($parts['scheme'] ?? 'https'); // protocol-relative //host counts as external
            if (! in_array($scheme, ['http', 'https'], true) || in_array(strtolower($parts['host']), $own, true)) {
                continue;
            }

            $tokens = preg_split('/\s+/', strtolower(trim($link->getAttribute('rel'))), -1, PREG_SPLIT_NO_EMPTY) ?: [];
            if (! in_array('noopener', $tokens, true)) {
                $tokens[] = 'noopener';
            }
            $link->setAttribute('rel', implode(' ', array_values(array_unique($tokens))));
        }

        return $fragment->html();
    }
}

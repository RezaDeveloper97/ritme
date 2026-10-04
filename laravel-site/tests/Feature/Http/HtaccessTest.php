<?php

declare(strict_types=1);

function htaccess(): string
{
    return (string) file_get_contents(public_path('.htaccess'));
}

it('routes everything that is not a file or folder to the front controller', function (): void {
    expect(htaccess())
        ->toContain('RewriteRule ^ index.php [L]')
        ->toContain('RewriteCond %{REQUEST_FILENAME} !-f')
        ->not->toContain('RewriteBase')       // works as public_html or in any document root
        ->not->toContain('R=301]'."\n".'    RewriteCond %{REQUEST_FILENAME} !-d'); // no trailing-slash hop (CanonicalizeUrl)
});

it('caches /build and /media for a year as immutable, sw.js and the manifest briefly', function (): void {
    expect(htaccess())
        ->toContain('RewriteRule ^(build|media)/ - [E=RT_IMMUTABLE:1]')
        ->toContain('Header set Cache-Control "public, max-age=31536000, immutable" env=RT_IMMUTABLE')
        ->toContain('<Files "sw.js">')
        ->toContain('Header set Cache-Control "public, max-age=0, must-revalidate"')
        ->toContain('<Files "manifest.webmanifest">');
});

it('compresses with brotli when present and gzip otherwise, varying on Accept-Encoding', function (): void {
    expect(htaccess())
        ->toContain('<IfModule mod_brotli.c>')
        ->toContain('AddOutputFilterByType BROTLI_COMPRESS;DEFLATE text/html')
        ->toContain('AddOutputFilterByType DEFLATE text/html')
        ->toContain('Header merge Vary Accept-Encoding');
});

it('denies dotfiles but keeps .well-known reachable, and declares modern MIME types', function (): void {
    expect(htaccess())
        ->toContain('RewriteRule (^|/)\.(?!well-known(/|$)) - [F,L]')
        ->toContain('Require all denied')
        ->toContain('AddType font/woff2 .woff2')
        ->toContain('AddType image/avif .avif')
        ->toContain('AddType image/webp .webp')
        ->toContain('AddType application/manifest+json .webmanifest');
});

it('ships the https redirect as a commented toggle', function (): void {
    expect(htaccess())->toContain('# RewriteRule ^ https://%{HTTP_HOST}%{REQUEST_URI} [L,R=301]');
});

it('explains every directive on the comment line above it', function (): void {
    $lines = preg_split('/\R/', htaccess()) ?: [];
    $previous = '';
    $uncommented = [];

    foreach ($lines as $number => $line) {
        $trimmed = trim($line);

        $isDirective = $trimmed !== '' && ! str_starts_with($trimmed, '#') && ! str_starts_with($trimmed, '<');
        $prevExplains = str_starts_with($previous, '#') || str_starts_with($previous, 'RewriteCond');

        if ($isDirective && ! $prevExplains && ! str_starts_with($trimmed, 'RewriteCond')) {
            $uncommented[] = ($number + 1).': '.$trimmed;
        }

        if ($trimmed !== '') {
            $previous = $trimmed;
        }
    }

    expect($uncommented)->toBe([]);
});

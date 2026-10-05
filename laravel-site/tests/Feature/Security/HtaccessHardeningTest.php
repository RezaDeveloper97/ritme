<?php

declare(strict_types=1);

/*
 * L9-04: public/.htaccess defence in depth (verified on Apache 2.4 with mod_rewrite + mod_headers, docs/SECURITY.md).
 */

function hardenedHtaccess(): string
{
    return (string) file_get_contents(public_path('.htaccess'));
}

it('refuses project folders and files that exist on disk if the project is unpacked into the document root', function (): void {
    expect(hardenedHtaccess())
        ->toContain("RewriteCond %{REQUEST_FILENAME} -f [OR]\n    RewriteCond %{REQUEST_FILENAME} -d\n    RewriteRule ^(app|bootstrap|config|database|lang|node_modules|resources|routes|storage|tests|tools|vendor|design|docs|tasks)(/|$) - [F,L]")
        ->toContain('RewriteRule ^(artisan|composer\.(json|lock)|package(-lock)?\.json|phpunit\.xml');
});

it('never executes PHP (or other scripts) from the public folder except the front controller', function (): void {
    $rule = '/RewriteRule \^\(\?!index\\\\\.php\$\)\.\*\\\\\.\(php\[0-9\]\?\|phtml\|phar[^)]*\)\$ - \[F,L,NC\]/';

    expect(preg_match($rule, hardenedHtaccess()))->toBe(1);

    // The rule must sit before the front-controller rewrite.
    expect(strpos(hardenedHtaccess(), 'phtml|phar'))->toBeLessThan(strpos(hardenedHtaccess(), 'RewriteRule ^ index.php [L]'));
});

it('sends nosniff on static files and sandboxes uploaded SVGs', function (): void {
    expect(hardenedHtaccess())
        ->toContain('Header always set X-Content-Type-Options "nosniff"')
        ->toContain('RewriteRule ^media/.+\.svgz?$ - [E=RT_MEDIA_SVG:1,NC]')
        ->toContain('sandbox" env=RT_MEDIA_SVG');
});

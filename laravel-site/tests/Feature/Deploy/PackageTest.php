<?php

declare(strict_types=1);

/**
 * L10-01: the cPanel package pieces that are not PHP classes — build script, layout templates, production .env
 * example. The package itself is built and booted by `bash deploy/build-cpanel.sh` (see docs/DEPLOY-CPANEL.md).
 */
function deployFile(string $path): string
{
    return (string) file_get_contents(base_path($path));
}

it('builds without dev dependencies, with the strict critical CSS build, and strips dev files', function (): void {
    $script = deployFile('deploy/build-cpanel.sh');

    expect($script)
        ->toContain('composer install --no-dev --optimize-autoloader --classmap-authoritative')
        ->toContain('CRITICAL_STRICT=1')
        ->toContain('npm ci')
        ->toContain('php artisan filament:assets')
        ->toContain('public/build/critical/manifest.json')
        ->toContain('public/sw.js')
        ->toContain('--dry-run')
        ->toContain('--layout=public_html')
        ->toContain('shasum -a 256')
        ->not->toMatch('/\b(rsync|scp|ssh|curl|ftp)\b/');  // packaging only — never uploads

    foreach (['tests', 'design', 'tasks', 'docs', 'tools', 'node_modules', '.env', 'phpunit.xml', 'resources/js'] as $excluded) {
        expect($script)->toMatch('/^EXCLUDES="[^"]*(?<=[\s"])'.preg_quote($excluded, '/').'(?=[\s"])/m');
    }
});

it('serves the fallback layout from public/ only and refuses everything without mod_rewrite', function (): void {
    $root = deployFile('deploy/cpanel/root.htaccess');

    expect($root)
        ->toContain('RewriteRule ^(.*)$ public/$1 [L]')
        ->toContain('RewriteRule ^\.well-known/ - [L]')
        ->toContain('<IfModule !mod_rewrite.c>')
        ->toContain('Require all denied')
        ->toContain('DirectoryIndex index.php');

    expect(deployFile('public/.htaccess'))
        ->toContain('RewriteCond %{THE_REQUEST} ^[A-Z]+\s/public(/|\s|\?)')
        ->toContain('DirectoryIndex index.php');
});

it('points the public_html front controller and the CLI at the same public folder', function (): void {
    $index = deployFile('deploy/cpanel/index.public_html.php');
    expect($index)
        ->toContain("dirname(__DIR__).'/__APP_DIR__'")
        ->toContain("\$appRoot.'/storage/framework/maintenance.php'")
        ->toContain("\$appRoot.'/vendor/autoload.php'")
        ->toContain('$app->usePublicPath(__DIR__);');

    expect(deployFile('deploy/cpanel/public-path.php'))->toContain("dirname(__DIR__, 2).'/__DOC_ROOT__'")
        ->and(deployFile('bootstrap/app.php'))->toContain("is_file(__DIR__.'/public-path.php')")->toContain('usePublicPath');
});

it('ships a production .env example for shared hosting', function (): void {
    $env = deployFile('.env.cpanel.example');

    foreach ([
        'APP_ENV=production', 'APP_DEBUG=false', 'APP_KEY=', 'APP_CANONICAL_REDIRECT=true', 'DB_CONNECTION=mysql',
        'CACHE_STORE=file', 'SESSION_DRIVER=file', 'QUEUE_CONNECTION=database', 'MEDIA_PUBLIC_ROOT', 'MAIL_MAILER=smtp',
        'TRUSTED_PROXIES=', 'ADMIN_MFA_ROLES=super-admin,shop-manager,directory-manager,support',
    ] as $line) {
        expect($env)->toContain($line);
    }
    expect($env)->not->toMatch('/^APP_KEY=base64:/m');
});

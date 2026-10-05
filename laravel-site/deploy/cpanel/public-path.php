<?php

declare(strict_types=1);

// cPanel layout "public_html" (L10-01): the document root that holds index.php, next to the app folder. Read by
// bootstrap/app.php so cron, queue and artisan share public_path() with the web. Rewrite it with
// `php artisan app:install --public-path=<folder>` if the public files live somewhere else.
return dirname(__DIR__, 2).'/__DOC_ROOT__';

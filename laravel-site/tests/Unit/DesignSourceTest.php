<?php

declare(strict_types=1);

it('keeps the design export as the fidelity source', function (): void {
    $root = dirname(__DIR__, 2).'/design/html';

    expect($root.'/index.html')->toBeFile()
        ->and($root.'/assets/ritme.css')->toBeFile()
        ->and(glob($root.'/*.html'))->toHaveCount(29);
});

<?php

declare(strict_types=1);

use App\Domain\Media\Support\FocalCrop;

it('crops a landscape source to the target ratio around the centre', function (): void {
    expect(FocalCrop::box(4000, 3000, 1200, 630))
        ->toBe(['x' => 0, 'y' => 450, 'width' => 4000, 'height' => 2100, 'outWidth' => 1200, 'outHeight' => 630]);
});

it('follows the focal point and clamps the box to the image', function (): void {
    expect(FocalCrop::box(2000, 1000, 320, 320, 1.0, 0.5))->toMatchArray(['x' => 1000, 'y' => 0, 'width' => 1000, 'height' => 1000])
        ->and(FocalCrop::box(2000, 1000, 320, 320, 0.0, 0.5))->toMatchArray(['x' => 0])
        ->and(FocalCrop::box(2000, 1000, 320, 320, 0.6, 0.5))->toMatchArray(['x' => 700])
        ->and(FocalCrop::box(1000, 2000, 320, 320, 0.5, 0.1))->toMatchArray(['x' => 0, 'y' => 0])
        ->and(FocalCrop::box(1000, 2000, 320, 320, 0.5, 9.0))->toMatchArray(['y' => 1000]);
});

it('never upscales: a small source keeps the crop size', function (): void {
    expect(FocalCrop::box(600, 400, 1200, 630))->toMatchArray(['width' => 600, 'height' => 315, 'outWidth' => 600, 'outHeight' => 315])
        ->and(FocalCrop::box(200, 200, 320, 320))->toMatchArray(['outWidth' => 200, 'outHeight' => 200]);
});

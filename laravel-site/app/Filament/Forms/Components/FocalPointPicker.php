<?php

declare(strict_types=1);

namespace App\Filament\Forms\Components;

use Closure;
use Filament\Forms\Components\Field;
use Illuminate\Support\Facades\View;

/**
 * Click on the image to set the focal point (the subject that focal crops — thumb, OG, square — keep in frame).
 * State: ['x' => 0..1, 'y' => 0..1] measured from the top-left corner of the image. Keyboard: arrow keys move the
 * point in 5 % steps once the image has focus.
 */
final class FocalPointPicker extends Field
{
    public const VIEW_NAMESPACE = 'ritme-admin-forms';

    protected string $view = self::VIEW_NAMESPACE.'::focal-point-picker';

    protected string|Closure|null $imageUrl = null;

    protected function setUp(): void
    {
        parent::setUp();

        self::registerViews();

        $this->label('نقطه کانونی');
        $this->default(['x' => 0.5, 'y' => 0.5]);
        $this->helperText('روی بخش اصلی تصویر کلیک کنید؛ برش‌های بندانگشتی و اشتراک‌گذاری حول این نقطه ساخته می‌شوند.');
    }

    public static function registerViews(): void
    {
        View::replaceNamespace(self::VIEW_NAMESPACE, __DIR__.'/views');
    }

    public function imageUrl(string|Closure|null $url): static
    {
        $this->imageUrl = $url;

        return $this;
    }

    public function getImageUrl(): ?string
    {
        $url = $this->evaluate($this->imageUrl);

        return is_string($url) && $url !== '' ? $url : null;
    }
}

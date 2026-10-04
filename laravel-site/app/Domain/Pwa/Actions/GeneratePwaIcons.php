<?php

declare(strict_types=1);

namespace App\Domain\Pwa\Actions;

use App\Domain\Media\Models\Media;
use App\Domain\Media\Support\MemoryLimit;
use App\Domain\Pwa\Support\PwaIconFiles;
use GdImage;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Filesystem\Factory as Filesystems;
use Intervention\Image\ImageManager;
use Intervention\Image\Interfaces\ImageInterface;
use InvalidArgumentException;

/**
 * Builds the PWA icon set from an uploaded square logo with the media pipeline's image driver: transparent `any`
 * icons (96/192/512), full-bleed maskable icons (logo inside the 80 % safe zone on the background colour), an opaque
 * 180 px apple-touch-icon, a white-on-transparent monochrome icon (only when the logo has transparency and the
 * driver is GD), favicon-32.png and a favicon.ico. Written to a versioned directory on the media disk (immutable
 * cache safe); older generated sets are removed. Returns the directory.
 */
final class GeneratePwaIcons
{
    /** Logo share of the canvas: maskable keeps it inside the safe circle, apple-touch nearly full-bleed. */
    private const MASKABLE_SCALE = 0.6;

    private const APPLE_SCALE = 0.84;

    public function __construct(
        private readonly ImageManager $images,
        private readonly Filesystems $filesystems,
        private readonly Config $config,
    ) {}

    public function handle(Media $media, string $backgroundColor): string
    {
        if (! PwaIconFiles::usableSource($media)) {
            throw new InvalidArgumentException('PWA icons need a raster logo of at least '.PwaIconFiles::MIN_SOURCE_SIZE.' px.');
        }

        MemoryLimit::raise((string) $this->config->get('media.memory_limit', '512M'));

        $disk = $this->filesystems->disk($media->disk);
        $directory = PwaIconFiles::generatedDirectory($media, $backgroundColor);

        // One decode of the (up to 2560 px) original; every size is derived from this 512 px master.
        $master = (string) $this->images->read((string) $disk->get($media->path()))
            ->scaleDown(PwaIconFiles::MIN_SOURCE_SIZE, PwaIconFiles::MIN_SOURCE_SIZE)
            ->toPng();

        $files = [];
        foreach (PwaIconFiles::ICONS as $name => [$size, $purpose]) {
            $icon = match ($purpose) {
                'maskable' => $this->compose($master, $size, self::MASKABLE_SCALE, $backgroundColor),
                'monochrome' => $this->monochrome($this->compose($master, $size, 1.0, null)),
                default => $this->compose($master, $size, 1.0, null),
            };
            if ($icon !== null) {
                $files[$name] = (string) $icon->toPng();
            }
        }
        $files[PwaIconFiles::SHORTCUT_ICON] = (string) $this->compose($master, PwaIconFiles::SHORTCUT_SIZE, 1.0, null)->toPng();
        $files[PwaIconFiles::APPLE_TOUCH_ICON] = (string) $this->compose($master, PwaIconFiles::APPLE_TOUCH_SIZE, self::APPLE_SCALE, $backgroundColor)->toPng();
        $files[PwaIconFiles::FAVICON_PNG] = (string) $this->compose($master, 32, 1.0, null)->toPng();

        foreach ($files as $name => $bytes) {
            $disk->put("{$directory}/{$name}.png", $bytes);
        }
        $disk->put("{$directory}/favicon.ico", $this->ico($master));

        foreach ($disk->directories(PwaIconFiles::GENERATED_ROOT) as $old) {
            if ($old !== $directory) {
                $disk->deleteDirectory($old);
            }
        }

        return $directory;
    }

    /** The logo scaled to `scale` of a square canvas, centred; transparent canvas when $background is null. */
    private function compose(string $master, int $size, float $scale, ?string $background): ImageInterface
    {
        $canvas = $this->images->create($size, $size);
        if ($background !== null) {
            $canvas->fill($background);
        }
        $box = max(1, (int) round($size * $scale));
        $logo = $this->images->read($master)->scale($box, $box);

        return $canvas->place($logo, 'center');
    }

    /** White silhouette from the alpha channel; null when the logo is fully opaque or the driver is not GD. */
    private function monochrome(ImageInterface $icon): ?ImageInterface
    {
        $gd = $icon->core()->native();
        if (! $gd instanceof GdImage) {
            return null;
        }

        imagealphablending($gd, false);
        imagesavealpha($gd, true);
        $transparent = false;
        $width = imagesx($gd);
        $height = imagesy($gd);
        $colors = [];
        for ($y = 0; $y < $height; $y++) {
            for ($x = 0; $x < $width; $x++) {
                $alpha = (imagecolorat($gd, $x, $y) >> 24) & 0x7F;
                $transparent = $transparent || $alpha > 0;
                $colors[$alpha] ??= (int) imagecolorallocatealpha($gd, 255, 255, 255, $alpha);
                imagesetpixel($gd, $x, $y, $colors[$alpha]);
            }
        }

        return $transparent ? $icon : null;
    }

    /** favicon.ico with PNG-encoded entries (supported by every current browser). */
    private function ico(string $master): string
    {
        $images = [];
        foreach (PwaIconFiles::ICO_SIZES as $size) {
            $images[$size] = (string) $this->compose($master, $size, 1.0, null)->toPng();
        }

        $header = pack('vvv', 0, 1, count($images));
        $offset = 6 + 16 * count($images);
        $entries = '';
        foreach ($images as $size => $png) {
            $entries .= pack('CCCCvvVV', $size, $size, 0, 0, 1, 32, strlen($png), $offset);
            $offset += strlen($png);
        }

        return $header.$entries.implode('', $images);
    }
}

<?php

declare(strict_types=1);

use App\Domain\Media\Actions\StoreMedia;
use App\Domain\Media\Data\MediaUpload;
use App\Domain\Media\Models\Media;
use App\Domain\Pwa\Manifest\WebManifest;
use App\Domain\Pwa\Support\PwaIconFiles;
use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Filament\Auth\AdminRole;
use App\Filament\Pages\Settings\PwaSettings;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Facades\Filament;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Queue;
use Illuminate\Support\Facades\Storage;
use Livewire\Livewire;

/** A square PNG logo with a transparent background (a filled disc). */
function pwaLogoPng(int $size = 600, bool $transparent = true): string
{
    $image = imagecreatetruecolor($size, $size);
    imagesavealpha($image, true);
    imagealphablending($image, false);
    imagefill($image, 0, 0, (int) imagecolorallocatealpha($image, 255, 255, 255, $transparent ? 127 : 0));
    imagealphablending($image, true);
    imagefilledellipse($image, intdiv($size, 2), intdiv($size, 2), $size - 20, $size - 20, (int) imagecolorallocate($image, 110, 84, 240));
    $path = tempnam(sys_get_temp_dir(), 'pwa-logo').'.png';
    imagepng($image, $path);

    return $path;
}

function storePwaLogo(int $size = 600): Media
{
    return app(StoreMedia::class)->handle(new MediaUpload(pwaLogoPng($size), 'logo.png', alt: 'لوگوی ریتمی'));
}

/**
 * @return array<string, mixed>
 */
function fetchManifest(): array
{
    $response = test()->get('/manifest.webmanifest')->assertOk();

    return (array) json_decode((string) $response->getContent(), true, 512, JSON_THROW_ON_ERROR);
}

/** Path in public/ of a root-relative URL (query dropped). */
function publicFileOf(string $url): string
{
    return public_path(ltrim((string) parse_url($url, PHP_URL_PATH), '/'));
}

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
});

it('serves a Persian RTL manifest with the install fields, the right content type and no cookies', function (): void {
    $response = $this->get('/manifest.webmanifest')->assertOk();

    expect($response->headers->get('Content-Type'))->toStartWith('application/manifest+json')
        ->and($response->headers->get('Cache-Control'))->toContain('max-age=3600')
        ->and($response->headers->get('ETag'))->not->toBeNull()
        ->and($response->headers->getCookies())->toBe([]);

    $manifest = fetchManifest();
    expect($manifest)->toMatchArray([
        'id' => '/',
        'lang' => 'fa',
        'dir' => 'rtl',
        'start_url' => '/?source=pwa',
        'scope' => '/',
        'display' => 'standalone',
        'theme_color' => '#17112B',
        'background_color' => '#FFFFFF',
        'short_name' => 'ریتمی',
        'prefer_related_applications' => false,
    ])->and($manifest['name'])->toBeString()->not->toBe('')
        ->and($manifest['description'])->toBeString()->not->toBe('');

    $purposes = collect($manifest['icons'])->map(fn (array $icon): string => $icon['purpose'].'@'.$icon['sizes'])->all();
    expect($purposes)->toContain('any@192x192', 'any@512x512', 'maskable@512x512', 'monochrome@512x512');

    expect(collect($manifest['screenshots'])->pluck('form_factor')->all())->toBe(['narrow', 'wide'])
        ->and(collect($manifest['shortcuts'])->pluck('name')->all())->toBe(['ابزارها', 'مجله', 'فروشگاه'])
        ->and(collect($manifest['shortcuts'])->pluck('url')->all())->toBe(['/tools', '/blog', '/shop']);
});

it('answers 304 when the ETag matches', function (): void {
    $etag = (string) $this->get('/manifest.webmanifest')->headers->get('ETag');

    $this->get('/manifest.webmanifest', ['If-None-Match' => $etag])->assertStatus(304);
});

it('points only at committed default files with the declared pixel sizes', function (): void {
    $manifest = fetchManifest();

    $images = [...$manifest['icons'], ...$manifest['screenshots'], ...array_merge(...array_column($manifest['shortcuts'], 'icons'))];
    foreach ($images as $image) {
        $file = publicFileOf($image['src']);
        expect($file)->toBeFile();
        $info = getimagesize($file);
        expect($info)->not->toBeFalse()
            ->and("{$info[0]}x{$info[1]}")->toBe($image['sizes'])
            ->and($info['mime'])->toBe($image['type']);
    }

    // Screenshots: Chrome's richer install UI wants 320–3840 px and an aspect ratio of at most 2.3.
    foreach ($manifest['screenshots'] as $shot) {
        [$w, $h] = array_map(intval(...), explode('x', $shot['sizes']));
        expect(max($w, $h) / min($w, $h))->toBeLessThanOrEqual(2.3);
    }

    $apple = getimagesize(public_path('icons/apple-touch-icon.png'));
    expect([$apple[0] ?? 0, $apple[1] ?? 0])->toBe([180, 180])
        ->and(public_path('icons/favicon.svg'))->toBeFile()
        ->and(public_path('icons/mask-icon.svg'))->toBeFile();

    // favicon.ico: ICO header with a 32 px entry.
    $ico = (string) file_get_contents(public_path('favicon.ico'));
    $header = unpack('vreserved/vtype/vcount', substr($ico, 0, 6));
    expect($header['type'])->toBe(1)->and($header['count'])->toBeGreaterThan(0);
    $sizes = [];
    for ($i = 0; $i < $header['count']; $i++) {
        $sizes[] = ord($ico[6 + 16 * $i]);
    }
    expect($sizes)->toContain(32);
});

it('caches the manifest (no queries when warm) and busts it when the PWA settings change', function (): void {
    fetchManifest(); // warm

    DB::enableQueryLog();
    $this->get('/manifest.webmanifest')->assertOk();
    expect(DB::getQueryLog())->toBe([]);
    DB::disableQueryLog();

    app(UpdateSettings::class)->handle(SettingGroup::Pwa, ['name' => 'ریتمی — همراه سلامت زنان', 'theme_color' => '#6E54F0']);

    expect(fetchManifest())->toMatchArray(['name' => 'ریتمی — همراه سلامت زنان', 'theme_color' => '#6E54F0']);
});

it('prints the install head tags on public pages', function (): void {
    $html = (string) $this->get('/about')->assertOk()->getContent();

    expect($html)->toContain('<html lang="fa" dir="rtl">')
        ->toContain('<link rel="manifest" href="/manifest.webmanifest">')
        ->toContain('<meta name="theme-color" media="(prefers-color-scheme: light)" content="#17112B">')
        ->toContain('<meta name="theme-color" media="(prefers-color-scheme: dark)" content="'.WebManifest::DARK_THEME_COLOR.'">')
        ->toContain('<meta name="application-name" content="ریتمی">')
        ->toContain('<meta name="apple-mobile-web-app-capable" content="yes">')
        ->toContain('<meta name="apple-mobile-web-app-title" content="ریتمی">')
        ->toContain('<link rel="icon" href="/favicon.ico" sizes="32x32">')
        ->toMatch('~<link rel="icon" href="/icons/favicon\.svg\?v=[0-9a-f]{8}" type="image/svg\+xml">~')
        ->toMatch('~<link rel="apple-touch-icon" href="/icons/apple-touch-icon\.png\?v=[0-9a-f]{8}" sizes="180x180">~')
        ->toMatch('~<link rel="mask-icon" href="/icons/mask-icon\.svg\?v=[0-9a-f]{8}" color="#17112B">~');
});

it('generates the icon set from an uploaded logo and serves it in the manifest and head', function (): void {
    Storage::fake('public');
    Queue::fake();
    $media = storePwaLogo();

    app(UpdateSettings::class)->handle(SettingGroup::Pwa, ['icon_media_id' => $media->id]);

    $manifest = fetchManifest();
    $directory = PwaIconFiles::generatedDirectory($media, '#FFFFFF');
    $disk = Storage::disk('public');

    $purposes = collect($manifest['icons'])->map(fn (array $icon): string => $icon['purpose'].'@'.$icon['sizes'])->all();
    expect($purposes)->toBe(['any@192x192', 'any@512x512', 'maskable@192x192', 'maskable@512x512', 'monochrome@512x512']);

    foreach ($manifest['icons'] as $icon) {
        expect($icon['src'])->toContain($directory);
        $path = $directory.'/'.basename((string) parse_url($icon['src'], PHP_URL_PATH));
        $info = getimagesizefromstring((string) $disk->get($path));
        expect("{$info[0]}x{$info[1]}")->toBe($icon['sizes']);
    }

    // Maskable + apple-touch are opaque (background colour), `any` keeps the transparency.
    $corner = static function (string $file) use ($disk, $directory): int {
        $image = imagecreatefromstring((string) $disk->get("{$directory}/{$file}"));

        return (imagecolorat($image, 0, 0) >> 24) & 0x7F;
    };
    expect($corner('maskable-512.png'))->toBe(0)
        ->and($corner('apple-touch-icon.png'))->toBe(0)
        ->and($corner('icon-512.png'))->toBe(127)
        ->and($disk->exists("{$directory}/favicon.ico"))->toBeTrue();

    $html = (string) $this->get('/about')->assertOk()->getContent();
    expect($html)->toContain($directory.'/apple-touch-icon.png')
        ->toContain($directory.'/favicon-32.png')
        ->not->toContain('/icons/favicon.svg');
});

it('falls back to the default icons when the logo is too small', function (): void {
    Storage::fake('public');
    Queue::fake();
    $media = storePwaLogo(256);

    app(UpdateSettings::class)->handle(SettingGroup::Pwa, ['icon_media_id' => $media->id]);

    expect(fetchManifest()['icons'][0]['src'])->toStartWith('/icons/icon-192.png');
});

it('lets an editor save the PWA settings page, which generates the icons', function (): void {
    Storage::fake('public');
    Queue::fake();
    $this->seed(AdminRolesSeeder::class);
    Filament::setCurrentPanel('admin');
    $editor = User::factory()->create();
    $editor->assignRole(AdminRole::Editor->value);
    $media = storePwaLogo();

    $this->actingAs($editor);
    Livewire::test(PwaSettings::class)
        ->fillForm(['name' => 'ریتمی — همراه سلامت زنان', 'icon_media_id' => $media->id])
        ->call('save')
        ->assertHasNoFormErrors()
        ->assertNotified();

    expect(Storage::disk('public')->exists(PwaIconFiles::generatedDirectory($media, '#FFFFFF').'/icon-512.png'))->toBeTrue()
        ->and(fetchManifest()['name'])->toBe('ریتمی — همراه سلامت زنان');

    Livewire::test(PwaSettings::class)
        ->fillForm(['theme_color' => 'purple', 'icon_media_id' => storePwaLogo(256)->id])
        ->call('save')
        ->assertHasFormErrors(['theme_color', 'icon_media_id']);
});

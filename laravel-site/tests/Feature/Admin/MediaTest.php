<?php

declare(strict_types=1);

use App\Domain\Blog\Models\Post;
use App\Domain\Media\Actions\FindMediaUsages;
use App\Domain\Media\Actions\StoreMedia;
use App\Domain\Media\Data\MediaUpload;
use App\Domain\Media\Models\Media;
use App\Filament\Auth\AdminRole;
use App\Filament\Forms\Components\MediaPicker;
use App\Filament\Resources\Media\MediaResource;
use App\Filament\Resources\Media\Pages\EditMedia;
use App\Filament\Resources\Media\Pages\ListMedia;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Actions\Concerns\InteractsWithActions;
use Filament\Actions\Contracts\HasActions;
use Filament\Actions\Testing\TestAction;
use Filament\Schemas\Concerns\InteractsWithSchemas;
use Filament\Schemas\Contracts\HasSchemas;
use Filament\Schemas\Schema;
use Illuminate\Http\UploadedFile;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Storage;
use Livewire\Component;
use Livewire\Livewire;
use Tests\Feature\Media\MediaFixtures;

function mediaAdmin(?AdminRole $role = AdminRole::Editor): User
{
    $user = User::factory()->create();
    if ($role !== null) {
        $user->assignRole($role->value);
    }

    return $user;
}

function libraryItem(int $width = 1600, int $height = 1000, ?string $alt = 'منظره'): Media
{
    return app(StoreMedia::class)->handle(new MediaUpload(MediaFixtures::jpeg($width, $height), "photo-{$width}.jpg", alt: $alt));
}

function fixtureUpload(int $width, int $height, string $name): UploadedFile
{
    return UploadedFile::fake()->createWithContent($name, (string) file_get_contents(MediaFixtures::jpeg($width, $height, name: pathinfo($name, PATHINFO_FILENAME))));
}

beforeEach(function (): void {
    Storage::fake('public');
    $this->seed([SettingsSeeder::class, AdminRolesSeeder::class]);
});

it('lets editors and other content roles open the library and denies support, guests and role-less users', function (): void {
    $this->get(MediaResource::getUrl('index'))->assertRedirect('/admin/login');

    foreach ([AdminRole::Editor, AdminRole::SeoManager, AdminRole::ShopManager, AdminRole::DirectoryManager] as $role) {
        $this->actingAs(mediaAdmin($role))->get(MediaResource::getUrl('index'))->assertOk();
    }

    $this->actingAs(mediaAdmin(AdminRole::Support))->get(MediaResource::getUrl('index'))->assertForbidden();
    $this->actingAs(mediaAdmin(null))->get(MediaResource::getUrl('index'))->assertForbidden();

    $media = libraryItem();
    $support = mediaAdmin(AdminRole::Support);
    expect($support->can('update', $media))->toBeFalse()
        ->and($support->can('delete', $media))->toBeFalse()
        ->and(mediaAdmin()->can('update', $media))->toBeTrue();
});

it('uploads several files from the admin, creates the variants and shows them in the grid with savings', function (): void {
    $this->actingAs(mediaAdmin());

    Livewire::test(ListMedia::class)
        ->callAction('upload', data: [
            'files' => [fixtureUpload(1600, 1000, 'cover.jpg'), fixtureUpload(900, 700, 'detail.jpg')],
            'alt' => '',
        ])
        ->assertHasNoActionErrors()
        ->assertNotified();

    $items = Media::query()->orderBy('id')->get();
    expect($items)->toHaveCount(2);

    $cover = $items->first();
    expect($cover->optimized_at)->not->toBeNull()
        ->and($cover->variants)->toHaveKeys(['mobile_480', 'mobile_768', 'desktop_1280', 'thumb', 'og'])
        ->and($cover->alt)->toBeNull()
        ->and($cover->uploaded_by)->toBe(auth()->id());
    Storage::disk('public')->assertExists($cover->variants['thumb']['webp']['path']);

    Livewire::test(ListMedia::class)
        ->assertCanSeeTableRecords($items)
        ->assertSee('بدون متن جایگزین')
        ->assertSee('کم‌حجم‌تر')
        ->assertSee($cover->variants['thumb']['webp']['path']);

    $this->get(MediaResource::getUrl('index'))->assertOk()->assertSee('cover.jpg');
});

it('fills a missing alt when the same file is uploaded again instead of creating a duplicate', function (): void {
    $this->actingAs(mediaAdmin());
    $media = libraryItem(1200, 800, alt: null);

    Livewire::test(ListMedia::class)
        ->callAction('upload', data: [
            'files' => [UploadedFile::fake()->createWithContent('again.jpg', (string) file_get_contents(MediaFixtures::jpeg(1200, 800)))],
            'alt' => 'دوباره',
        ])
        ->assertHasNoActionErrors();

    expect(Media::query()->count())->toBe(1)
        ->and($media->refresh()->alt)->toBe('دوباره');
});

it('requires alt text and regenerates the focal crops when the focal point moves', function (): void {
    $this->actingAs(mediaAdmin());
    $media = libraryItem(1600, 1000, alt: null);
    $disk = Storage::disk('public');
    $ogBefore = md5((string) $disk->get($media->variants['og']['jpg']['path']));
    $thumbBefore = md5((string) $disk->get($media->variants['thumb']['webp']['path']));
    $desktopBefore = md5((string) $disk->get($media->variants['desktop_1280']['webp']['path']));

    $this->get(MediaResource::getUrl('edit', ['record' => $media]))
        ->assertOk()
        ->assertSee('data-focal-picker', false)
        ->assertSee('نسخه‌ها');

    Livewire::test(EditMedia::class, ['record' => $media->getRouteKey()])
        ->assertSchemaStateSet(['focal' => ['x' => 0.5, 'y' => 0.5]])
        ->fillForm(['alt' => ''])
        ->call('save')
        ->assertHasFormErrors(['alt' => 'required']);

    Livewire::test(EditMedia::class, ['record' => $media->getRouteKey()])
        ->fillForm(['alt' => 'دو رنگ', 'title' => 'عنوان', 'caption' => 'زیرنویس', 'focal' => ['x' => 0.05, 'y' => 0.05]])
        ->call('save')
        ->assertHasNoFormErrors();

    $media->refresh();
    expect($media->alt)->toBe('دو رنگ')
        ->and($media->title)->toBe('عنوان')
        ->and($media->caption)->toBe('زیرنویس')
        ->and($media->focal_x)->toBe(0.05)
        ->and($media->focal_y)->toBe(0.05)
        ->and(md5((string) $disk->get($media->variants['og']['jpg']['path'])))->not->toBe($ogBefore)
        ->and(md5((string) $disk->get($media->variants['thumb']['webp']['path'])))->not->toBe($thumbBefore)
        ->and(md5((string) $disk->get($media->variants['desktop_1280']['webp']['path'])))->toBe($desktopBefore);

    expect(DB::table('activity_log')->where('description', 'media.updated')->count())->toBe(1);
});

it('shows the variants with savings and regenerates them on demand', function (): void {
    $this->actingAs(mediaAdmin());
    $media = libraryItem();
    $media->forceFill(['variants' => [], 'optimized_at' => null])->save();

    Livewire::test(EditMedia::class, ['record' => $media->getRouteKey()])
        ->assertSee('در صف بهینه‌سازی')
        ->callAction('regenerate')
        ->assertNotified('نسخه‌ها دوباره ساخته شدند.');

    $media->refresh();
    expect($media->optimized_at)->not->toBeNull()->and($media->variants)->toHaveKey('desktop_1280');

    Livewire::test(EditMedia::class, ['record' => $media->getRouteKey()])
        ->assertSee('desktop_1280 · webp')
        ->assertSee('کم‌حجم‌تر');
});

it('finds usages in settings and SEO overrides and in registered columns', function (): void {
    $logo = libraryItem(800, 800);
    $og = libraryItem(1300, 700);
    $unused = libraryItem(1000, 900);

    DB::table('settings')->updateOrInsert(['group' => 'organization', 'key' => 'logo_media_id'], ['value' => json_encode($logo->id)]);
    DB::table('seo_meta')->insert(['route_name' => 'cycle', 'og_media_id' => $og->id, 'sitemap_include' => true]);

    $usages = app(FindMediaUsages::class)->handle([$logo->id, $og->id, $unused->id]);

    expect($usages[$logo->id])->toBe(['تنظیمات: organization.logo_media_id'])
        ->and($usages[$og->id])->toBe(['تصویر اشتراک‌گذاری سئو: cycle'])
        ->and($usages)->not->toHaveKey($unused->id);

    FindMediaUsages::column('missing_table', 'media_id', 'جدول آینده');
    expect(app(FindMediaUsages::class)->used([$unused->id]))->toBe([]);

    $this->actingAs(mediaAdmin());
    Livewire::test(EditMedia::class, ['record' => $logo->getRouteKey()])
        ->assertSee('تنظیمات: organization.logo_media_id')
        ->mountAction('usages')
        ->assertSee('organization.logo_media_id');
});

it('bulk-deletes only unused media and keeps the files of used ones', function (): void {
    $used = libraryItem(800, 800);
    $unused = libraryItem(1000, 900);
    DB::table('seo_meta')->insert(['route_name' => 'home', 'og_media_id' => $used->id, 'sitemap_include' => true]);
    $unusedPath = $unused->path();

    $this->actingAs(mediaAdmin());

    Livewire::test(ListMedia::class)
        ->callTableBulkAction('deleteUnused', [$used, $unused])
        ->assertNotified();

    expect(Media::query()->pluck('id')->all())->toBe([$used->id]);
    Storage::disk('public')->assertMissing($unusedPath);
    Storage::disk('public')->assertExists($used->path());

    Livewire::test(EditMedia::class, ['record' => $used->getRouteKey()])
        ->callAction('delete')
        ->assertNotified('این تصویر در سایت استفاده شده و حذف نشد.');
    expect(Media::query()->whereKey($used->id)->exists())->toBeTrue();
});

it('counts an image used only inside a post body as used and keeps it on bulk delete', function (): void {
    $inBody = libraryItem(800, 800);
    $other = libraryItem(1000, 900);

    $post = Post::factory()->create([
        'title' => 'مقاله با تصویر',
        'body' => '<p>متن</p><figure><img data-media-id="'.$inBody->id.'" src="/media/'.$inBody->path().'" alt="تصویر"></figure>',
    ]);
    // A body that mentions the attribute without the id we look for must not match by prefix (id 1 vs 12…).
    Post::factory()->create(['body' => '<p><img data-media-id="'.$other->id.'9" alt="x"></p>']);

    expect($post->refresh()->body)->toContain('data-media-id="'.$inBody->id.'"');

    $usages = app(FindMediaUsages::class)->handle([$inBody->id, $other->id]);
    expect($usages[$inBody->id] ?? [])->toContain('تصویر داخل متن مقاله: مقاله با تصویر')
        ->and($usages)->not->toHaveKey($other->id);

    $this->actingAs(mediaAdmin());
    Livewire::test(ListMedia::class)
        ->callTableBulkAction('deleteUnused', [$inBody, $other])
        ->assertNotified();

    expect(Media::query()->whereKey($inBody->id)->exists())->toBeTrue()
        ->and(Media::query()->whereKey($other->id)->exists())->toBeFalse();
    Storage::disk('public')->assertExists($inBody->path());
});

it('provides a reusable media picker that selects existing media and uploads new ones', function (): void {
    $this->actingAs(mediaAdmin());
    $existing = libraryItem(1200, 800, alt: 'گل‌های بهاری');

    $other = libraryItem(900, 600, alt: 'دریا');

    $component = Livewire::test(MediaPickerHarness::class);
    $picker = $component->instance()->form->getFlatFields()['media_id'];
    expect($picker)->toBeInstanceOf(MediaPicker::class)
        ->and(array_keys($picker->getSearchResults('بهار')))->toBe([$existing->id])
        ->and($picker->getSearchResults('بهار')[$existing->id])->toContain('<img')->toContain('گل‌های بهاری')
        ->and(array_keys($picker->getOptions()))->toContain($existing->id, $other->id);

    $component->fillForm(['media_id' => $existing->id, 'mobile_media_id' => null])
        ->call('save')
        ->assertHasNoFormErrors()
        ->assertSet('saved.media_id', $existing->id);

    expect($component->instance()->form->getFlatFields()['media_id']->getOptionLabel())->toContain('گل‌های بهاری');

    $component->fillForm(['media_id' => 999999])->call('save')->assertHasFormErrors(['media_id']);

    Livewire::test(MediaPickerHarness::class)
        ->callAction(TestAction::make('createOption')->schemaComponent('media_id'), data: [
            'file' => fixtureUpload(1000, 700, 'new-upload.jpg'),
            'alt' => 'تصویر تازه',
        ])
        ->assertHasNoActionErrors();

    $new = Media::query()->where('alt', 'تصویر تازه')->firstOrFail();
    expect($new->variants)->toHaveKey('thumb');

    $field = MediaPicker::make('mobile')->mobileOverride();
    expect($field->getLabel())->toBe('تصویر موبایل (اختیاری)');
});

/**
 * Minimal Filament form hosting two MediaPickers, as a later resource would.
 */
final class MediaPickerHarness extends Component implements HasActions, HasSchemas
{
    use InteractsWithActions;
    use InteractsWithSchemas;

    /**
     * @var array<string, mixed>|null
     */
    public ?array $data = [];

    /**
     * @var array<string, mixed>
     */
    public array $saved = [];

    public function mount(): void
    {
        $this->form->fill();
    }

    public function form(Schema $schema): Schema
    {
        return $schema->components([
            MediaPicker::make('media_id')->required(),
            MediaPicker::make('mobile_media_id')->mobileOverride(),
        ])->statePath('data');
    }

    public function save(): void
    {
        $this->saved = $this->form->getState();
    }

    public function render(): string
    {
        return '<div>{{ $this->form }}<x-filament-actions::modals /></div>';
    }
}

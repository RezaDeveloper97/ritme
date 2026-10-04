<?php

declare(strict_types=1);

use App\Domain\Blog\Models\Post;
use App\Domain\Seo\Analysis\Queries\FindDuplicateSeoMeta;
use App\Domain\Seo\Models\SeoMeta;
use App\Filament\Auth\AdminRole;
use App\Filament\Resources\Blog\Posts\Pages\EditPost;
use App\Filament\Resources\Blog\Posts\PostResource;
use App\Filament\Resources\Seo\StaticPageSeo\StaticPageSeoResource;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Facades\Filament;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Storage;
use Livewire\Livewire;
use Tests\TestCase;

// Wiring of the analyser into the reusable SEO tab (needs the app; the analyser itself is covered by pure unit tests).
uses(TestCase::class, RefreshDatabase::class);

function analysisAdmin(AdminRole $role = AdminRole::Editor): User
{
    $user = User::factory()->create();
    $user->assignRole($role->value);

    return $user;
}

beforeEach(function (): void {
    Storage::fake('public');
    $this->seed([SettingsSeeder::class, AdminRolesSeeder::class]);
    Filament::setCurrentPanel('admin');
});

it('shows the live analysis in the post SEO tab and re-runs it when the focus keyword changes', function (): void {
    $this->actingAs(analysisAdmin());
    $post = Post::factory()->create([
        'title' => 'درد پریود چه زمانی طبیعی است',
        'slug' => 'درد-پریود',
        'body' => '<p>درد پریود برای بسیاری آشناست.</p><h3>پرش سطح</h3><img src="/media/x.webp">',
    ]);

    $this->get(PostResource::getUrl('edit', ['record' => $post]))
        ->assertOk()
        ->assertSee('تحلیل سئو')
        ->assertSee('data-seo-analysis', false)
        ->assertSee('data-seo-check="heading_hierarchy"', false)
        ->assertSee('کلیدواژه کانونی تعیین نشده است');

    Livewire::test(EditPost::class, ['record' => $post->getRouteKey()])
        ->fillForm(['seoMeta.focus_keyword' => 'درد پريود'])
        ->assertSee('data-seo-check="keyword_first_paragraph" data-severity="pass"', false)
        ->assertSee('data-seo-check="image_alt" data-severity="error"', false)
        ->assertDontSee('کلیدواژه کانونی تعیین نشده است')
        ->fillForm(['body' => '<p>این روش حتماً جواب می‌دهد.</p>'])
        ->assertSee('data-seo-check="red_lines" data-severity="error"', false);
});

it('analyses static pages without a body', function (): void {
    $this->actingAs(analysisAdmin(AdminRole::SeoManager));

    $this->get(StaticPageSeoResource::getUrl('edit', ['page' => 'stage-cycle']))
        ->assertOk()
        ->assertSee('data-seo-analysis', false)
        ->assertSee('data-seo-check="title_length"', false)
        ->assertDontSee('data-seo-check="content_length"', false);
});

it('finds SEO titles and descriptions already used by another page', function (): void {
    $own = SeoMeta::query()->create(['route_name' => 'cycle', 'title' => 'عنوان یکتا', 'description' => 'توضیح خودم']);
    SeoMeta::query()->create(['route_name' => 'tools', 'title' => ' عنوان تکراری ', 'description' => 'توضیح تکراری']);
    $find = new FindDuplicateSeoMeta;

    expect($find->handle('عنوان تکراری', 'توضیح دیگر'))->toBe(['title' => true, 'description' => false])
        ->and($find->handle('عنوان یکتا', 'توضیح خودم', $own->id))->toBe(['title' => false, 'description' => false])
        ->and($find->handle('', null))->toBe(['title' => null, 'description' => null])
        ->and($find->handle(null, 'توضیح تکراری'))->toBe(['title' => null, 'description' => true]);
});

<?php

declare(strict_types=1);

use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Seo\Redirects\Actions\DeleteRedirects;
use App\Domain\Seo\Redirects\Actions\SaveRedirect;
use App\Domain\Seo\Redirects\Models\Redirect;
use App\Models\User;
use Database\Seeders\SettingsSeeder;
use Spatie\Activitylog\Models\Activity;

beforeEach(function (): void {
    $this->seed([SettingsSeeder::class]);
});

it('deletes only the given redirects, logs each one and stops redirecting at once', function (): void {
    $save = app(SaveRedirect::class);
    $gone = $save->handle(new Redirect, ['from_path' => '/old-a', 'to_url' => '/blog', 'code' => 301]);
    $kept = $save->handle(new Redirect, ['from_path' => '/old-b', 'to_url' => '/shop', 'code' => 301]);
    $this->get('/old-a')->assertStatus(301);
    $admin = User::factory()->create();

    $deleted = app(DeleteRedirects::class)->handle([$gone->id, $gone->id, 999_999], $admin);

    expect($deleted)->toBe(1)
        ->and(Redirect::query()->pluck('id')->all())->toBe([$kept->id]);
    $log = Activity::query()->where('event', 'deleted')->sole();
    expect($log->causer?->is($admin))->toBeTrue()
        ->and($log->properties->get('old'))->toBe(['from_path' => '/old-a', 'to_url' => '/blog', 'code' => 301]);
    $this->get('/old-a')->assertNotFound();
    $this->get('/old-b')->assertStatus(301);
});

it('creates a regex redirect for a renamed directory category that covers every city', function (): void {
    $tehran = City::factory()->create(['slug' => 'tehran']);
    $category = PlaceCategory::factory()->create(['slug' => 'old-cat']);

    $category->update(['slug' => 'new-cat']);

    $row = Redirect::query()->sole();
    expect($row->is_regex)->toBeTrue()->and($row->is_auto)->toBeTrue();
    $this->get("/directory/{$tehran->slug}/old-cat")->assertStatus(301)->assertRedirect(url("/directory/{$tehran->slug}/new-cat"));
    $this->get('/directory/shiraz/old-cat')->assertStatus(301)->assertRedirect(url('/directory/shiraz/new-cat'));
});

it('ignores saves that leave the slug alone', function (): void {
    $category = PlaceCategory::factory()->create(['slug' => 'same']);

    $category->update(['name' => 'نام تازه']);
    $category->update(['slug' => ' same ']);

    expect(Redirect::query()->count())->toBe(0);
});

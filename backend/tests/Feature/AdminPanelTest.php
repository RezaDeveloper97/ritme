<?php

namespace Tests\Feature;

use App\Models\Admin;
use App\Models\Article;
use App\Models\OtpVerification;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class AdminPanelTest extends TestCase
{
    use RefreshDatabase;

    private function admin(string $role = Admin::ROLE_SUPER): Admin
    {
        return Admin::create([
            'name' => 'Test Admin',
            'email' => $role . '@ritme.test',
            'password' => 'secret123',
            'role' => $role,
            'is_active' => true,
        ]);
    }

    public function test_guest_is_redirected_to_login(): void
    {
        $this->get('/admin')->assertRedirect('/admin/login');
    }

    public function test_admin_can_log_in_with_valid_credentials(): void
    {
        $this->admin();

        $this->post('/admin/login', [
            'email' => 'super@ritme.test',
            'password' => 'secret123',
        ])->assertRedirect('/admin');

        $this->assertAuthenticatedAs(Admin::first(), 'admin');
    }

    public function test_login_fails_with_wrong_password(): void
    {
        $this->admin();

        $this->from('/admin/login')->post('/admin/login', [
            'email' => 'super@ritme.test',
            'password' => 'wrong',
        ])->assertRedirect('/admin/login');

        $this->assertGuest('admin');
    }

    public function test_deactivated_admin_cannot_log_in(): void
    {
        $admin = $this->admin();
        $admin->update(['is_active' => false]);

        $this->post('/admin/login', [
            'email' => 'super@ritme.test',
            'password' => 'secret123',
        ])->assertSessionHasErrors('email');

        $this->assertGuest('admin');
    }

    public function test_admin_can_change_own_password(): void
    {
        $admin = $this->admin();

        $this->actingAs($admin, 'admin')
            ->put('/admin/account/password', [
                'current_password' => 'secret123',
                'password' => 'newsecret123',
                'password_confirmation' => 'newsecret123',
            ])->assertSessionHas('status');

        $this->assertTrue(\Hash::check('newsecret123', $admin->fresh()->password));
    }

    public function test_admin_can_update_user_subscription_and_goal(): void
    {
        $admin = $this->admin();
        $user = User::factory()->create();

        $this->actingAs($admin, 'admin')->put("/admin/users/{$user->id}", [
            'name' => 'Renamed',
            'subscription_type' => 'premium',
            'user_goal' => 'ttc',
        ])->assertSessionHas('status');

        $this->assertSame('premium', $user->profile->subscription_type);
        $this->assertSame('ttc', $user->profile->user_goal);
        $this->assertSame('Renamed', $user->fresh()->name);
    }

    public function test_blocking_a_user_denies_new_otp_tokens(): void
    {
        $admin = $this->admin();
        $user = User::factory()->create(['mobile' => '09121110000', 'mobile_verified_at' => now()]);

        $this->actingAs($admin, 'admin')->post("/admin/users/{$user->id}/block")->assertSessionHas('status');
        $this->assertNotNull($user->fresh()->blocked_at);

        // A blocked user cannot obtain a token even with a valid OTP.
        OtpVerification::create([
            'mobile' => '09121110000',
            'code' => '1234',
            'expires_at' => now()->addMinutes(5),
        ]);

        $this->postJson('/api/v1/auth/verify-otp', [
            'mobile' => '09121110000',
            'code' => '1234',
        ])->assertStatus(403);
    }

    public function test_admin_can_create_article(): void
    {
        $admin = $this->admin();

        $this->actingAs($admin, 'admin')->post('/admin/articles', [
            'slug' => 'hello-world',
            'title' => ['fa' => 'سلام', 'en' => 'Hello'],
            'is_published' => '1',
        ])->assertRedirect('/admin/articles');

        $article = Article::first();
        $this->assertSame('سلام', $article->title['fa']);
        $this->assertTrue($article->is_published);
    }

    public function test_editor_cannot_access_admin_management(): void
    {
        $editor = $this->admin(Admin::ROLE_EDITOR);

        $this->actingAs($editor, 'admin')->get('/admin/admins')->assertForbidden();
    }

    public function test_super_admin_can_access_admin_management(): void
    {
        $super = $this->admin();

        $this->actingAs($super, 'admin')->get('/admin/admins')->assertOk();
    }
}

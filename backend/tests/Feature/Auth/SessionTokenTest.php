<?php

namespace Tests\Feature\Auth;

use App\Models\Admin;
use App\Models\OtpVerification;
use App\Models\User;
use DateInterval;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Queue;
use Illuminate\Testing\TestResponse;
use Laravel\Passport\ClientRepository;
use Laravel\Passport\Passport;
use Laravel\Passport\Token;
use Tests\TestCase;

/**
 * A signed-in user stays signed in for a year, and every 401 the API sends
 * says why (`error_code`), so the client only drops its session when the
 * server really ended it.
 *
 * These use real bearer tokens (not Passport::actingAs) so the JWT, the token
 * row and the guard's validation all run exactly as in production.
 */
class SessionTokenTest extends TestCase
{
    use RefreshDatabase;

    private const MOBILE = '09125556666';

    private static ?string $keyDir = null;

    protected function setUp(): void
    {
        parent::setUp();

        // A throwaway key pair, so the suite never depends on (or writes) the
        // developer's storage/oauth-*.key.
        if (self::$keyDir === null) {
            self::$keyDir = sys_get_temp_dir().'/ritme-passport-test-'.getmypid();
            @mkdir(self::$keyDir, 0700, true);
            $key = openssl_pkey_new(['private_key_bits' => 2048, 'private_key_type' => OPENSSL_KEYTYPE_RSA]);
            openssl_pkey_export($key, $private);
            file_put_contents(self::$keyDir.'/oauth-private.key', $private);
            file_put_contents(self::$keyDir.'/oauth-public.key', openssl_pkey_get_details($key)['key']);
            chmod(self::$keyDir.'/oauth-private.key', 0600);
            chmod(self::$keyDir.'/oauth-public.key', 0600);
        }
        Passport::loadKeysFrom(self::$keyDir);

        app(ClientRepository::class)->createPersonalAccessGrantClient('Test Personal Access');
    }

    // ── Lifetime ────────────────────────────────────────────────────────────

    public function test_token_issued_at_otp_verification_lives_365_days(): void
    {
        $accessToken = $this->signIn();

        $row = Token::sole();
        $this->assertSame(365, (int) round($row->created_at->diffInDays($row->expires_at)));

        $claims = $this->jwtClaims($accessToken);
        // lcobucci writes iat/exp with microseconds; compare whole seconds.
        $this->assertSame(365 * 86400, (int) round($claims['exp'] - $claims['iat']));
        $this->assertSame($row->getKey(), $claims['jti']);
        $this->assertSame($row->expires_at->getTimestamp(), (int) $claims['exp']);
    }

    public function test_valid_token_is_accepted(): void
    {
        $token = $this->signIn();

        $this->api('GET', '/api/v1/auth/user', $token)
            ->assertOk()
            ->assertJsonPath('data.user.mobile', self::MOBILE);
    }

    // ── 401 error codes ─────────────────────────────────────────────────────

    public function test_missing_token_is_unauthenticated(): void
    {
        $this->api('GET', '/api/v1/auth/user')
            ->assertStatus(401)
            ->assertExactJson(['message' => 'Unauthenticated.', 'error_code' => 'unauthenticated']);
    }

    public function test_malformed_token_is_unauthenticated(): void
    {
        $this->api('GET', '/api/v1/auth/user', 'not-a-jwt')
            ->assertStatus(401)
            ->assertJsonPath('error_code', 'unauthenticated');
    }

    public function test_token_with_a_bad_signature_is_unauthenticated(): void
    {
        $token = $this->signIn();
        // Flip the signature: the payload is still ours, the signature isn't.
        [$header, $payload, $signature] = explode('.', $token);
        $forged = $header.'.'.$payload.'.'.strrev($signature);

        $this->api('GET', '/api/v1/auth/user', $forged)
            ->assertStatus(401)
            ->assertJsonPath('error_code', 'unauthenticated');
    }

    public function test_logged_out_token_is_token_revoked(): void
    {
        $token = $this->signIn();

        $this->api('POST', '/api/v1/auth/logout', $token)->assertOk();

        $this->api('GET', '/api/v1/auth/user', $token)
            ->assertStatus(401)
            ->assertExactJson(['message' => 'Unauthenticated.', 'error_code' => 'token_revoked']);
    }

    public function test_expired_token_is_token_expired(): void
    {
        $user = User::factory()->create();
        // JWT validation reads the real clock, not Carbon's test clock, so
        // mint a token whose expiry is already an hour in the past.
        $past = new DateInterval('PT1H');
        $past->invert = 1;
        Passport::personalAccessTokensExpireIn($past);
        $token = $user->createToken('auth_token')->accessToken;

        $this->api('GET', '/api/v1/auth/user', $token)
            ->assertStatus(401)
            ->assertExactJson(['message' => 'Unauthenticated.', 'error_code' => 'token_expired']);
    }

    // ── Sliding refresh ─────────────────────────────────────────────────────

    public function test_refresh_is_a_no_op_while_plenty_of_time_is_left(): void
    {
        $token = $this->signIn();

        $this->api('POST', '/api/v1/auth/refresh-session', $token)
            ->assertOk()
            ->assertJsonPath('data.refreshed', false)
            ->assertJsonMissingPath('data.access_token');

        $this->assertSame(1, Token::count());
        $this->api('GET', '/api/v1/auth/user', $token)->assertOk();
    }

    public function test_refresh_near_expiry_issues_a_fresh_year_and_revokes_the_old_token(): void
    {
        $old = $this->signIn();
        $oldRow = Token::sole();
        $oldRow->forceFill(['expires_at' => now()->addDays(10)])->save();

        $response = $this->api('POST', '/api/v1/auth/refresh-session', $old)
            ->assertOk()
            ->assertJsonPath('data.refreshed', true)
            ->assertJsonPath('data.token_type', 'Bearer');

        $new = $response->json('data.access_token');
        $this->assertNotEmpty($new);

        $newRow = Token::whereKeyNot($oldRow->getKey())->sole();
        $this->assertSame(365, (int) round(now()->diffInDays($newRow->expires_at)));
        $this->assertFalse((bool) $newRow->revoked);
        $this->assertTrue((bool) $oldRow->fresh()->revoked);

        $this->api('GET', '/api/v1/auth/user', $new)->assertOk();
        $this->api('GET', '/api/v1/auth/user', $old)
            ->assertStatus(401)
            ->assertJsonPath('error_code', 'token_revoked');
    }

    public function test_refresh_requires_a_valid_token(): void
    {
        $this->api('POST', '/api/v1/auth/refresh-session')
            ->assertStatus(401)
            ->assertJsonPath('error_code', 'unauthenticated');
    }

    // ── Server-side revocation paths still work ─────────────────────────────

    public function test_admin_block_revokes_every_token(): void
    {
        $first = $this->signIn();
        $second = $this->signIn();
        $user = User::where('mobile', self::MOBILE)->sole();

        $admin = Admin::create([
            'name' => 'Test Admin',
            'email' => 'super@ritme.test',
            'password' => 'secret123',
            'role' => Admin::ROLE_SUPER,
            'is_active' => true,
        ]);
        $this->actingAs($admin, 'admin')
            ->post(route('admin.users.block', $user))
            ->assertRedirect();

        $this->assertSame(0, Token::where('revoked', false)->count());
        foreach ([$first, $second] as $token) {
            $this->api('GET', '/api/v1/auth/user', $token)
                ->assertStatus(401)
                ->assertJsonPath('error_code', 'token_revoked');
        }
    }

    public function test_account_deletion_revokes_every_token(): void
    {
        $first = $this->signIn();
        $second = $this->signIn();

        $this->api('DELETE', '/api/v1/account', $second)->assertOk();

        $this->assertSame(0, Token::where('revoked', false)->count());
        foreach ([$first, $second] as $token) {
            $this->api('GET', '/api/v1/auth/user', $token)
                ->assertStatus(401)
                ->assertJsonPath('error_code', 'token_revoked');
        }
    }

    // ── Helpers ─────────────────────────────────────────────────────────────

    /** Runs the real send-otp → verify-otp flow and returns the access token. */
    private function signIn(): string
    {
        Queue::fake();
        OtpVerification::where('mobile', self::MOBILE)->delete();

        $this->postJson('/api/v1/auth/send-otp', ['mobile' => self::MOBILE])->assertOk();
        $code = (string) OtpVerification::where('mobile', self::MOBILE)->latest()->value('code');

        return $this->postJson('/api/v1/auth/verify-otp', ['mobile' => self::MOBILE, 'code' => $code])
            ->assertOk()
            ->json('data.access_token');
    }

    private function api(string $method, string $uri, ?string $token = null): TestResponse
    {
        // The guard caches its user for the life of the app; a real request
        // starts from nothing, so each call here does too.
        $this->app['auth']->forgetGuards();

        $headers = $token === null ? [] : ['Authorization' => 'Bearer '.$token];

        $response = $this->withHeaders($headers)->json($method, $uri);
        $this->flushHeaders();

        return $response;
    }

    /** @return array<string, mixed> */
    private function jwtClaims(string $jwt): array
    {
        $payload = explode('.', $jwt)[1];

        return json_decode(base64_decode(strtr($payload, '-_', '+/')), true);
    }
}

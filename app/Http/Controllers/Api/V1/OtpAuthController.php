<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Models\OtpVerification;
use App\Models\User;
use App\Services\Sms\SmsService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Validator;

class OtpAuthController extends Controller
{
    /**
     * @OA\Post(
     *     path="/v1/auth/send-otp",
     *     summary="Send OTP for login/register (unified)",
     *     description="Sends a 4-digit OTP to the mobile number. Creates new user if not exists. Use is_test=true for testing (OTP will be 1111).",
     *     tags={"V1 - OTP Authentication"},
     *
     *     @OA\RequestBody(
     *         required=true,
     *
     *         @OA\JsonContent(
     *             required={"mobile"},
     *
     *             @OA\Property(property="mobile", type="string", example="09123456789", description="Iranian mobile number (11 digits)"),
     *             @OA\Property(property="is_test", type="boolean", example=true, description="Test mode - skips SMS and uses 1111 as OTP")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="OTP sent successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="OTP sent successfully"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="new_user", type="boolean", example=true, description="True if this is a new user registration"),
     *                 @OA\Property(property="expires_in", type="integer", example=120, description="OTP expiration time in seconds")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=422,
     *         description="Validation error",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=false),
     *             @OA\Property(property="message", type="string", example="Validation failed"),
     *             @OA\Property(property="errors", type="object")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=429,
     *         description="Too many requests",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=false),
     *             @OA\Property(property="message", type="string", example="Please wait before requesting a new OTP"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="retry_after", type="integer", example=30)
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=500,
     *         description="SMS sending failed",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=false),
     *             @OA\Property(property="message", type="string", example="Failed to send OTP. Please try again.")
     *         )
     *     )
     * )
     */
    public function sendOtp(Request $request): JsonResponse
    {
        $validator = Validator::make($request->all(), [
            'mobile' => ['required', 'string', 'regex:/^09[0-9]{9}$/'],
            'is_test' => ['nullable', 'boolean'],
        ], [
            'mobile.regex' => 'Mobile number must be a valid Iranian mobile number (e.g., 09123456789)',
        ]);

        if ($validator->fails()) {
            return response()->json([
                'success' => false,
                'message' => 'Validation failed',
                'errors' => $validator->errors(),
            ], 422);
        }

        $mobile = $request->mobile;
        $isTest = $request->boolean('is_test', false);
        $resendAfter = config('sms.otp.resend_after', 60);

        // Check for recent OTP requests (rate limiting) - skip in test mode
        if (! $isTest) {
            $recentOtp = OtpVerification::where('mobile', $mobile)
                ->where('created_at', '>', now()->subSeconds($resendAfter))
                ->first();

            if ($recentOtp) {
                $retryAfter = $resendAfter - now()->diffInSeconds($recentOtp->created_at);

                return response()->json([
                    'success' => false,
                    'message' => 'Please wait before requesting a new OTP',
                    'data' => [
                        'retry_after' => $retryAfter,
                    ],
                ], 429);
            }
        }

        // Check if user exists
        $user = User::where('mobile', $mobile)->first();
        $isNewUser = ! $user;

        // Generate OTP - use 1111 in test mode
        $otpCode = $isTest ? '1111' : SmsService::generateOtp(config('sms.otp.length', 4));
        $expiresIn = config('sms.otp.expires_in', 2);

        // Delete old OTPs for this mobile
        OtpVerification::where('mobile', $mobile)->delete();

        // Create new OTP record
        OtpVerification::create([
            'mobile' => $mobile,
            'code' => $otpCode,
            'expires_at' => now()->addMinutes($expiresIn),
        ]);

        // Send OTP via SMS - skip in test mode
        if (! $isTest) {
            $smsService = new SmsService;
            $sent = $smsService->sendOtp($mobile, $otpCode, 'login_otp');

            if (! $sent) {
                return response()->json([
                    'success' => false,
                    'message' => 'Failed to send OTP. Please try again.',
                ], 500);
            }
        }

        return response()->json([
            'success' => true,
            'message' => $isTest ? 'OTP set to 1111 (test mode)' : 'OTP sent successfully',
            'data' => [
                'new_user' => $isNewUser,
                'expires_in' => $expiresIn * 60, // seconds
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/v1/auth/verify-otp",
     *     summary="Verify OTP and get access token",
     *     description="Verifies the OTP code and returns access token. Creates user if new.",
     *     tags={"V1 - OTP Authentication"},
     *
     *     @OA\RequestBody(
     *         required=true,
     *
     *         @OA\JsonContent(
     *             required={"mobile", "code"},
     *
     *             @OA\Property(property="mobile", type="string", example="09123456789"),
     *             @OA\Property(property="code", type="string", example="1234", description="4-digit OTP code")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=200,
     *         description="OTP verified successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Login successful"),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="user", type="object",
     *                     @OA\Property(property="id", type="integer", example=1),
     *                     @OA\Property(property="name", type="string", nullable=true, example=null),
     *                     @OA\Property(property="mobile", type="string", example="09123456789"),
     *                     @OA\Property(property="mobile_verified_at", type="string", format="date-time")
     *                 ),
     *                 @OA\Property(property="new_user", type="boolean", example=true),
     *                 @OA\Property(property="access_token", type="string", example="eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9..."),
     *                 @OA\Property(property="token_type", type="string", example="Bearer")
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=400,
     *         description="OTP expired or invalid",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=false),
     *             @OA\Property(property="message", type="string", example="OTP has expired")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=422,
     *         description="Validation error or invalid code",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=false),
     *             @OA\Property(property="message", type="string", example="Invalid OTP code")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=429,
     *         description="Too many attempts",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=false),
     *             @OA\Property(property="message", type="string", example="Too many attempts. Please request a new OTP.")
     *         )
     *     )
     * )
     */
    public function verifyOtp(Request $request): JsonResponse
    {
        $validator = Validator::make($request->all(), [
            'mobile' => ['required', 'string', 'regex:/^09[0-9]{9}$/'],
            'code' => ['required', 'string', 'size:4'],
        ]);

        if ($validator->fails()) {
            return response()->json([
                'success' => false,
                'message' => 'Validation failed',
                'errors' => $validator->errors(),
            ], 422);
        }

        $mobile = $request->mobile;
        $code = $request->code;

        // Find the OTP record
        $otp = OtpVerification::where('mobile', $mobile)
            ->whereNull('verified_at')
            ->latest()
            ->first();

        if (! $otp) {
            return response()->json([
                'success' => false,
                'message' => 'No OTP found. Please request a new one.',
            ], 400);
        }

        // Check if expired
        if ($otp->isExpired()) {
            return response()->json([
                'success' => false,
                'message' => 'OTP has expired. Please request a new one.',
            ], 400);
        }

        // Check max attempts
        if ($otp->maxAttemptsExceeded()) {
            return response()->json([
                'success' => false,
                'message' => 'Too many attempts. Please request a new OTP.',
            ], 429);
        }

        // Verify the code
        if ($otp->code !== $code) {
            $otp->incrementAttempts();
            $remainingAttempts = config('sms.otp.max_attempts', 5) - $otp->attempts;

            return response()->json([
                'success' => false,
                'message' => 'Invalid OTP code',
                'data' => [
                    'remaining_attempts' => max(0, $remainingAttempts),
                ],
            ], 422);
        }

        // Mark OTP as verified
        $otp->markAsVerified();

        // Find or create user
        $user = User::where('mobile', $mobile)->first();
        $isNewUser = ! $user;

        if (! $user) {
            $user = User::create([
                'mobile' => $mobile,
            ]);
        }

        // Update mobile verification timestamp
        if (! $user->mobile_verified_at) {
            $user->update(['mobile_verified_at' => now()]);
        }

        // Create access token
        $token = $user->createToken('auth_token')->accessToken;

        return response()->json([
            'success' => true,
            'message' => $isNewUser ? 'Registration successful' : 'Login successful',
            'data' => [
                'user' => $user->fresh(),
                'new_user' => $isNewUser,
                'access_token' => $token,
                'token_type' => 'Bearer',
            ],
        ]);
    }

    /**
     * @OA\Post(
     *     path="/v1/auth/logout",
     *     summary="Logout user (revoke token)",
     *     tags={"V1 - OTP Authentication"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Response(
     *         response=200,
     *         description="Logout successful",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="message", type="string", example="Successfully logged out")
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=401,
     *         description="Unauthenticated"
     *     )
     * )
     */
    public function logout(Request $request): JsonResponse
    {
        $request->user()->token()->revoke();

        return response()->json([
            'success' => true,
            'message' => 'Successfully logged out',
        ]);
    }

    /**
     * @OA\Get(
     *     path="/v1/auth/user",
     *     summary="Get authenticated user info",
     *     tags={"V1 - OTP Authentication"},
     *     security={{"bearerAuth":{}}},
     *
     *     @OA\Response(
     *         response=200,
     *         description="User info retrieved successfully",
     *
     *         @OA\JsonContent(
     *
     *             @OA\Property(property="success", type="boolean", example=true),
     *             @OA\Property(property="data", type="object",
     *                 @OA\Property(property="user", type="object",
     *                     @OA\Property(property="id", type="integer", example=1),
     *                     @OA\Property(property="name", type="string", nullable=true, example="Sara"),
     *                     @OA\Property(property="mobile", type="string", example="09123456789"),
     *                     @OA\Property(property="mobile_verified_at", type="string", format="date-time"),
     *                     @OA\Property(property="created_at", type="string", format="date-time"),
     *                     @OA\Property(property="updated_at", type="string", format="date-time")
     *                 )
     *             )
     *         )
     *     ),
     *
     *     @OA\Response(
     *         response=401,
     *         description="Unauthenticated"
     *     )
     * )
     */
    public function user(Request $request): JsonResponse
    {
        return response()->json([
            'success' => true,
            'data' => [
                'user' => $request->user(),
            ],
        ]);
    }
}

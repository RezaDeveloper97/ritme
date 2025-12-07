<?php

namespace App\Http\Controllers;

/**
 * @OA\Info(
 *     version="1.0.0",
 *     title="Ritme Salamat API",
 *     description="API Documentation for Ritme Salamat Application",
 *
 *     @OA\Contact(
 *         email="support@ritmesalamat.com"
 *     )
 * )
 *
 * @OA\Server(
 *     url="/api/v1",
 *     description="API Server V1"
 * )
 *
 * @OA\SecurityScheme(
 *     securityScheme="bearerAuth",
 *     type="http",
 *     scheme="bearer",
 *     bearerFormat="JWT",
 *     description="Enter JWT Bearer token"
 * )
 *
 * @OA\Tag(
 *     name="Auth",
 *     description="Authentication endpoints"
 * )
 *
 * @OA\Tag(
 *     name="Profile",
 *     description="User profile management"
 * )
 *
 * @OA\Tag(
 *     name="Daily Health Log",
 *     description="Daily health tracking"
 * )
 */
abstract class Controller
{
    //
}

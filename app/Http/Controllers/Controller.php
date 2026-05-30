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
 *     url="/api/v1/",
 *     description="API Server"
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
 *
 * @OA\Tag(
 *     name="Cycle Calculation",
 *     description="Cycle data calculation and fertility prediction engine"
 * )
 *
 * @OA\Tag(
 *     name="Pregnancy",
 *     description="Pregnancy mode management and onboarding"
 * )
 *
 * @OA\Tag(
 *     name="Pregnancy Symptoms",
 *     description="Daily pregnancy symptom tracking"
 * )
 *
 * @OA\Tag(
 *     name="Pregnancy Weekly",
 *     description="Weekly pregnancy monitoring and content"
 * )
 *
 * @OA\Tag(
 *     name="Pregnancy Alerts",
 *     description="Pregnancy health alerts and emergency notifications"
 * )
 *
 * @OA\Tag(
 *     name="Home",
 *     description="Aggregated home page: server-driven sections, per-section endpoints and section actions"
 * )
 */
abstract class Controller
{
    //
}

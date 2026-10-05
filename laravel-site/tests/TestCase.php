<?php

declare(strict_types=1);

namespace Tests;

use Illuminate\Filesystem\Filesystem;
use Illuminate\Foundation\Testing\TestCase as BaseTestCase;
use Illuminate\Support\Facades\ParallelTesting;

abstract class TestCase extends BaseTestCase
{
    private static bool $cleanupRegistered = false;

    protected function setUp(): void
    {
        parent::setUp();

        // Tests never depend on a built asset manifest; asset tests opt back in explicitly.
        $this->withoutVite();

        $this->isolateFakeDisks();
    }

    /**
     * Storage::fake() roots get the parallel token as suffix (`public_test_3`) — but a plain serial run has no token
     * and uses `storage/framework/testing/disks/public` itself, so two runs at the same time (an IDE run beside
     * `composer verify`, several agents) wiped each other's media files mid-test. Serial runs get a per-process token
     * (only the fake-disk roots read it: `ParallelTesting::running()` stays false, databases are untouched) and the
     * process removes its disks when it exits.
     */
    private function isolateFakeDisks(): void
    {
        if (ParallelTesting::token() !== false) {
            return;
        }

        $token = 'pid'.getmypid();
        ParallelTesting::resolveTokenUsing(static fn (): string => $token);

        if (! self::$cleanupRegistered) {
            self::$cleanupRegistered = true;
            $disks = storage_path('framework/testing/disks');
            register_shutdown_function(static function () use ($disks, $token): void {
                $files = new Filesystem;
                foreach (glob($disks.'/*_test_'.$token, GLOB_ONLYDIR) ?: [] as $directory) {
                    $files->deleteDirectory($directory);
                }
            });
        }
    }
}

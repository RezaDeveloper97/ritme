<?php

namespace App\Console\Commands;

use App\Services\Language\TranslationStore;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\File;

/**
 * Copies the frontend's checked-in message namespaces into the backend's seed
 * folder (resources/translations), which is the template every newly created
 * language is cloned from and the fallback the API serves.
 *
 * `frontend/messages` stays the source of truth for the built-in locales — run
 * this after adding or renaming keys there so a language created tomorrow gets
 * the new keys instead of falling back for them.
 *
 *   php artisan translations:import              # defaults to ../frontend/messages
 *   php artisan translations:import /path/to/messages
 */
class ImportTranslations extends Command
{
    protected $signature = 'translations:import {path? : Directory holding one folder per locale}';

    protected $description = "Sync the frontend's message JSON into the backend translation seed";

    public function handle(TranslationStore $store): int
    {
        $source = $this->argument('path') ?? base_path('../frontend/messages');

        if (! File::isDirectory($source)) {
            $this->error("Not a directory: {$source}");

            return self::FAILURE;
        }

        $copied = 0;

        foreach (File::directories($source) as $localeDir) {
            $code = basename($localeDir);
            $target = $store->seedPath($code);
            File::ensureDirectoryExists($target);

            foreach (File::files($localeDir) as $file) {
                if ($file->getExtension() !== 'json') {
                    continue;
                }

                File::copy($file->getPathname(), $target.'/'.$file->getFilename());
                $copied++;
            }

            $this->line("  {$code}: ".count(File::files($target)).' namespaces');
        }

        $this->info("Imported {$copied} translation files into ".$store->seedPath());

        return self::SUCCESS;
    }
}

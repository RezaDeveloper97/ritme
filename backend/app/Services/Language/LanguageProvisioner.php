<?php

namespace App\Services\Language;

use App\Models\Language;
use App\Models\MessageContent;
use Illuminate\Support\Facades\File;
use Illuminate\Support\Facades\Log;

/**
 * Everything that must happen around a language row, beyond the row itself.
 *
 * Creating a language is not just an insert: the locale needs UI-string files
 * the frontend can serve, Laravel `lang/` files for server-side validation
 * messages, and its own copy of the editable smart-message rows. Provisioning
 * all three here keeps LanguageController thin and makes "add a language" a
 * single, repeatable operation.
 *
 * Every step is idempotent and non-destructive: re-provisioning an existing
 * locale never overwrites text an admin has already translated.
 */
class LanguageProvisioner
{
    public function __construct(
        private readonly LanguageRegistry $registry,
        private readonly TranslationStore $translations,
    ) {}

    /**
     * Generate everything a freshly created locale needs.
     *
     * @return array{messages: int, lang_files: int, smart_messages: int}
     */
    public function provision(Language $language, ?string $copyFrom = null): array
    {
        $source = $copyFrom ?? $this->registry->defaultCode();

        return [
            'messages' => $this->translations->generateFor($language->code, $source),
            'lang_files' => $this->copyLangFiles($language->code, $source),
            'smart_messages' => $this->cloneMessageContents($language->code, $source),
        ];
    }

    /** Remove the generated artifacts of a deleted locale. */
    public function deprovision(string $code): void
    {
        $this->translations->deleteFor($code);

        MessageContent::query()->where('locale', $code)->delete();

        $langDir = lang_path($code);

        if (File::isDirectory($langDir)) {
            File::deleteDirectory($langDir);
        }
    }

    /**
     * Clone `lang/{source}/*.php` (validation + server-rendered strings) so
     * `__()` under the new locale resolves instead of echoing raw keys.
     * Laravel's own fallback_locale covers anything still missing.
     */
    private function copyLangFiles(string $code, string $source): int
    {
        $from = lang_path($source);
        $to = lang_path($code);

        if (! File::isDirectory($from) || $code === $source) {
            return 0;
        }

        File::ensureDirectoryExists($to);
        $copied = 0;

        foreach (File::files($from) as $file) {
            $target = $to.'/'.$file->getFilename();

            // Never clobber a translation someone already wrote by hand.
            if (is_file($target)) {
                continue;
            }

            File::copy($file->getPathname(), $target);
            $copied++;
        }

        return $copied;
    }

    /**
     * Give the new locale its own editable smart-message rows, copied from the
     * source locale. Without this the messages page would show nothing to edit
     * for the language and users would silently get the fallback locale's text.
     *
     * New rows land unapproved on purpose: they still hold the source
     * language's words, and an admin must review the translation before it
     * reaches users (MessageContent::scopeLive requires is_approved).
     */
    private function cloneMessageContents(string $code, string $source): int
    {
        if ($code === $source) {
            return 0;
        }

        if (MessageContent::query()->where('locale', $code)->exists()) {
            return 0;
        }

        $cloned = 0;

        MessageContent::query()
            ->where('locale', $source)
            ->chunkById(200, function ($rows) use ($code, &$cloned): void {
                foreach ($rows as $row) {
                    try {
                        MessageContent::create([
                            'group' => $row->group,
                            'item_key' => $row->item_key,
                            'locale' => $code,
                            'label' => $row->label,
                            'payload' => $row->payload,
                            'is_active' => $row->is_active,
                            'is_approved' => false,
                            'sort_order' => $row->sort_order,
                        ]);
                        $cloned++;
                    } catch (\Throwable $e) {
                        // A unique-constraint clash means the row already
                        // exists; anything else is worth knowing about but must
                        // not abort creating the language.
                        Log::warning('Could not clone message content for new locale', [
                            'locale' => $code,
                            'group' => $row->group,
                            'item_key' => $row->item_key,
                            'error' => $e->getMessage(),
                        ]);
                    }
                }
            });

        return $cloned;
    }
}

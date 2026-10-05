<?php

declare(strict_types=1);

namespace Tests\Arch;

use FilesystemIterator;
use RecursiveDirectoryIterator;
use RecursiveIteratorIterator;
use SplFileInfo;

/**
 * Token-based source scanning for the architecture tests. Pest's arch expectations parse every class with
 * php-parser and keep the ASTs (~550 MB RSS for app/); a token pass per file stays in a few MB and is far faster.
 * Files are read one at a time and nothing but small result arrays is kept.
 */
final class SourceFiles
{
    public static function root(): string
    {
        return dirname(__DIR__, 2);
    }

    /**
     * @return list<string> absolute paths of the PHP files under `$directory` (relative to the project root), sorted
     */
    public static function in(string $directory): array
    {
        $path = self::root().'/'.$directory;
        if (! is_dir($path)) {
            return [];
        }

        $files = [];
        $iterator = new RecursiveIteratorIterator(new RecursiveDirectoryIterator($path, FilesystemIterator::SKIP_DOTS));
        foreach ($iterator as $file) {
            if ($file instanceof SplFileInfo && str_ends_with($file->getFilename(), '.php')) {
                $files[] = $file->getPathname();
            }
        }
        sort($files);

        return $files;
    }

    public static function relative(string $path): string
    {
        return str_replace(self::root().'/', '', $path);
    }

    /**
     * @return list<array{0: int, 1: string, 2: int}|string>
     */
    public static function tokens(string $path): array
    {
        return token_get_all((string) file_get_contents($path));
    }

    /**
     * Every class/namespace name the file mentions — `use` imports, fully qualified and qualified names — without the
     * leading backslash. Relative names inside the file's own namespace are not resolved: they can only point into
     * that namespace, which is never a forbidden layer for the file itself.
     *
     * @return list<string>
     */
    public static function names(string $path): array
    {
        $names = [];
        foreach (self::tokens($path) as $token) {
            if (is_array($token) && in_array($token[0], [T_NAME_QUALIFIED, T_NAME_FULLY_QUALIFIED], true)) {
                $names[] = ltrim($token[1], '\\');
            }
        }

        return array_values(array_unique($names));
    }

    /**
     * Names in `$path` that are `$forbidden` or live below it (namespace prefix match, case-insensitive like PHP).
     *
     * @param  list<string>  $forbidden
     * @return list<string>
     */
    public static function forbiddenNames(string $path, array $forbidden): array
    {
        $hits = [];
        foreach (self::names($path) as $name) {
            foreach ($forbidden as $prefix) {
                $lower = strtolower($name);
                $prefix = strtolower($prefix);
                if ($lower === $prefix || str_starts_with($lower, $prefix.'\\')) {
                    $hits[] = $name;
                }
            }
        }

        return $hits;
    }

    /**
     * Calls of global functions named in `$functions` (not methods, static calls or declarations), as `line` numbers.
     *
     * @param  list<string>  $functions
     * @return list<int>
     */
    public static function functionCalls(string $path, array $functions): array
    {
        $tokens = self::tokens($path);
        $lines = [];
        foreach ($tokens as $i => $token) {
            if (! is_array($token) || ! in_array($token[0], [T_STRING, T_NAME_FULLY_QUALIFIED], true)
                || ! in_array(strtolower(ltrim($token[1], '\\')), $functions, true)) {
                continue;
            }
            $prev = self::neighbour($tokens, $i, -1);
            if (is_array($prev) && in_array($prev[0], [T_OBJECT_OPERATOR, T_NULLSAFE_OBJECT_OPERATOR, T_DOUBLE_COLON, T_FUNCTION, T_NEW, T_CONST], true)) {
                continue;
            }
            if (self::neighbour($tokens, $i, 1) === '(') {
                $lines[] = $token[2];
            }
        }

        return $lines;
    }

    /** `declare(strict_types=1);` is the first statement after the open tag. */
    public static function declaresStrictTypes(string $path): bool
    {
        $code = [];
        foreach (self::tokens($path) as $token) {
            if (is_array($token) && in_array($token[0], [T_OPEN_TAG, T_WHITESPACE, T_COMMENT, T_DOC_COMMENT], true)) {
                continue;
            }
            $code[] = is_array($token) ? $token[1] : $token;
            if (count($code) === 7) {
                break;
            }
        }

        return strtolower(implode('', $code)) === 'declare(strict_types=1);';
    }

    /**
     * Named classes declared in the file: [name, isFinal, isAbstract]. Anonymous classes are skipped.
     *
     * @return list<array{0: string, 1: bool, 2: bool}>
     */
    public static function classes(string $path): array
    {
        $tokens = self::tokens($path);
        $classes = [];
        foreach ($tokens as $i => $token) {
            if (! is_array($token) || $token[0] !== T_CLASS) {
                continue;
            }
            $name = self::neighbour($tokens, $i, 1);
            if (! is_array($name) || $name[0] !== T_STRING) {
                continue; // `new class`, `Foo::class`
            }
            $prev = self::neighbour($tokens, $i, -1);
            if (is_array($prev) && $prev[0] === T_DOUBLE_COLON) {
                continue;
            }
            $final = false;
            $abstract = false;
            for ($j = $i - 1; $j >= 0; $j--) {
                $modifier = $tokens[$j];
                if (! is_array($modifier) || ! in_array($modifier[0], [T_FINAL, T_ABSTRACT, T_READONLY, T_WHITESPACE, T_COMMENT, T_DOC_COMMENT, T_ATTRIBUTE], true)) {
                    break;
                }
                $final = $final || $modifier[0] === T_FINAL;
                $abstract = $abstract || $modifier[0] === T_ABSTRACT;
            }
            $classes[] = [$name[1], $final, $abstract];
        }

        return $classes;
    }

    /**
     * The nearest non-whitespace/comment token before (`$step` = -1) or after (+1) `$i`.
     *
     * @param  list<array{0: int, 1: string, 2: int}|string>  $tokens
     * @return array{0: int, 1: string, 2: int}|string|null
     */
    private static function neighbour(array $tokens, int $i, int $step): array|string|null
    {
        for ($j = $i + $step; isset($tokens[$j]); $j += $step) {
            $token = $tokens[$j];
            if (is_array($token) && in_array($token[0], [T_WHITESPACE, T_COMMENT, T_DOC_COMMENT], true)) {
                continue;
            }

            return $token;
        }

        return null;
    }
}

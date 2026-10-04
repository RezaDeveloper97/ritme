<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema;

use App\Domain\Seo\Data\SeoImage;

/**
 * Small helpers shared by the node builders. A node is a plain `array<string, mixed>` (JSON-LD object).
 */
final class Node
{
    /**
     * Drops null, '' and empty arrays recursively (keeps 0 and false); lists stay lists.
     *
     * @param  array<array-key, mixed>  $node
     * @return array<array-key, mixed>
     */
    public static function clean(array $node): array
    {
        $isList = array_is_list($node);
        $clean = [];
        foreach ($node as $key => $value) {
            if (is_array($value)) {
                $value = self::clean($value);
            }
            if ($value === null || $value === '' || $value === []) {
                continue;
            }
            $clean[$key] = $value;
        }

        return $isList ? array_values($clean) : $clean;
    }

    /**
     * A reference to another node of the graph.
     *
     * @return array{'@id': string}
     */
    public static function ref(string $id): array
    {
        return ['@id' => $id];
    }

    /**
     * @return array<string, mixed>
     */
    public static function image(SeoImage $image, ?string $id = null): array
    {
        return self::clean([
            '@type' => 'ImageObject',
            '@id' => $id,
            'url' => $image->url,
            'contentUrl' => $image->url,
            'width' => $image->width,
            'height' => $image->height,
            'caption' => $image->alt,
            'encodingFormat' => $image->type,
        ]);
    }

    /**
     * Unique, non-empty strings, in first-seen order.
     *
     * @param  iterable<mixed>  $values
     * @return list<string>
     */
    public static function strings(iterable $values): array
    {
        $out = [];
        foreach ($values as $value) {
            if (is_string($value) && trim($value) !== '') {
                $out[] = trim($value);
            }
        }

        return array_values(array_unique($out));
    }
}

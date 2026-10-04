<?php

declare(strict_types=1);

namespace App\Support\Html;

use DOMDocument;
use DOMElement;
use DOMNode;

/**
 * Loads an HTML fragment (UTF-8, Persian-safe) into a DOM wrapped in one root element and serialises it back.
 * libxml warnings about HTML5 tags (figure, figcaption …) are swallowed.
 */
final class HtmlFragment
{
    private function __construct(public readonly DOMDocument $document, public readonly DOMElement $root) {}

    public static function load(string $html): self
    {
        $document = new DOMDocument('1.0', 'UTF-8');
        $previous = libxml_use_internal_errors(true);
        $document->loadHTML(
            '<?xml encoding="UTF-8"><div id="__rt_root">'.$html.'</div>',
            LIBXML_HTML_NOIMPLIED | LIBXML_HTML_NODEFDTD | LIBXML_NONET,
        );
        libxml_clear_errors();
        libxml_use_internal_errors($previous);

        $root = $document->getElementById('__rt_root');
        if (! $root instanceof DOMElement) {
            $root = $document->createElement('div');
            $document->appendChild($root);
        }

        return new self($document, $root);
    }

    /**
     * @return list<DOMElement>
     */
    public function elements(string ...$tags): array
    {
        $found = [];
        $wanted = array_map(strtolower(...), $tags);
        $walk = static function (DOMNode $node) use (&$walk, &$found, $wanted): void {
            foreach ($node->childNodes as $child) {
                if ($child instanceof DOMElement) {
                    if (in_array(strtolower($child->tagName), $wanted, true)) {
                        $found[] = $child;
                    }
                    $walk($child);
                }
            }
        };
        $walk($this->root);

        return $found;
    }

    public function html(): string
    {
        $html = '';
        foreach ($this->root->childNodes as $child) {
            $html .= (string) $this->document->saveHTML($child);
        }

        return trim($html);
    }
}

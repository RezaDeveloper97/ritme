<?php

declare(strict_types=1);

use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaGraph;

it('keys nodes by @id and lets later properties win', function (): void {
    $graph = (new SchemaGraph)
        ->add(['@type' => 'Thing', '@id' => 'https://ritme.test/#a', 'name' => 'اول', 'url' => 'https://ritme.test/'])
        ->add(['@id' => 'https://ritme.test/#a', 'name' => 'دوم'])
        ->add(['@type' => 'Thing', 'name' => 'بی‌نام'])
        ->add(['@type' => 'Thing', 'name' => 'بی‌نام ۲']);

    expect($graph->nodes())->toHaveCount(3)
        ->and($graph->get('https://ritme.test/#a'))->toBe(['@type' => 'Thing', '@id' => 'https://ritme.test/#a', 'name' => 'دوم', 'url' => 'https://ritme.test/'])
        ->and($graph->has('https://ritme.test/#a'))->toBeTrue();

    $graph->remove('https://ritme.test/#a');
    expect($graph->has('https://ritme.test/#a'))->toBeFalse()->and($graph->nodes())->toHaveCount(2);
});

it('puts defaults first and never overwrites what the page set', function (): void {
    $graph = (new SchemaGraph)
        ->add(['@type' => 'Product', '@id' => '#p'])
        ->add(['@id' => 'https://ritme.test/x#webpage', 'name' => 'از صفحه']);

    $graph->addDefaults([
        ['@type' => 'Organization', '@id' => 'https://ritme.test/#organization', 'name' => 'ریتمی'],
        ['@type' => 'WebPage', '@id' => 'https://ritme.test/x#webpage', 'name' => 'پیش‌فرض', 'url' => 'https://ritme.test/x'],
    ]);

    expect(array_column($graph->nodes(), '@id'))->toBe(['https://ritme.test/#organization', 'https://ritme.test/x#webpage', '#p'])
        ->and($graph->get('https://ritme.test/x#webpage'))->toMatchArray(['@type' => 'WebPage', 'name' => 'از صفحه', 'url' => 'https://ritme.test/x']);
});

it('merges into a node by full @id or by fragment', function (): void {
    $graph = (new SchemaGraph)->add(['@type' => 'WebPage', '@id' => 'https://ritme.test/x#webpage', 'name' => 'a']);

    $graph->merge('#webpage', ['about' => ['@id' => '#topic']])->merge('https://ritme.test/x#webpage', ['name' => 'b'])->merge('#none', ['x' => 1]);

    expect($graph->get('https://ritme.test/x#webpage'))->toBe([
        '@type' => 'WebPage', '@id' => 'https://ritme.test/x#webpage', 'name' => 'b', 'about' => ['@id' => '#topic'],
    ]);
});

it('drops empty values recursively but keeps zero and false', function (): void {
    expect(Node::clean(['a' => null, 'b' => '', 'c' => [], 'd' => 0, 'e' => false, 'f' => ['x' => null, 'y' => [null, 'v', '']]]))
        ->toBe(['d' => 0, 'e' => false, 'f' => ['y' => ['v']]]);
});

it('encodes valid compact JSON with Persian unescaped and no way to close the script tag', function (): void {
    $graph = (new SchemaGraph)->add([
        '@type' => 'Question', '@id' => 'https://ritme.test/faq#q', 'name' => 'پریود چیست؟',
        'text' => '<p>پاسخ</p></script><script>alert(1)</script>',
    ]);

    $json = $graph->toJson();
    $script = $graph->toScript();

    expect(json_decode($json, true, 512, JSON_THROW_ON_ERROR))->toBe($graph->toArray())
        ->and($json)->toContain('پریود چیست؟')
        ->and($json)->toContain('"@context":"https://schema.org"')
        ->and($json)->not->toContain('</script>')
        ->and($json)->not->toContain('<p>')
        ->and(substr_count($script, '<script'))->toBe(1)
        ->and($script)->toStartWith('<script type="application/ld+json">{"@context":"https://schema.org","@graph":[');
});

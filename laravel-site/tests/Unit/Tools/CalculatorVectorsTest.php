<?php

declare(strict_types=1);

use App\Domain\Content\Tools\CalculatorInput;
use App\Domain\Content\Tools\Data\JalaliDay;
use App\Domain\Content\Tools\DueDateCalculator;
use App\Domain\Content\Tools\Enums\CalculatorKind;
use App\Domain\Content\Tools\FertilityWindowCalculator;
use App\Domain\Content\Tools\ResultText;
use App\Domain\Content\Tools\ToolsCalculators;
use App\Support\Jalali\JalaliCalendar;

/*
 * The vectors shared with `node --test tests/js` (resources/js/lib/jalali.js): same inputs → same words on both sides.
 */

/** @return array<string, mixed> */
function toolsVectors(): array
{
    $json = (string) file_get_contents(dirname(__DIR__, 2).'/js/fixtures/calculator-vectors.json');

    return json_decode($json, true, 512, JSON_THROW_ON_ERROR);
}

function toolsDay(string $iso): JalaliDay
{
    [$y, $m, $d] = array_map('intval', explode('-', $iso));

    return JalaliDay::of($y, $m, $d);
}

it('keeps lang/fa/tools.php `text` equal to the shared templates', function (): void {
    $lang = require dirname(__DIR__, 3).'/lang/fa/tools.php';

    expect($lang['text'])->toBe(toolsVectors()['templates'])
        ->and(array_keys($lang['text']))->toEqualCanonicalizing(ResultText::KEYS);
});

it('converts the calendar vectors (incl. 30 Esfand of leap years 1403 and 1408)', function (): void {
    foreach (toolsVectors()['calendar'] as $v) {
        [$jy, $jm, $jd] = array_map('intval', explode('-', $v['jalali']));
        [$gy, $gm, $gd] = array_map('intval', explode('-', $v['gregorian']));

        expect(JalaliCalendar::toGregorian($jy, $jm, $jd))->toBe([$gy, $gm, $gd])
            ->and(JalaliCalendar::toJalali($gy, $gm, $gd))->toBe([$jy, $jm, $jd])
            ->and(JalaliCalendar::toDayNumber($jy, $jm, $jd))->toBe($v['jdn'])
            ->and(JalaliCalendar::fromDayNumber($v['jdn']))->toBe([$jy, $jm, $jd]);
    }

    foreach (toolsVectors()['leapYears'] as $year => $leap) {
        expect(JalaliCalendar::isLeapYear((int) $year))->toBe($leap);
    }
    expect(JalaliCalendar::isLeapYear(1403))->toBeTrue()->and(JalaliCalendar::isLeapYear(1408))->toBeTrue();
});

it('produces the expected result or error for every calculation vector', function (): void {
    $vectors = toolsVectors();
    $text = new ResultText($vectors['templates']);
    $calculators = new ToolsCalculators(new DueDateCalculator, new FertilityWindowCalculator);

    foreach ($vectors['calculations'] as $v) {
        $form = $calculators->evaluate(CalculatorKind::from($v['kind']), $v['lmp'], $v['cycle'], $text, $v['today'] ? toolsDay($v['today']) : null);
        $actual = $form->error ? ['error' => $form->error->value] : ['value' => $form->value, 'detail' => $form->detail];

        expect($actual)->toBe($v['expect'], "{$v['kind']} {$v['lmp']} {$v['cycle']} {$v['today']}");
    }
});

it('parses Persian, Arabic-Indic and Latin dates and cycle lengths', function (): void {
    foreach (['۱۴۰۵/۰۲/۱۲', '1405-2-12', '1405.02.12', '۱۲ اردیبهشت ۱۴۰۵', ' ۱۲  ارديبهشت ۱۴۰۵ ', '١٤٠٥/٠٢/١٢'] as $text) {
        expect(CalculatorInput::parseDate($text)?->toIso())->toBe('1405-02-12');
    }
    foreach (['', 'hello', '1405/2', '1404/12/30', '1299/12/01', '1500/01/01', '1405/07/31'] as $text) {
        expect(CalculatorInput::parseDate($text))->toBeNull();
    }

    expect(CalculatorInput::parseCycle('۲۸ روز'))->toBe(28)
        ->and(CalculatorInput::parseCycle(null))->toBe(28)
        ->and(CalculatorInput::parseCycle('21'))->toBe(21)
        ->and(CalculatorInput::parseCycle('45'))->toBe(45)
        ->and(CalculatorInput::parseCycle('20'))->toBeNull()
        ->and(CalculatorInput::parseCycle('46'))->toBeNull()
        ->and(CalculatorInput::parseCycle('0028'))->toBeNull();
});

it('applies Naegele with the cycle adjustment and the calendar fertile window', function (): void {
    $lmp = JalaliDay::of(1405, 2, 12);

    $due = (new DueDateCalculator)->calculate($lmp, 28, JalaliDay::of(1405, 7, 12));
    expect($due->due->number - $lmp->number)->toBe(280)
        ->and($due->weeks)->toBe(22)->and($due->days)->toBe(1)
        ->and((new DueDateCalculator)->calculate($lmp, 35)->due->number - $lmp->number)->toBe(287)
        ->and((new DueDateCalculator)->calculate($lmp, 28)->weeks)->toBeNull();

    $fert = (new FertilityWindowCalculator)->calculate($lmp, 30);
    expect($fert->ovulation->number - $lmp->number)->toBe(16)
        ->and($fert->windowFrom->number - $lmp->number)->toBe(11)
        ->and($fert->nextPeriod->number - $lmp->number)->toBe(30);
});

it('builds both cards: the submitted one from the query, the other from its example', function (): void {
    $vectors = toolsVectors();
    $calculators = new ToolsCalculators(new DueDateCalculator, new FertilityWindowCalculator);
    $examples = ['due' => ['lmp' => '۱۲ اردیبهشت ۱۴۰۵', 'cycle' => '۲۸ روز'], 'fert' => ['lmp' => '۲ مهر ۱۴۰۵', 'cycle' => '۲۹ روز']];

    $forms = $calculators->forms(['calc' => 'fert', 'lmp' => '1405/13/01', 'cycle' => '28'], $examples, new ResultText($vectors['templates']), JalaliDay::of(1405, 7, 12));

    expect($forms['due']->submitted)->toBeFalse()
        ->and($forms['due']->value)->toBe('۱۷ بهمن ۱۴۰۵')
        ->and($forms['due']->detail)->not->toContain('هفته')
        ->and($forms['fert']->submitted)->toBeTrue()
        ->and($forms['fert']->error?->value)->toBe('invalid_date')
        ->and($forms['fert']->hasResult())->toBeFalse()
        ->and(ToolsCalculators::submitted('due'))->toBeTrue()
        ->and(ToolsCalculators::submitted(['due']))->toBeFalse()
        ->and(ToolsCalculators::submitted('x'))->toBeFalse();
});

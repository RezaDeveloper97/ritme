<?php

// Dumps every Laravel enum (values, labels, descriptions, icons, options and the per-case result of
// every argument-free instance method) plus sampled static helpers as JSON, for the Go parity test.
// No framework boot: a tiny PSR-4 loader maps App\ onto backend/app.
//
//   php internal/enums/testdata/dump_enums.php ../backend > internal/enums/testdata/php_enums.json

$backend = rtrim($argv[1] ?? __DIR__.'/../../../../backend', '/');

spl_autoload_register(function (string $class) use ($backend): void {
    if (str_starts_with($class, 'App\\')) {
        $file = $backend.'/app/'.str_replace('\\', '/', substr($class, 4)).'.php';
        if (is_file($file)) {
            require_once $file;
        }
    }
});

$dirs = ['app/Enums' => 'App\\Enums\\', 'app/Services/MessageSystem/Enums' => 'App\\Services\\MessageSystem\\Enums\\'];
$locales = ['fa', 'en', 'de'];

$norm = function ($v) use (&$norm) {
    if ($v instanceof BackedEnum) {
        return $v->value;
    }
    if (is_array($v)) {
        return array_map($norm, $v);
    }

    return $v;
};

$out = ['enums' => [], 'static' => []];

foreach ($dirs as $dir => $ns) {
    $files = glob($backend.'/'.$dir.'/*.php');
    sort($files);
    foreach ($files as $file) {
        $name = basename($file, '.php');
        $class = $ns.$name;
        $ref = new ReflectionEnum($class);
        $cases = $class::cases();
        $e = ['values' => array_map(fn ($c) => $c->value, $cases), 'methods' => [], 'per_case' => [], 'options' => null];

        foreach ($ref->getMethods(ReflectionMethod::IS_PUBLIC) as $m) {
            if ($m->isStatic() || $m->getDeclaringClass()->getName() !== $class) {
                continue;
            }
            $mn = $m->getName();
            if (in_array($mn, ['label', 'description'], true)) {
                foreach ($locales as $loc) {
                    // PHP silently accepts the extra argument for label() without a $locale parameter.
                    $e['methods'][$mn][$loc] = array_map(fn ($c) => $c->$mn($loc), $cases);
                }
            } elseif ($m->getNumberOfRequiredParameters() === 0) {
                $e['per_case'][$mn] = array_map(fn ($c) => $norm($c->$mn()), $cases);
            }
        }
        if (method_exists($class, 'options')) {
            foreach ($locales as $loc) {
                $e['options'][$loc] = $class::options($loc);
            }
        }
        // Empty maps must stay JSON objects, not [].
        $e['methods'] = (object) $e['methods'];
        $e['per_case'] = (object) $e['per_case'];
        $out['enums'][$name] = $e;
    }
}

$E = 'App\\Enums\\';
$s = &$out['static'];
foreach ([0, 3.9, 4, 4.0001, 5.5, 7, 7.01, 12] as $v) {
    $s['CycleVariability::fromStdDev'][] = [$v, ($E.'CycleVariability')::fromStdDev($v)->value];
}
foreach ([[], [28], [28, 30], [28, 29, 30], [25, 32, 28], [25, 33, 28], [40, 21, 30, 28], [30, 30, 30, 30]] as $v) {
    $s['RegularityStatus::fromCycleLengths'][] = [$v, ($E.'RegularityStatus')::fromCycleLengths($v)->value];
}
foreach ([10, 18.49, 18.5, 24.99, 25, 29.99, 30, 45] as $v) {
    $s['BmiCategory::fromBmi'][] = [$v, ($E.'BmiCategory')::fromBmi($v)->value];
}
$s['CycleSubphase::contentBacked'] = $norm(($E.'CycleSubphase')::contentBacked());
$s['CyclePhase::allSubphases'] = $norm(($E.'CyclePhase')::allSubphases());
foreach ([null, 'menstruation', 'follicular', 'ovulation', 'luteal', 'bogus'] as $v) {
    $s['CyclePhase::subphaseValuesFor'][] = [$v, ($E.'CyclePhase')::subphaseValuesFor($v)];
}
foreach (['CyclePhase', 'CycleSubphase', 'RecommendationTrigger', 'RecommendationType'] as $enum) {
    foreach ([null, '', 'bogus', ...($E.$enum)::values()] as $v) {
        foreach (['fa', 'en'] as $loc) {
            $s[$enum.'::labelFor'][] = [$v, $loc, ($E.$enum)::labelFor($v, $loc)];
        }
    }
}
foreach ([null, '', 'bogus', ...($E.'RecommendationType')::values()] as $v) {
    $s['RecommendationType::iconFor'][] = [$v, ($E.'RecommendationType')::iconFor($v)];
}
$s['RecommendationTrigger::activeFor(null)'] = ($E.'RecommendationTrigger')::activeFor(null);

echo json_encode($out, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_PRESERVE_ZERO_FRACTION), "\n";

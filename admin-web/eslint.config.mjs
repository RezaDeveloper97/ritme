import { dirname } from 'path';
import { fileURLToPath } from 'url';
import { FlatCompat } from '@eslint/eslintrc';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const compat = new FlatCompat({ baseDirectory: __dirname });

// Same base as frontend/ (next/core-web-vitals + next/typescript), plus the
// two rules that frontend/CLAUDE.md enforces with its own scripts: no static
// style props and no hex colours outside globals.css (see docs/go-migration/admin-web.md).
const eslintConfig = [
  { ignores: ['.next/**', 'node_modules/**', 'next-env.d.ts'] },
  ...compat.extends('next/core-web-vitals', 'next/typescript'),
  {
    files: ['src/**/*.{ts,tsx}'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'error',
      'no-restricted-syntax': [
        'error',
        {
          selector: 'Literal[value=/^#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/]',
          message: 'No hex colours in src/: use a token from src/app/globals.css.',
        },
        {
          selector: 'JSXAttribute[name.name="style"] > JSXExpressionContainer > ObjectExpression',
          message: 'No static style props: put the rule in a class (only data-driven values may be inline, via a variable).',
        },
      ],
    },
  },
];

export default eslintConfig;

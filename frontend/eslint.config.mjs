import { dirname } from 'path';
import { fileURLToPath } from 'url';
import { FlatCompat } from '@eslint/eslintrc';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const compat = new FlatCompat({
  baseDirectory: __dirname,
});

const eslintConfig = [
  {
    // `sample/` is a standalone design prototype (plain HTML/CSS/JS), not app
    // source — kept for reference but excluded from linting and the build.
    ignores: ['.next/**', 'node_modules/**', 'next-env.d.ts', 'sample/**'],
  },
  ...compat.extends('next/core-web-vitals', 'next/typescript'),
];

export default eslintConfig;

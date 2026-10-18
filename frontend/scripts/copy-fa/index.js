// Directory entry so `node --test scripts/copy-fa` works on Node >= 22, where a directory
// argument is resolved as a module path instead of being scanned for *.test.mjs files.
import('./copy-fa.test.mjs');

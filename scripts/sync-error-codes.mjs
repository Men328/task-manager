import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(here, '..');
const source = resolve(repoRoot, 'common/errorcode/error_codes.json');
const target = resolve(repoRoot, 'frontend/src/config/error_codes.json');
const checkOnly = process.argv.includes('--check');

let canonical;
try {
  canonical = readFileSync(source, 'utf8');
} catch (cause) {
  console.error(`error-codes: không đọc được ${source}: ${cause.message}`);
  process.exit(2);
}

try {
  JSON.parse(canonical);
} catch (cause) {
  console.error(`error-codes: ${source} không phải JSON hợp lệ: ${cause.message}`);
  process.exit(2);
}

const content = canonical.endsWith('\n') ? canonical : `${canonical}\n`;

if (checkOnly) {
  let current = '';
  try {
    current = readFileSync(target, 'utf8');
  } catch {
    current = '';
  }
  if (current !== content) {
    console.error(
      'error-codes: mirror frontend/src/config/error_codes.json đã lệch. Chạy: make error-codes',
    );
    process.exit(1);
  }
  console.log('error-codes: OK');
} else {
  mkdirSync(dirname(target), { recursive: true });
  writeFileSync(target, content);
  console.log(`error-codes: đã ghi ${target}`);
}

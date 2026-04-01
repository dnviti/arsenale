import { cpSync, existsSync, mkdirSync } from 'fs';
import { dirname, resolve } from 'path';
import { fileURLToPath } from 'url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const clientDir = resolve(scriptDir, '..');
const sourceDir = resolve(clientDir, '../node_modules/monaco-editor/min/vs');
const targetDir = resolve(clientDir, 'public/monaco/vs');

if (!existsSync(sourceDir)) {
  console.error(`Monaco source assets not found at ${sourceDir}`);
  process.exit(1);
}

mkdirSync(dirname(targetDir), { recursive: true });
// Keep this copy idempotent and safe under concurrent build/dev invocations.
mkdirSync(targetDir, { recursive: true });
cpSync(sourceDir, targetDir, { recursive: true, force: true });

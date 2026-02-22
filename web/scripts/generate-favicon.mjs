#!/usr/bin/env node
/**
 * Generates favicon sizes from public/logo.png.
 * Run: node scripts/generate-favicon.mjs
 * Requires: npm install sharp --save-dev
 */
import sharp from 'sharp';
import { readFileSync, existsSync } from 'fs';
import { dirname, join } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = join(__dirname, '..');
const logoPath = join(root, 'public', 'logo.png');
const publicDir = join(root, 'public');

if (!existsSync(logoPath)) {
  console.error('public/logo.png not found');
  process.exit(1);
}

const buffer = readFileSync(logoPath);
const sizes = [16, 32, 48, 96, 180, 192];

await Promise.all(
  sizes.map((size) =>
    sharp(buffer)
      .resize(size, size)
      .png()
      .toFile(join(publicDir, size === 180 ? 'apple-touch-icon.png' : `favicon-${size}x${size}.png`))
      .then(() => console.log(`Wrote public/${size === 180 ? 'apple-touch-icon.png' : `favicon-${size}x${size}.png`}`))
  )
);

console.log('Favicons generated.');

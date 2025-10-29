// @ts-nocheck
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import sharp from 'sharp';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const svgPath = path.join(__dirname, '..', 'static', 'og-image.svg');
const pngPath = path.join(__dirname, '..', 'static', 'og-image.png');

try {
  const svg = await fs.readFile(svgPath);
  const image = sharp(svg, { density: 300 });
  await image.png({ compressionLevel: 9 }).toFile(pngPath);
  console.log('Generated', pngPath);
} catch (e) {
  console.warn('gen-og: failed to generate PNG from SVG:', e?.message);
}

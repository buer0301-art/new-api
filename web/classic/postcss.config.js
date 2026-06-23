/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import fs from 'fs';
import path from 'path';
import { createRequire } from 'module';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const require = createRequire(import.meta.url);

const resolveTailwindPlugin = () => {
  try {
    const packageJsonPath = require.resolve('tailwindcss/package.json');
    const packageJson = require(packageJsonPath);
    if (packageJson.version?.startsWith('3.')) {
      return require('tailwindcss');
    }
  } catch {
    // Fall back to the Bun workspace package store below.
  }

  const bunStoreDir = path.resolve(__dirname, '../node_modules/.bun');
  const tailwind3Dir = fs
    .readdirSync(bunStoreDir)
    .find((name) => name.startsWith('tailwindcss@3.'));
  if (tailwind3Dir) {
    return require(
      path.join(bunStoreDir, tailwind3Dir, 'node_modules/tailwindcss'),
    );
  }

  return require('tailwindcss');
};

export default {
  plugins: [resolveTailwindPlugin(), require('autoprefixer')()],
};

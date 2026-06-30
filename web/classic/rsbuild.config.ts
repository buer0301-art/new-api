import path from 'path'
import fs from 'fs'
import { createRequire } from 'module'
import { fileURLToPath } from 'url'
import { defineConfig, loadEnv } from '@rsbuild/core'
import { pluginReact } from '@rsbuild/plugin-react'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const require = createRequire(import.meta.url)
const semiUiDir = fs.realpathSync(
  path.resolve(path.dirname(require.resolve('@douyinfe/semi-ui')), '../..'),
)
const semiDateFnsDir = path.dirname(
  require.resolve('date-fns/package.json', { paths: [semiUiDir] }),
)
const packageNameFromSpecifier = (specifier: string) =>
  specifier.startsWith('@')
    ? specifier.split('/').slice(0, 2).join('/')
    : specifier.split('/')[0]
const resolvePackageDir = (
  specifier: string,
  options?: { paths?: string[] },
) => {
  const packageName = packageNameFromSpecifier(specifier)
  let currentDir = fs.realpathSync(
    path.dirname(require.resolve(specifier, options)),
  )

  while (true) {
    const packageJsonPath = path.join(currentDir, 'package.json')
    if (fs.existsSync(packageJsonPath)) {
      const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'))
      if (packageJson.name === packageName) {
        return currentDir
      }
    }

    const parentDir = path.dirname(currentDir)
    if (parentDir === currentDir) {
      throw new Error(`Unable to resolve package directory for ${specifier}`)
    }
    currentDir = parentDir
  }
}
const vchartDir = resolvePackageDir('@visactor/vchart')
const resolveVisActorVChartDep = (specifier: string) =>
  fs.existsSync(path.join(vchartDir, 'node_modules', specifier))
    ? fs.realpathSync(path.join(vchartDir, 'node_modules', specifier))
    : resolvePackageDir(specifier, { paths: [vchartDir] })
const classicVisActorAlias = {
  '@visactor/react-vchart': resolvePackageDir('@visactor/react-vchart'),
  '@visactor/vchart': resolvePackageDir('@visactor/vchart'),
  '@visactor/vchart-semi-theme': resolvePackageDir(
    '@visactor/vchart-semi-theme',
  ),
  '@visactor/vchart-theme-utils': resolvePackageDir(
    '@visactor/vchart-theme-utils',
  ),
  '@visactor/vdataset': resolveVisActorVChartDep('@visactor/vdataset'),
  '@visactor/vrender-components': resolveVisActorVChartDep(
    '@visactor/vrender-components',
  ),
  '@visactor/vrender-core': resolveVisActorVChartDep(
    '@visactor/vrender-core',
  ),
  '@visactor/vrender-kits': resolveVisActorVChartDep(
    '@visactor/vrender-kits',
  ),
  '@visactor/vscale': resolveVisActorVChartDep('@visactor/vscale'),
  '@visactor/vutils': resolveVisActorVChartDep('@visactor/vutils'),
  '@visactor/vutils-extension': resolveVisActorVChartDep(
    '@visactor/vutils-extension',
  ),
}

export default defineConfig(({ envMode }) => {
  const env = loadEnv({ mode: envMode, prefixes: ['VITE_'] })
  const clientServerUrl =
    process.env.VITE_REACT_APP_SERVER_URL ||
    env.rawPublicVars.VITE_REACT_APP_SERVER_URL ||
    ''
  const proxyServerUrl =
    clientServerUrl ||
    'http://localhost:3000'
  const isProd = envMode === 'production'
  const devProxy = Object.fromEntries(
    (['/api', '/mj', '/pg'] as const).map((key) => [
      key,
      { target: proxyServerUrl, changeOrigin: true },
    ]),
  ) as Record<string, { target: string; changeOrigin: boolean }>

  return {
    plugins: [pluginReact()],
    source: {
      entry: {
        index: './src/index.jsx',
      },
      define: {
        'import.meta.env.VITE_REACT_APP_SERVER_URL': JSON.stringify(
          clientServerUrl,
        ),
      },
    },
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
        '@douyinfe/semi-ui/dist/css/semi.css': path.resolve(
          semiUiDir,
          'dist/css/semi.css',
        ),
        'date-fns': semiDateFnsDir,
        ...classicVisActorAlias,
      },
    },
    html: {
      template: './index.html',
    },
    server: {
      host: '0.0.0.0',
      strictPort: true,
      proxy: devProxy,
    },
    output: {
      minify: isProd,
      target: 'web',
      distPath: {
        root: 'dist',
      },
    },
    performance: {
      removeConsole: isProd ? ['log'] : false,
      buildCache: {
        cacheDigest: [process.env.VITE_REACT_APP_VERSION],
      },
    },
    tools: {
      rspack: {
        module: {
          rules: [
            {
              test: /src[\\/].*\.js$/,
              type: 'javascript/auto',
              use: [
                {
                  loader: 'builtin:swc-loader',
                  options: {
                    jsc: {
                      parser: {
                        syntax: 'ecmascript',
                        jsx: true,
                      },
                      transform: {
                        react: {
                          runtime: 'automatic',
                          development: !isProd,
                          refresh: !isProd,
                        },
                      },
                    },
                  },
                },
              ],
            },
          ],
        },
      },
    },
  }
})

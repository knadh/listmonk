import vue from '@vitejs/plugin-vue2';
import { defineConfig, loadEnv } from 'vite';
import path from 'path';
import { createRequire } from 'module';
import { fileURLToPath } from 'url';

const require = createRequire(import.meta.url);
const __dirname = path.dirname(fileURLToPath(import.meta.url));

const injectCustomAssets = () => {
  return {
    name: 'inject-custom-assets',
    transformIndexHtml(html) {
      return html.replace(
        '</title>',
        `</title>\n    <link href="/admin/custom.css?v=asset_version" rel="stylesheet" type="text/css">\n    <script src="/admin/custom.js?v=asset_version" async defer></script>`
      );
    },
  };
};

// https://vitejs.dev/config/
export default defineConfig(({ _, mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  return {
    plugins: [
      vue(),
      injectCustomAssets(),
    ],
    define: {
      global: 'window',
    },
    base: '/admin',
    mode,
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
        bulma: require.resolve('bulma/bulma.sass'),
      },
    },
    build: {
      assetsDir: 'static',
    },
    server: {
      port: env.LISTMONK_FRONTEND_PORT || 8080,
      proxy: {
        '^/$': {
          target: env.LISTMONK_API_URL || 'http://127.0.0.1:9000',
        },
        '^/(api|webhooks|subscription|public|health)': {
          target: env.LISTMONK_API_URL || 'http://127.0.0.1:9000',
        },
        '^/admin/login': {
          target: env.LISTMONK_API_URL || 'http://127.0.0.1:9000',
        },
        '^/(admin\/custom\.(css|js))': {
          target: env.LISTMONK_API_URL || 'http://127.0.0.1:9000',
        },
      },
    },
  };
});

// vite.config.js
import vue from '@vitejs/plugin-vue';
import { defineConfig, loadEnv } from 'vite';

const path = require('path');
const purgecss = require('@fullhuman/postcss-purgecss').default;

export default defineConfig(({ _, mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const isProd = mode === 'production';
  const THEME_IS_STATIC = true;

  const purge = purgecss({
    content: [
      './index.html', // Source file.
      './src/**/*.{vue,js,ts,jsx,tsx}',
      './node_modules/buefy/**/*.{js,vue}', // Let PurgeCSS scan Buefy.
    ],
    defaultExtractor: (content) => content.match(/[\w-/:%.]+(?<!:)/g) || [],

    safelist: {
      standard: [
        // states.
        'is-active', 'is-loading', 'is-selected', 'is-expanded', 'is-current',
        // colors.
        'is-primary', 'is-link', 'is-info', 'is-success', 'is-warning', 'is-danger',
        // sizes/modifiers.
        'is-small', 'is-medium', 'is-large', 'is-fullwidth', 'is-outlined', 'is-rounded',
        // components.
        'modal', 'dropdown', 'navbar', 'tabs', 'pagination', 'notification', 'message',
        'tag', 'tags', 'table', 'toast',
      ],
      deep: [/^b-/, /^fa-/, /^icon-/, /^mdi-/], // Icons.
      greedy: [/^modal/, /^dropdown/, /^navbar/, /^pagination/, /^tabs?$/, /^table/],
      keyframes: [],
      variables: [],
    },

    keyframes: false,
    fontFace: false,
    rejected: false, // Log removed stuff.
  });

  const postcssPlugins = [
    // Remove unused --vars.
    require('postcss-prune-var')(),

    // Inline remaining CSS custom properties.
    THEME_IS_STATIC && require('postcss-custom-properties')({
      preserve: false, // replace var(--x) with computed values; drop --x definitions
    }),

    isProd && purge,

    require('postcss-discard-duplicates'),
    require('postcss-discard-empty'),
    isProd && require('cssnano')({ preset: 'default' }),
  ].filter(Boolean);

  return {
    plugins: [vue()],
    base: '/admin',
    mode,
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
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
        '^/(admin\\/custom\\.(css|js))': {
          target: env.LISTMONK_API_URL || 'http://127.0.0.1:9000',
        },
      },
    },
    css: {
      postcss: {
        plugins: postcssPlugins,
      },
    },
  };
});

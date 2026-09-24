import { resolve } from 'path';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react-swc';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      // Make the renderer inside @usewaypoint/email-builder use our Text block (Markdown element styles).
      '@usewaypoint/block-text': resolve(__dirname, 'src/documents/blocks/Text/index.tsx'),
    },
  },
  define: {
    'process.env.NODE_ENV': '"production"',
  },
  build: {
    lib: {
      entry: resolve(__dirname, 'src/main.tsx'),
      name: 'EmailBuilder',
      formats: ['umd'],
      fileName: (format) => `email-builder.${format}.js`,
    },
    minify: 'terser',
    cssCodeSplit: true,
    cssMinify: true,

    // Option to externalize deps.
    // rollupOptions: {
    //   external: ['react', 'react-dom'],
    //   output: {
    //     globals: {
    //       react: 'React',
    //       'react-dom': 'ReactDOM',
    //     },
    //   },
    // }
  },
});

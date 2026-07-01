import { defineConfig } from 'vite';

export default defineConfig({
  base: '/player/',
  build: {
    emptyOutDir: true,
    outDir: '../../static/player'
  }
});


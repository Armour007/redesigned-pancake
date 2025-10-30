import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import path from 'node:path';

// Vitest config lives here so we can use the Vite pipeline for Svelte components
// This enables jsdom environment and a global test setup.

export default defineConfig(() => {
	const disableSvelteHotUpdateInVitest = {
		name: 'aura:disable-svelte-hot-update-in-vitest',
		enforce: 'pre',
		configResolved(resolved) {
			// Vitest sets this env var; guard to avoid impacting normal dev/build
			if (process.env.VITEST) {
				const plugins = (resolved.plugins as unknown as any[]);
				const toStrip = new Set([
					'vite-plugin-svelte:hot-update',
					'vite-plugin-svelte:load-custom'
				]);
				for (let i = plugins.length - 1; i >= 0; i--) {
					const name = plugins[i]?.name;
					if (name && toStrip.has(name)) {
						plugins.splice(i, 1);
					}
				}
			}
		}
	} as import('vite').Plugin;
	return {
		plugins: [disableSvelteHotUpdateInVitest, sveltekit()],
		// Ensure HMR config is always an object so plugin code doesn't crash under Vitest
		server: { hmr: { overlay: false } },
		resolve: process.env.VITEST
			? {
			    alias: {
			      '$app/stores': path.resolve(process.cwd(), 'src/test-stubs/app-stores.ts')
			    }
			  }
			: undefined,
		// @ts-ignore - Vitest options are recognized when Vitest is installed
		test: {
			globals: true,
			environment: 'jsdom',
			setupFiles: ['src/setupTests.ts'],
			css: true,
			include: ['src/__tests__/**/*.spec.ts'],
			exclude: ['tests-e2e/**'],
			// Run unit tests in real browser to avoid SSR mount issues in Svelte 5
			browser: { enabled: true, provider: 'playwright', name: 'chromium' },
			// Avoid worker/thread weirdness with some Svelte internals in tests
			poolOptions: {
				threads: { singleThread: true }
			},
			coverage: {
				reporter: ['text', 'html', 'lcov']
			}
		}
	};
});

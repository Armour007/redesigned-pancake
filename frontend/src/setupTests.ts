import { beforeAll, afterAll, afterEach } from 'vitest';
import '@testing-library/jest-dom';
import { installFetchMock } from './test-utils/fetchMock';
import { cleanup as tlCleanup } from '@testing-library/svelte';

// Start MSW in either browser (worker) or node (server) depending on the runtime
let cleanup: (() => void) | undefined;

beforeAll(async () => {
	// In real browser runs, navigator.serviceWorker exists. In jsdom it does not.
	if (typeof window !== 'undefined' && typeof navigator !== 'undefined' && 'serviceWorker' in navigator) {
		// In Vitest browser mode, Service Workers may not control nested contexts reliably.
		// Use a lightweight fetch mock for API requests instead of MSW worker.
		cleanup = installFetchMock();
	} else {
		const { server } = await import('./mocks/server');
		server.listen({ onUnhandledRequest: 'warn' });
		cleanup = () => server.close();
	}
});

afterEach(async () => {
	// Clear local storage between tests to avoid cross-test coupling
	try { localStorage.clear(); } catch { /* ignore */ }
	// Ensure Testing Library cleans up the DOM between tests in browser mode
	try { tlCleanup(); } catch { /* ignore */ }
	if (typeof window !== 'undefined' && typeof navigator !== 'undefined' && 'serviceWorker' in navigator) {
		// no-op; using fetch mock
	} else {
		const { server } = await import('./mocks/server');
		server.resetHandlers();
	}
});

afterAll(() => {
	cleanup?.();
});

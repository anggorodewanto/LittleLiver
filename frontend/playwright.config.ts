import { defineConfig } from '@playwright/test';

export default defineConfig({
	testDir: './e2e',
	timeout: 30000,
	expect: {
		timeout: 10000
	},
	fullyParallel: false,
	retries: 0,
	workers: 1,
	reporter: 'list',
	globalSetup: './e2e/global-setup.ts',
	use: {
		baseURL: 'http://localhost:3847',
		trace: 'on-first-retry'
	},
	projects: [
		{
			name: 'chromium',
			use: {
				browserName: 'chromium'
			}
		}
	]
});

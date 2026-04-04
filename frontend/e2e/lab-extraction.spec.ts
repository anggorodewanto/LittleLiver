import { test, expect } from '@playwright/test';
import { testLogin, getCsrfToken } from './helpers';
import { join } from 'path';
import { mkdtempSync } from 'fs';
import { tmpdir } from 'os';
import { execSync } from 'child_process';

const MOCK_CLAUDE_URL = 'http://localhost:3848';

async function resetMockClaude(): Promise<void> {
	await fetch(`${MOCK_CLAUDE_URL}/mock/reset`, { method: 'POST' });
}

async function configureMockClaudeError(status: number, message: string): Promise<void> {
	await fetch(`${MOCK_CLAUDE_URL}/mock/configure`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ error: { status, message } })
	});
}

/**
 * Create a valid 2x2 pixel PNG file for upload testing using ImageMagick.
 */
function createTestImage(): string {
	const dir = mkdtempSync(join(tmpdir(), 'lab-e2e-'));
	const filePath = join(dir, 'test-lab-report.png');
	execSync(`convert -size 2x2 xc:white "${filePath}"`);
	return filePath;
}

test.describe('Lab extraction E2E', () => {
	test.beforeEach(async () => {
		await resetMockClaude();
	});

	test('upload -> extract -> review -> edit -> confirm -> verify in labs list', async ({
		page
	}) => {
		// Step 1: Login and create baby
		await testLogin(page, {
			google_id: 'lab-e2e-user-1',
			email: 'lab-e2e@example.com',
			name: 'Lab E2E User'
		});

		const csrfToken = await getCsrfToken(page);
		const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

		// Create a baby
		const createBabyRes = await page.request.post('/api/babies', {
			data: {
				name: 'Lab Test Baby',
				date_of_birth: '2025-06-15',
				sex: 'female'
			},
			headers: {
				'X-CSRF-Token': csrfToken,
				'X-Timezone': timezone,
				'Content-Type': 'application/json'
			}
		});
		expect(createBabyRes.ok()).toBeTruthy();
		const baby = await createBabyRes.json();

		// Step 2: Navigate to the lab log page (where import is available)
		await page.goto('/log/lab');
		await expect(page.getByText('Lab Test Baby')).toBeVisible({ timeout: 10000 });

		// Step 3: Click "Import from photo" button
		const importButton = page.getByRole('button', { name: /import from photo/i });
		await expect(importButton).toBeVisible({ timeout: 5000 });
		await importButton.click();

		// Step 4: Upload a test image file
		const testImagePath = createTestImage();
		const fileInput = page.locator('input[type="file"]');
		await expect(fileInput).toBeAttached({ timeout: 5000 });
		await fileInput.setInputFiles(testImagePath);

		// Step 5: Wait for extraction to complete and review screen to appear
		await expect(page.getByText(/review extracted results/i)).toBeVisible({ timeout: 20000 });

		// Verify the expected values appear in the review screen
		// Use label-based selectors to find inputs by their associated test name label
		await expect(page.locator('#test-name-0')).toHaveValue('total_bilirubin');
		await expect(page.locator('#value-0')).toHaveValue('1.8');
		await expect(page.locator('#test-name-1')).toHaveValue('ALT');
		await expect(page.locator('#value-1')).toHaveValue('52');
		await expect(page.locator('#test-name-2')).toHaveValue('AST');
		await expect(page.locator('#value-2')).toHaveValue('38');

		// Verify extraction notes are shown
		await expect(page.getByText('Sample collected at Regional Hospital')).toBeVisible();

		// Step 6: Edit one value (change ALT from 52 to 55)
		const altValueInput = page.locator('#value-1');
		await altValueInput.fill('55');

		// Step 7: Confirm
		await page.getByRole('button', { name: /confirm/i }).click();

		// Step 8: Wait for save to complete (navigates to dashboard on success)
		// Wait a moment for navigation and then check via API
		await page.waitForURL('/', { timeout: 10000 });

		// Step 9: Verify saved results via the API
		const labsRes = await page.request.get(`/api/babies/${baby.id}/labs`, {
			headers: { 'X-Timezone': timezone }
		});
		expect(labsRes.ok()).toBeTruthy();
		const labsData = await labsRes.json();
		expect(labsData.data.length).toBe(3);

		// Build a map of results by test_name for verification
		const resultMap: Record<string, string> = {};
		for (const lab of labsData.data) {
			resultMap[lab.test_name] = lab.value;
		}

		expect(resultMap['total_bilirubin']).toBe('1.8');
		expect(resultMap['ALT']).toBe('55'); // edited value
		expect(resultMap['AST']).toBe('38');

		// Step 10: Navigate to labs page and verify results are visible in the list
		await page.goto('/labs');
		// total_bilirubin is displayed as "Total Bilirubin" via labTestLabel
		// Use cell role to avoid matching the filter button too
		await expect(page.getByRole('cell', { name: 'Total Bilirubin' })).toBeVisible({ timeout: 10000 });
		await expect(page.getByRole('cell', { name: 'ALT' })).toBeVisible();
		await expect(page.getByRole('cell', { name: 'AST' })).toBeVisible();
	});

	test('shows error on extraction failure and allows retry', async ({ page }) => {
		// Configure mock to return an error on first call
		await configureMockClaudeError(500, 'Internal server error');

		// Login and create baby
		await testLogin(page, {
			google_id: 'lab-e2e-user-2',
			email: 'lab-e2e2@example.com',
			name: 'Lab E2E User 2'
		});

		const csrfToken = await getCsrfToken(page);
		const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

		const createBabyRes = await page.request.post('/api/babies', {
			data: {
				name: 'Lab Error Baby',
				date_of_birth: '2025-06-15',
				sex: 'male'
			},
			headers: {
				'X-CSRF-Token': csrfToken,
				'X-Timezone': timezone,
				'Content-Type': 'application/json'
			}
		});
		expect(createBabyRes.ok()).toBeTruthy();

		// Navigate to lab log page
		await page.goto('/log/lab');
		await expect(page.getByText('Lab Error Baby')).toBeVisible({ timeout: 10000 });

		// Click import
		const importButton = page.getByRole('button', { name: /import from photo/i });
		await expect(importButton).toBeVisible({ timeout: 5000 });
		await importButton.click();

		// Upload file - should fail on extraction
		const testImagePath = createTestImage();
		const fileInput = page.locator('input[type="file"]');
		await expect(fileInput).toBeAttached({ timeout: 5000 });
		await fileInput.setInputFiles(testImagePath);

		// Should show error message
		await expect(page.getByText(/extraction failed/i)).toBeVisible({ timeout: 20000 });

		// Retry: upload again (mock auto-clears error after first use)
		const fileInput2 = page.locator('input[type="file"]');
		await fileInput2.setInputFiles(testImagePath);

		// Should now show review screen
		await expect(page.getByText(/review extracted results/i)).toBeVisible({ timeout: 20000 });
		await expect(page.locator('#test-name-0')).toHaveValue('total_bilirubin');
	});
});

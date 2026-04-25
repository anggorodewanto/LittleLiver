import { test, expect } from '@playwright/test';
import { testLogin, getCsrfToken } from './helpers';

test.describe('Care plans', () => {
	test('create plan via API → see on dashboard → edit phase → reload reflects change', async ({
		page
	}) => {
		await testLogin(page, {
			google_id: 'e2e-care-plans-user',
			email: 'careplans@example.com',
			name: 'Care Plans User'
		});

		await page.goto('/');
		await expect(page.getByText('Create a Baby')).toBeVisible();
		await page.getByText('Create a Baby').click();
		await page.fill('#baby-name', 'CP Baby');
		await page.fill('#baby-dob', '2025-06-15');
		await page.selectOption('#baby-sex', 'female');
		await page.getByRole('button', { name: 'Create Baby' }).click();
		await expect(page.getByText('CP Baby')).toBeVisible({ timeout: 10000 });

		const csrfToken = await getCsrfToken(page);
		const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

		const babiesResponse = await page.request.get('/api/babies', {
			headers: { 'X-Timezone': timezone }
		});
		const babiesData = await babiesResponse.json();
		const babyId = babiesData.babies[0].id;

		// Create a care plan whose phase 1 is well in the past so it shows up
		// as the current phase regardless of the test machine's clock.
		const planResp = await page.request.post(`/api/babies/${babyId}/care-plans`, {
			data: {
				name: 'Antibiotic Rotation',
				timezone,
				phases: [
					{ seq: 1, label: 'Cefixime', start_date: '2000-01-01' },
					{ seq: 2, label: 'Amoxicillin', start_date: '2099-01-01' }
				]
			},
			headers: {
				'X-CSRF-Token': csrfToken,
				'X-Timezone': timezone,
				'Content-Type': 'application/json'
			}
		});
		expect(planResp.ok()).toBeTruthy();
		const plan = await planResp.json();

		await page.goto('/');
		await expect(page.getByText('Care Plans')).toBeVisible({ timeout: 10000 });
		await expect(page.getByText('Antibiotic Rotation')).toBeVisible();
		await expect(page.getByText('Cefixime')).toBeVisible();

		// Edit phase 1's label via the detail page.
		await page.goto(`/care-plans/${plan.id}`);
		const phase1Label = page.getByLabel('Phase 1 label');
		await phase1Label.fill('Cefixime renamed');
		await page.getByRole('button', { name: /save plan/i }).click();

		await page.goto('/');
		await expect(page.getByText('Cefixime renamed')).toBeVisible({ timeout: 10000 });
	});
});

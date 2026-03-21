import { test, expect } from '@playwright/test';
import { testLogin, getCsrfToken } from './helpers';

test.describe('Full user journey', () => {
	test('login → baby creation → feeding entry → stool entry → dashboard verification → alert dismissal', async ({
		page
	}) => {
		// Step 1: Login via test endpoint
		await testLogin(page, {
			google_id: 'e2e-test-user-1',
			email: 'e2e@example.com',
			name: 'E2E Test User'
		});

		// Step 2: Navigate to app — should show FirstLogin (no babies)
		await page.goto('/');
		await expect(page.getByText('Welcome to LittleLiver')).toBeVisible();
		await expect(page.getByText('Create a Baby')).toBeVisible();
		await expect(page.getByText('Join with Invite Code')).toBeVisible();

		// Step 3: Create a baby
		await page.getByText('Create a Baby').click();
		await page.fill('#baby-name', 'Test Baby');
		await page.fill('#baby-dob', '2025-06-15');
		await page.selectOption('#baby-sex', 'female');
		await page.fill('#baby-diagnosis-date', '2025-07-01');
		await page.fill('#baby-kasai-date', '2025-07-10');
		await page.getByRole('button', { name: 'Create Baby' }).click();

		// After baby creation, the dashboard should load
		await expect(page.getByText('Test Baby')).toBeVisible({ timeout: 10000 });

		// Step 4: Verify the dashboard is showing (summary cards should be visible)
		await expect(page.getByText('Feeds')).toBeVisible();
		await expect(page.getByText('Calories')).toBeVisible();
		await expect(page.getByText('Wet Diapers')).toBeVisible();
		await expect(page.getByText('Stools')).toBeVisible();

		// Step 5: Log a feeding entry via the API directly (to verify dashboard updates)
		const csrfToken = await getCsrfToken(page);
		const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

		// Get baby ID from the API
		const babiesResponse = await page.request.get('/api/babies', {
			headers: { 'X-Timezone': timezone }
		});
		const babiesData = await babiesResponse.json();
		const babyId = babiesData.babies[0].id;

		// Log a formula feeding with 120mL
		const now = new Date().toISOString();
		const feedingResponse = await page.request.post(`/api/babies/${babyId}/feedings`, {
			data: {
				timestamp: now,
				feed_type: 'formula',
				volume_ml: 120,
				notes: 'E2E test feeding'
			},
			headers: {
				'X-CSRF-Token': csrfToken,
				'X-Timezone': timezone,
				'Content-Type': 'application/json'
			}
		});
		expect(feedingResponse.ok()).toBeTruthy();

		// Step 6: Log a stool entry with acholic color (color_rating=2, triggers alert)
		const stoolResponse = await page.request.post(`/api/babies/${babyId}/stools`, {
			data: {
				timestamp: now,
				color_rating: 2,
				color_label: 'clay',
				consistency: 'soft',
				volume_estimate: 'medium'
			},
			headers: {
				'X-CSRF-Token': csrfToken,
				'X-Timezone': timezone,
				'Content-Type': 'application/json'
			}
		});
		expect(stoolResponse.ok()).toBeTruthy();

		// Step 7: Reload the page so the dashboard picks up the new data
		await page.reload();
		await expect(page.getByText('Feeds')).toBeVisible({ timeout: 10000 });

		// Verify summary cards reflect the data
		// Total feeds should be 1
		const feedsCard = page.locator('.card').filter({ hasText: 'Feeds' });
		await expect(feedsCard.locator('.card-value')).toContainText('1');

		// Total stools should be 1
		const stoolsCard = page.locator('.card').filter({ hasText: 'Stools' });
		await expect(stoolsCard.locator('.card-value')).toContainText('1');

		// Step 8: Verify acholic stool alert is displayed
		await expect(page.getByText('Acholic Stool')).toBeVisible();

		// Step 9: Dismiss the alert
		const dismissButton = page
			.locator('.alert-banner')
			.filter({ hasText: 'Acholic Stool' })
			.getByRole('button', { name: 'Dismiss' });
		await dismissButton.click();

		// Alert should no longer be visible
		await expect(page.getByText('Acholic Stool')).not.toBeVisible();

		// Step 10: Reload — dismissed alert should stay dismissed (localStorage)
		await page.reload();
		await expect(page.getByText('Feeds')).toBeVisible({ timeout: 10000 });
		await expect(page.getByText('Acholic Stool')).not.toBeVisible();
	});

	test('dashboard shows correct calorie data for formula feeding', async ({ page }) => {
		// Login
		await testLogin(page, {
			google_id: 'e2e-test-user-2',
			email: 'e2e2@example.com',
			name: 'E2E User 2'
		});

		const csrfToken = await getCsrfToken(page);
		const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

		// Create a baby via API
		const createBabyResponse = await page.request.post('/api/babies', {
			data: {
				name: 'Calorie Baby',
				date_of_birth: '2025-08-01',
				sex: 'male'
			},
			headers: {
				'X-CSRF-Token': csrfToken,
				'X-Timezone': timezone,
				'Content-Type': 'application/json'
			}
		});
		expect(createBabyResponse.ok()).toBeTruthy();
		const baby = await createBabyResponse.json();

		// Log a formula feeding with 150mL (should auto-calculate ~101.4 kcal at 20kcal/oz)
		const now = new Date().toISOString();
		const feedRes = await page.request.post(`/api/babies/${baby.id}/feedings`, {
			data: {
				timestamp: now,
				feed_type: 'formula',
				volume_ml: 150
			},
			headers: {
				'X-CSRF-Token': csrfToken,
				'X-Timezone': timezone,
				'Content-Type': 'application/json'
			}
		});
		expect(feedRes.ok()).toBeTruthy();

		// Navigate to home page
		await page.goto('/');
		await expect(page.getByText('Calorie Baby')).toBeVisible({ timeout: 10000 });

		// Verify calories card shows non-zero value
		const caloriesCard = page.locator('.card').filter({ hasText: 'Calories' });
		const calorieValue = await caloriesCard.locator('.card-value').textContent();
		const calories = parseFloat(calorieValue ?? '0');
		expect(calories).toBeGreaterThan(0);
	});

	test('stool color trend dots appear after logging stools', async ({ page }) => {
		// Login
		await testLogin(page, {
			google_id: 'e2e-test-user-3',
			email: 'e2e3@example.com',
			name: 'E2E User 3'
		});

		const csrfToken = await getCsrfToken(page);
		const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

		// Create a baby
		const createBabyResponse = await page.request.post('/api/babies', {
			data: {
				name: 'Stool Trend Baby',
				date_of_birth: '2025-09-01',
				sex: 'female'
			},
			headers: {
				'X-CSRF-Token': csrfToken,
				'X-Timezone': timezone,
				'Content-Type': 'application/json'
			}
		});
		const baby = await createBabyResponse.json();

		// Log stools for the last few days with varying colors
		for (let daysAgo = 0; daysAgo < 3; daysAgo++) {
			const date = new Date();
			date.setDate(date.getDate() - daysAgo);
			const colorRating = daysAgo + 5; // ratings 5, 6, 7
			await page.request.post(`/api/babies/${baby.id}/stools`, {
				data: {
					timestamp: date.toISOString(),
					color_rating: colorRating,
					color_label: ['light_green', 'green', 'brown'][daysAgo]
				},
				headers: {
					'X-CSRF-Token': csrfToken,
					'X-Timezone': timezone,
					'Content-Type': 'application/json'
				}
			});
		}

		// Navigate to page
		await page.goto('/');
		await expect(page.getByTestId('active-baby-name')).toHaveText('Stool Trend Baby', { timeout: 10000 });

		// Verify stool color trend section appears
		await expect(page.getByText('Stool Color Trend (7 days)')).toBeVisible();

		// Verify trend dots are rendered
		const dots = page.locator('.stool-trend-dot');
		const dotCount = await dots.count();
		expect(dotCount).toBeGreaterThanOrEqual(3);
	});
});

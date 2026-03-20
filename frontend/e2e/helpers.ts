import { type Page } from '@playwright/test';

/**
 * Log in via the test login endpoint (bypasses Google OAuth).
 * Sets the session cookie on the page's browser context.
 */
export async function testLogin(
	page: Page,
	user: { google_id: string; email: string; name: string }
): Promise<{ user_id: string; session_id: string }> {
	const response = await page.request.post('/api/test/login', {
		data: user
	});

	if (!response.ok()) {
		throw new Error(`Test login failed: ${response.status()} ${await response.text()}`);
	}

	return response.json();
}

/**
 * Fetch CSRF token from the server (requires an active session).
 */
export async function getCsrfToken(page: Page): Promise<string> {
	const response = await page.request.get('/api/csrf-token');
	if (!response.ok()) {
		throw new Error(`CSRF token fetch failed: ${response.status()}`);
	}
	const data = await response.json();
	return data.csrf_token;
}

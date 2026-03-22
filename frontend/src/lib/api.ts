const BASE_URL = '/api';

let csrfToken: string | null = null;

const userTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

async function fetchCsrfToken(): Promise<string> {
	if (csrfToken) {
		return csrfToken;
	}
	const response = await fetch('/api/csrf-token', { credentials: 'include' });
	if (!response.ok) {
		throw new Error(`Failed to fetch CSRF token: ${response.status}`);
	}
	const data = await response.json();
	csrfToken = data.csrf_token as string;
	return csrfToken;
}

function clearCsrfToken(): void {
	csrfToken = null;
}

/** Reset CSRF token cache. Exported for testing only. */
export function _resetCsrfToken(): void {
	csrfToken = null;
}

const STATE_CHANGING_METHODS = ['POST', 'PUT', 'DELETE', 'PATCH'];

async function buildRequest(path: string, options?: RequestInit): Promise<Response> {
	const method = options?.method ?? 'GET';
	const headers: Record<string, string> = {
		'X-Timezone': userTimezone,
		...(options?.headers as Record<string, string>)
	};

	if (options?.body && typeof options.body === 'string') {
		headers['Content-Type'] = 'application/json';
	}

	if (STATE_CHANGING_METHODS.includes(method.toUpperCase())) {
		const token = await fetchCsrfToken();
		headers['X-CSRF-Token'] = token;
	}

	const response = await fetch(`${BASE_URL}${path}`, {
		credentials: 'include',
		...options,
		headers
	});

	if (!response.ok) {
		if (response.status === 401) {
			clearCsrfToken();
			window.location.href = '/login';
		}
		throw new Error(`API error: ${response.status}`);
	}

	return response;
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
	const response = await buildRequest(path, options);

	if (response.status === 204) {
		return undefined as T;
	}

	return response.json() as Promise<T>;
}

interface HealthCheckResponse {
	status: string;
}

export const apiClient = {
	healthCheck(): Promise<HealthCheckResponse> {
		return request<HealthCheckResponse>('/health');
	},

	get<T>(path: string): Promise<T> {
		return request<T>(path);
	},

	post<T>(path: string, body: unknown): Promise<T> {
		return request<T>(path, {
			method: 'POST',
			body: JSON.stringify(body)
		});
	},

	put<T>(path: string, body: unknown): Promise<T> {
		return request<T>(path, {
			method: 'PUT',
			body: JSON.stringify(body)
		});
	},

	del<T>(path: string): Promise<T> {
		return request<T>(path, {
			method: 'DELETE'
		});
	},

	postForm<T>(path: string, formData: FormData): Promise<T> {
		return request<T>(path, {
			method: 'POST',
			body: formData
		});
	},

	getRaw(path: string): Promise<Response> {
		return buildRequest(path);
	}
};

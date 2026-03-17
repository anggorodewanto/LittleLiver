const BASE_URL = '/api';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
	const response = await fetch(`${BASE_URL}${path}`, {
		credentials: 'include',
		...options,
		headers: {
			'Content-Type': 'application/json',
			...options?.headers
		}
	});

	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}

	return response.json() as Promise<T>;
}

interface HealthCheckResponse {
	status: string;
}

export const apiClient = {
	healthCheck(): Promise<HealthCheckResponse> {
		return request<HealthCheckResponse>('/health');
	}
};

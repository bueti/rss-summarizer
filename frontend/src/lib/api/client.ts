import { env } from '$env/dynamic/public';

const API_BASE = env.PUBLIC_API_URL || import.meta.env.VITE_API_URL || 'http://localhost:8080';

export type ErrorType<Error> = Error;

export const customFetch = async <T>(
	url: string,
	options: RequestInit
): Promise<T> => {
	const fullUrl = `${API_BASE}${url}`;
	console.log('[API] Fetching:', fullUrl, 'API_BASE:', API_BASE);

	const response = await fetch(fullUrl, {
		...options,
		credentials: 'include', // Enable cookies
	});

	console.log('[API] Response status:', response.status, 'for', fullUrl);

	// Handle errors
	if (!response.ok) {
		const errorBody = await response.json().catch(() => ({}));
		throw new Error(errorBody.message || `HTTP ${response.status}: ${response.statusText}`);
	}

	// Handle empty responses (204 No Content)
	let data: any;
	if (response.status === 204) {
		data = {};
	} else {
		data = await response.json();
	}

	// Return Orval response wrapper format
	// Orval expects: { data: ResponseBody, status: StatusCode, headers: Headers }
	return {
		data,
		status: response.status,
		headers: response.headers,
	} as T;
};

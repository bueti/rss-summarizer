import { defineConfig } from 'orval';

const OPENAPI_URL = process.env.VITE_API_URL || 'http://localhost:8080';

export default defineConfig({
	api: {
		input: {
			target: `${OPENAPI_URL}/openapi.json`,
		},
		output: {
			mode: 'single',
			target: './src/lib/api/generated.ts',
			client: 'fetch',
			override: {
				mutator: {
					path: './src/lib/api/client.ts',
					name: 'customFetch',
				},
			},
		},
	},
});

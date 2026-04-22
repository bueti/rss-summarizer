import { execSync } from 'child_process';
import fs from 'fs';

const OUTPUT_PATH = './src/lib/api/generated.ts';
const MAX_RETRIES = 5;
const RETRY_DELAY = 2000; // 2 seconds

async function sleep(ms: number) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

async function generate(attempt = 1): Promise<void> {
	try {
		console.log(`🔄 Generating API client with Orval... (attempt ${attempt}/${MAX_RETRIES})`);
		execSync('npx orval', { stdio: 'inherit' });
		console.log(`✅ API client generated successfully`);
	} catch (error: any) {
		if (attempt < MAX_RETRIES) {
			console.log(`⏳ Backend not ready, retrying in ${RETRY_DELAY / 1000}s...`);
			await sleep(RETRY_DELAY);
			return generate(attempt + 1);
		}

		// On final failure, check if we're in dev mode
		const isDev = process.env.NODE_ENV !== 'production';
		if (isDev && fs.existsSync(OUTPUT_PATH)) {
			console.warn(
				'⚠️  Could not fetch latest schema, using existing types. Backend may not be running.'
			);
			process.exit(0);
		}

		console.error('❌ Failed to generate API client after', MAX_RETRIES, 'attempts');
		console.error(error?.message || error);
		process.exit(isDev ? 0 : 1); // Don't fail dev builds if backend is down
	}
}

generate();

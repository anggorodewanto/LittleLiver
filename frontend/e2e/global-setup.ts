import { execSync, spawn, type ChildProcess } from 'child_process';
import { mkdtempSync, existsSync, rmSync } from 'fs';
import { join, dirname } from 'path';
import { tmpdir } from 'os';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const E2E_PORT = '3847';
const MOCK_CLAUDE_PORT = '3848';
let backendProcess: ChildProcess | null = null;
let mockClaudeProcess: ChildProcess | null = null;
let tmpDir: string | null = null;

const BACKEND_DIR = join(__dirname, '..', '..', 'backend');
const FRONTEND_DIR = join(__dirname, '..');

async function waitForServer(url: string, timeoutMs = 15000): Promise<void> {
	const start = Date.now();
	while (Date.now() - start < timeoutMs) {
		try {
			const res = await fetch(url);
			if (res.ok) {
				return;
			}
		} catch {
			// server not ready yet
		}
		await new Promise((r) => setTimeout(r, 200));
	}
	throw new Error(`Server at ${url} did not start within ${timeoutMs}ms`);
}

async function globalSetup(): Promise<() => Promise<void>> {
	// Kill any existing process on the E2E port and mock Claude port
	try {
		execSync(`lsof -ti :${E2E_PORT} | xargs kill -9 2>/dev/null || true`, { stdio: 'pipe' });
		execSync(`lsof -ti :${MOCK_CLAUDE_PORT} | xargs kill -9 2>/dev/null || true`, { stdio: 'pipe' });
		// Brief wait for port to be released
		await new Promise((r) => setTimeout(r, 500));
	} catch {
		// ignore
	}

	// Build the frontend
	console.log('Building frontend...');
	execSync('npm run build', { cwd: FRONTEND_DIR, stdio: 'pipe' });

	// Build the backend
	console.log('Building backend...');
	execSync('go build -o /tmp/littleliver-e2e-server ./cmd/server', {
		cwd: BACKEND_DIR,
		stdio: 'pipe'
	});

	// Create temp directory for DB
	tmpDir = mkdtempSync(join(tmpdir(), 'littleliver-e2e-'));
	const dbPath = join(tmpDir, 'test.db');
	const staticDir = join(FRONTEND_DIR, 'build');

	// Start mock Claude API server
	console.log('Starting mock Claude API server...');
	mockClaudeProcess = spawn('npx', ['tsx', join(FRONTEND_DIR, 'e2e', 'mock-claude-server.ts')], {
		stdio: ['pipe', 'pipe', 'pipe']
	});

	mockClaudeProcess.stderr?.on('data', (data: Buffer) => {
		console.error('[mock-claude]', data.toString().trim());
	});

	// Wait for mock Claude server
	await waitForServer(`http://localhost:${MOCK_CLAUDE_PORT}/mock/call-count`);
	console.log('Mock Claude API server is ready.');

	console.log(`Starting backend server (DB: ${dbPath})...`);

	backendProcess = spawn('/tmp/littleliver-e2e-server', [], {
		env: {
			...process.env,
			PORT: E2E_PORT,
			DATABASE_PATH: dbPath,
			MIGRATIONS_DIR: join(BACKEND_DIR, 'migrations'),
			STATIC_DIR: staticDir,
			TEST_MODE: '1',
			GOOGLE_CLIENT_ID: 'test-client-id',
			GOOGLE_CLIENT_SECRET: 'test-client-secret',
			SESSION_SECRET: 'test-session-secret-for-e2e',
			BASE_URL: `http://localhost:${E2E_PORT}`,
			ANTHROPIC_API_KEY: 'test-e2e-key',
			CLAUDE_API_BASE_URL: `http://localhost:${MOCK_CLAUDE_PORT}`
		},
		stdio: ['pipe', 'pipe', 'pipe']
	});

	backendProcess.stderr?.on('data', (data: Buffer) => {
		const msg = data.toString();
		if (!msg.includes('listening on')) {
			console.error('[backend]', msg.trim());
		}
	});

	// Wait for server to be ready
	await waitForServer(`http://localhost:${E2E_PORT}/health`);
	console.log('Backend server is ready.');

	// Return teardown function
	return async () => {
		if (backendProcess) {
			backendProcess.kill('SIGTERM');
			backendProcess = null;
		}
		if (mockClaudeProcess) {
			mockClaudeProcess.kill('SIGTERM');
			mockClaudeProcess = null;
		}
		if (tmpDir && existsSync(tmpDir)) {
			rmSync(tmpDir, { recursive: true, force: true });
		}
	};
}

export default globalSetup;

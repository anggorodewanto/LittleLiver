/**
 * Mock Claude API server for E2E tests.
 * Responds to POST /v1/messages with pre-configured lab extraction results.
 * Supports simulating errors via POST /mock/configure.
 */
import { createServer, type IncomingMessage, type ServerResponse } from 'http';

const PORT = 3848;

let nextError: { status: number; message: string } | null = null;
let callCount = 0;

const defaultResponse = {
	content: [
		{
			type: 'text',
			text: JSON.stringify({
				extracted: [
					{
						test_name: 'total_bilirubin',
						value: '1.8',
						unit: 'mg/dL',
						normal_range: '0.1-1.2',
						confidence: 'high'
					},
					{
						test_name: 'ALT',
						value: '52',
						unit: 'U/L',
						normal_range: '7-56',
						confidence: 'high'
					},
					{
						test_name: 'AST',
						value: '38',
						unit: 'U/L',
						normal_range: '10-40',
						confidence: 'medium'
					}
				],
				notes: 'Sample collected at Regional Hospital'
			})
		}
	]
};

function readBody(req: IncomingMessage): Promise<string> {
	return new Promise((resolve) => {
		let body = '';
		req.on('data', (chunk: Buffer) => {
			body += chunk.toString();
		});
		req.on('end', () => resolve(body));
	});
}

const server = createServer(async (req: IncomingMessage, res: ServerResponse) => {
	// CORS headers for cross-origin requests
	res.setHeader('Access-Control-Allow-Origin', '*');
	res.setHeader('Access-Control-Allow-Methods', 'POST, OPTIONS');
	res.setHeader('Access-Control-Allow-Headers', '*');

	if (req.method === 'OPTIONS') {
		res.writeHead(200);
		res.end();
		return;
	}

	// Configure mock behavior
	if (req.url === '/mock/configure' && req.method === 'POST') {
		const body = await readBody(req);
		const config = JSON.parse(body);
		if (config.error) {
			nextError = config.error;
		} else {
			nextError = null;
		}
		callCount = 0;
		res.writeHead(200, { 'Content-Type': 'application/json' });
		res.end(JSON.stringify({ status: 'configured' }));
		return;
	}

	// Reset mock state
	if (req.url === '/mock/reset' && req.method === 'POST') {
		nextError = null;
		callCount = 0;
		res.writeHead(200, { 'Content-Type': 'application/json' });
		res.end(JSON.stringify({ status: 'reset' }));
		return;
	}

	// Get call count
	if (req.url === '/mock/call-count' && req.method === 'GET') {
		res.writeHead(200, { 'Content-Type': 'application/json' });
		res.end(JSON.stringify({ count: callCount }));
		return;
	}

	// Mock Claude API endpoint
	if (req.url === '/v1/messages' && req.method === 'POST') {
		callCount++;
		await readBody(req); // consume request body

		if (nextError) {
			const err = nextError;
			nextError = null; // clear after one use (allows retry to succeed)
			res.writeHead(err.status, { 'Content-Type': 'application/json' });
			res.end(JSON.stringify({ error: { type: 'api_error', message: err.message } }));
			return;
		}

		res.writeHead(200, { 'Content-Type': 'application/json' });
		res.end(JSON.stringify(defaultResponse));
		return;
	}

	res.writeHead(404);
	res.end('Not found');
});

server.listen(PORT, () => {
	console.log(`Mock Claude API server listening on port ${PORT}`);
});

export default server;

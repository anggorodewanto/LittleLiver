import { describe, it, expect } from 'vitest';
import { readFileSync } from 'fs';
import { resolve } from 'path';

describe('PWA manifest.json', () => {
	const manifestPath = resolve(__dirname, '../../static/manifest.json');
	let manifest: Record<string, unknown>;

	it('is valid JSON', () => {
		const raw = readFileSync(manifestPath, 'utf-8');
		manifest = JSON.parse(raw);
		expect(manifest).toBeDefined();
	});

	it('has required fields', () => {
		const raw = readFileSync(manifestPath, 'utf-8');
		manifest = JSON.parse(raw);
		expect(manifest.name).toBe('LittleLiver');
		expect(manifest.short_name).toBe('LittleLiver');
		expect(manifest.start_url).toBe('/');
		expect(manifest.display).toBe('standalone');
		expect(manifest.theme_color).toBeDefined();
		expect(manifest.background_color).toBeDefined();
	});

	it('has at least one icon with required properties', () => {
		const raw = readFileSync(manifestPath, 'utf-8');
		manifest = JSON.parse(raw);
		const icons = manifest.icons as Array<{ src: string; sizes: string; type: string }>;
		expect(Array.isArray(icons)).toBe(true);
		expect(icons.length).toBeGreaterThanOrEqual(1);
		for (const icon of icons) {
			expect(icon.src).toBeDefined();
			expect(icon.sizes).toBeDefined();
			expect(icon.type).toBeDefined();
		}
	});
});

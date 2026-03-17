import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Page from '../routes/+page.svelte';

describe('+page.svelte', () => {
	it('renders a placeholder heading', () => {
		render(Page);
		expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('LittleLiver');
	});
});

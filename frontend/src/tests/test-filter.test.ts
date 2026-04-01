import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import TestFilter from '$lib/components/TestFilter.svelte';

describe('TestFilter', () => {
	it('renders an "All" button', () => {
		render(TestFilter, {
			props: { tests: ['ALT', 'AST'], selected: new Set<string>(), onchange: vi.fn() }
		});

		expect(screen.getByRole('button', { name: /all/i })).toBeInTheDocument();
	});

	it('renders a button for each test name', () => {
		render(TestFilter, {
			props: { tests: ['ALT', 'total_bilirubin'], selected: new Set<string>(), onchange: vi.fn() }
		});

		expect(screen.getByRole('button', { name: 'ALT' })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: 'Total Bilirubin' })).toBeInTheDocument();
	});

	it('"All" button is active when selected is empty', () => {
		render(TestFilter, {
			props: { tests: ['ALT'], selected: new Set<string>(), onchange: vi.fn() }
		});

		const allBtn = screen.getByRole('button', { name: /all/i });
		expect(allBtn.classList.contains('active')).toBe(true);
	});

	it('clicking a test button calls onchange with that test in the set', async () => {
		const onchange = vi.fn();
		render(TestFilter, {
			props: { tests: ['ALT', 'AST'], selected: new Set<string>(), onchange }
		});

		await fireEvent.click(screen.getByRole('button', { name: 'ALT' }));

		expect(onchange).toHaveBeenCalledWith(new Set(['ALT']));
	});

	it('clicking an active test button removes it from the set', async () => {
		const onchange = vi.fn();
		render(TestFilter, {
			props: { tests: ['ALT', 'AST'], selected: new Set(['ALT', 'AST']), onchange }
		});

		await fireEvent.click(screen.getByRole('button', { name: 'ALT' }));

		expect(onchange).toHaveBeenCalledWith(new Set(['AST']));
	});

	it('clicking "All" calls onchange with empty set', async () => {
		const onchange = vi.fn();
		render(TestFilter, {
			props: { tests: ['ALT'], selected: new Set(['ALT']), onchange }
		});

		await fireEvent.click(screen.getByRole('button', { name: /all/i }));

		expect(onchange).toHaveBeenCalledWith(new Set());
	});

	it('test buttons show active class when selected', () => {
		render(TestFilter, {
			props: { tests: ['ALT', 'AST'], selected: new Set(['ALT']), onchange: vi.fn() }
		});

		const altBtn = screen.getByRole('button', { name: 'ALT' });
		const astBtn = screen.getByRole('button', { name: 'AST' });
		expect(altBtn.classList.contains('active')).toBe(true);
		expect(astBtn.classList.contains('active')).toBe(false);
	});

	it('renders color dots when colors prop is provided', () => {
		const colors = new Map([['ALT', '#ef4444'], ['AST', '#3b82f6']]);
		const { container } = render(TestFilter, {
			props: { tests: ['ALT', 'AST'], selected: new Set<string>(), onchange: vi.fn(), colors }
		});

		const dots = container.querySelectorAll('.color-dot');
		expect(dots).toHaveLength(2);
		expect((dots[0] as HTMLElement).style.backgroundColor).toBe('rgb(239, 68, 68)');
		expect((dots[1] as HTMLElement).style.backgroundColor).toBe('rgb(59, 130, 246)');
	});

	it('does not render color dots when colors prop is absent', () => {
		const { container } = render(TestFilter, {
			props: { tests: ['ALT', 'AST'], selected: new Set<string>(), onchange: vi.fn() }
		});

		const dots = container.querySelectorAll('.color-dot');
		expect(dots).toHaveLength(0);
	});

	it('"All" button never has a color dot', () => {
		const colors = new Map([['ALT', '#ef4444']]);
		render(TestFilter, {
			props: { tests: ['ALT'], selected: new Set<string>(), onchange: vi.fn(), colors }
		});

		const allBtn = screen.getByRole('button', { name: /all/i });
		expect(allBtn.querySelector('.color-dot')).toBeNull();
	});

	it('displays friendly labels for known test names', () => {
		render(TestFilter, {
			props: { tests: ['direct_bilirubin', 'GGT'], selected: new Set<string>(), onchange: vi.fn() }
		});

		expect(screen.getByRole('button', { name: 'Direct Bilirubin' })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: 'GGT' })).toBeInTheDocument();
	});
});

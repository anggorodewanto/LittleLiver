import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import LabDateGroup from '$lib/components/LabDateGroup.svelte';
import type { LabResult } from '$lib/types/lab';

function makeResult(overrides: Partial<LabResult> = {}): LabResult {
	return {
		id: 'lr1',
		baby_id: 'b1',
		logged_by: 'u1',
		timestamp: '2026-03-15T10:00:00Z',
		test_name: 'ALT',
		value: '120',
		unit: 'U/L',
		created_at: '2026-03-15T10:00:00Z',
		updated_at: '2026-03-15T10:00:00Z',
		...overrides
	};
}

describe('LabDateGroup', () => {
	it('renders the formatted date as heading', () => {
		render(LabDateGroup, {
			props: { date: '2026-03-15', results: [makeResult()] }
		});

		expect(screen.getByRole('heading')).toHaveTextContent('Mar 15, 2026');
	});

	it('renders a row for each lab result', () => {
		const results = [
			makeResult({ id: 'lr1', test_name: 'ALT', value: '120' }),
			makeResult({ id: 'lr2', test_name: 'AST', value: '85' })
		];

		render(LabDateGroup, { props: { date: '2026-03-15', results } });

		expect(screen.getByText('ALT')).toBeInTheDocument();
		expect(screen.getByText('AST')).toBeInTheDocument();
		expect(screen.getByText('120')).toBeInTheDocument();
		expect(screen.getByText('85')).toBeInTheDocument();
	});

	it('shows unit next to value', () => {
		render(LabDateGroup, {
			props: { date: '2026-03-15', results: [makeResult({ value: '120', unit: 'U/L' })] }
		});

		expect(screen.getByText('U/L')).toBeInTheDocument();
	});

	it('shows normal range when present', () => {
		render(LabDateGroup, {
			props: {
				date: '2026-03-15',
				results: [makeResult({ normal_range: '0-40' })]
			}
		});

		expect(screen.getByText('0-40')).toBeInTheDocument();
	});

	it('does not show normal range column when absent', () => {
		const { container } = render(LabDateGroup, {
			props: {
				date: '2026-03-15',
				results: [makeResult({ normal_range: undefined })]
			}
		});

		expect(container.textContent).not.toContain('0-40');
	});

	it('shows notes when present', () => {
		render(LabDateGroup, {
			props: {
				date: '2026-03-15',
				results: [makeResult({ notes: 'Fasting sample' })]
			}
		});

		expect(screen.getByText('Fasting sample')).toBeInTheDocument();
	});

	it('displays friendly labels for known test names', () => {
		render(LabDateGroup, {
			props: {
				date: '2026-03-15',
				results: [makeResult({ test_name: 'total_bilirubin' })]
			}
		});

		expect(screen.getByText('Total Bilirubin')).toBeInTheDocument();
	});
});

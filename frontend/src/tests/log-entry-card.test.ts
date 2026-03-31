import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

import { goto } from '$app/navigation';
import LogEntryCard from '$lib/components/LogEntryCard.svelte';
import type { LogTypeConfig } from '$lib/types/logs';

const feedingType: LogTypeConfig = {
	key: 'feeding',
	label: 'Feedings',
	endpoint: 'feedings',
	metricParam: 'feeding'
};

const temperatureType: LogTypeConfig = {
	key: 'temperature',
	label: 'Temperatures',
	endpoint: 'temperatures',
	metricParam: 'temperature'
};

const medLogType: LogTypeConfig = {
	key: 'med-log',
	label: 'Med Logs',
	endpoint: 'med-logs',
	metricParam: 'med'
};

describe('LogEntryCard', () => {
	let ondelete: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		ondelete = vi.fn();
		vi.mocked(goto).mockReset();
	});

	it('renders timestamp for a feeding entry', () => {
		const entry = {
			id: 'f1',
			timestamp: '2026-03-20T14:30:00Z',
			feed_type: 'breast_milk',
			volume_ml: 60
		};

		render(LogEntryCard, { props: { entry, logType: feedingType, ondelete } });

		expect(screen.getByText(new Date('2026-03-20T14:30:00Z').toLocaleString())).toBeInTheDocument();
	});

	it('renders feeding-specific fields', () => {
		const entry = {
			id: 'f2',
			timestamp: '2026-03-20T14:30:00Z',
			feed_type: 'formula',
			volume_ml: 90,
			calories: 65,
			duration_min: 20
		};

		render(LogEntryCard, { props: { entry, logType: feedingType, ondelete } });

		expect(screen.getByText(/formula/i)).toBeInTheDocument();
		expect(screen.getByText(/90\s*mL/)).toBeInTheDocument();
		expect(screen.getByText(/65\s*kcal/)).toBeInTheDocument();
		expect(screen.getByText(/20\s*min/)).toBeInTheDocument();
	});

	it('renders temperature-specific fields', () => {
		const entry = {
			id: 't1',
			timestamp: '2026-03-20T08:00:00Z',
			value: 37.2,
			method: 'axillary'
		};

		render(LogEntryCard, { props: { entry, logType: temperatureType, ondelete } });

		expect(screen.getByText(/37\.2\s*°C/)).toBeInTheDocument();
		expect(screen.getByText(/axillary/i)).toBeInTheDocument();
	});

	it('renders Edit button that navigates to edit page', async () => {
		const entry = {
			id: 'f3',
			timestamp: '2026-03-20T14:30:00Z',
			feed_type: 'breast_milk'
		};

		render(LogEntryCard, { props: { entry, logType: feedingType, ondelete } });

		const editBtn = screen.getByRole('button', { name: /edit/i });
		await fireEvent.click(editBtn);

		expect(goto).toHaveBeenCalledWith('/log/feeding?edit=f3');
	});

	it('renders Delete button with confirmation flow', async () => {
		const entry = {
			id: 'f4',
			timestamp: '2026-03-20T14:30:00Z',
			feed_type: 'breast_milk'
		};

		render(LogEntryCard, { props: { entry, logType: feedingType, ondelete } });

		const deleteBtn = screen.getByRole('button', { name: /delete/i });
		await fireEvent.click(deleteBtn);

		// Should show confirmation
		expect(screen.getByText(/are you sure/i)).toBeInTheDocument();

		// Confirm deletion
		const confirmBtn = screen.getByRole('button', { name: /confirm/i });
		await fireEvent.click(confirmBtn);

		expect(ondelete).toHaveBeenCalledWith('f4');
	});

	it('shows notes when present', () => {
		const entry = {
			id: 'f5',
			timestamp: '2026-03-20T14:30:00Z',
			feed_type: 'breast_milk',
			notes: 'Baby was fussy during feed'
		};

		render(LogEntryCard, { props: { entry, logType: feedingType, ondelete } });

		expect(screen.getByText(/baby was fussy during feed/i)).toBeInTheDocument();
	});

	it('renders med-log with medication name and skipped status', () => {
		const entry = {
			id: 'm1',
			medication_id: 'med-1',
			given_at: null,
			skipped: true,
			skip_reason: 'Vomiting',
			created_at: '2026-03-20T14:30:00Z'
		};
		const medNames = { 'med-1': 'Ursodiol 50mg' };

		render(LogEntryCard, { props: { entry, logType: medLogType, ondelete, medNames } });

		expect(screen.getByText('Ursodiol 50mg')).toBeInTheDocument();
		expect(screen.getByText(/skipped/i)).toBeInTheDocument();
		expect(screen.getByText(/vomiting/i)).toBeInTheDocument();
	});

	it('renders med-log given_at as header timestamp and shows medication name', () => {
		const entry = {
			id: 'm2',
			medication_id: 'med-2',
			given_at: '2026-03-20T14:30:00Z',
			skipped: false,
			created_at: '2026-03-20T14:00:00Z'
		};
		const medNames = { 'med-2': 'Vitamin D 400IU' };

		render(LogEntryCard, { props: { entry, logType: medLogType, ondelete, medNames } });

		// Should show given_at time in the header, not "Invalid Date"
		const header = document.querySelector('.card-header .timestamp');
		expect(header?.textContent).toBe(new Date('2026-03-20T14:30:00Z').toLocaleString());
		// Should show medication name, not duplicate date
		expect(screen.getByText('Vitamin D 400IU')).toBeInTheDocument();
	});

	it('renders med-log created_at as header when skipped (no given_at)', () => {
		const entry = {
			id: 'm3',
			given_at: null,
			skipped: true,
			created_at: '2026-03-20T14:00:00Z'
		};

		render(LogEntryCard, { props: { entry, logType: medLogType, ondelete } });

		expect(screen.getByText(new Date('2026-03-20T14:00:00Z').toLocaleString())).toBeInTheDocument();
	});
});

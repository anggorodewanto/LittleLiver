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

	it('renders med-log skipped status', () => {
		const entry = {
			id: 'm1',
			timestamp: '2026-03-20T14:30:00Z',
			given_at: null,
			skipped: true,
			skip_reason: 'Vomiting'
		};

		render(LogEntryCard, { props: { entry, logType: medLogType, ondelete } });

		expect(screen.getByText(/skipped/i)).toBeInTheDocument();
		expect(screen.getByText(/vomiting/i)).toBeInTheDocument();
	});
});

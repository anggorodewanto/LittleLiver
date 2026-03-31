import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

import { goto } from '$app/navigation';
import LogEntryRow from '$lib/components/LogEntryRow.svelte';
import type { LogTypeConfig } from '$lib/types/logs';

const feedingType: LogTypeConfig = {
	key: 'feeding',
	label: 'Feedings',
	endpoint: 'feedings',
	metricParam: 'feeding'
};

const stoolType: LogTypeConfig = {
	key: 'stool',
	label: 'Stools',
	endpoint: 'stools',
	metricParam: 'stool'
};

const medLogType: LogTypeConfig = {
	key: 'med-log',
	label: 'Med Logs',
	endpoint: 'med-logs',
	metricParam: 'med'
};

const noteType: LogTypeConfig = {
	key: 'note',
	label: 'Notes',
	endpoint: 'notes',
	metricParam: 'notes'
};

const fluidType: LogTypeConfig = {
	key: 'fluid',
	label: 'Fluid I/O',
	endpoint: 'fluid-log',
	metricParam: 'other_intake'
};

describe('LogEntryRow', () => {
	let ondelete: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		ondelete = vi.fn();
		vi.mocked(goto).mockReset();
	});

	it('renders time-only (not full date) for an entry', () => {
		const entry = {
			id: 'f1',
			timestamp: '2026-03-20T14:30:00Z',
			feed_type: 'formula',
			volume_ml: 90
		};

		render(LogEntryRow, { props: { entry, logType: feedingType, ondelete } });

		// Should not show the full date string
		expect(screen.queryByText(/2026/)).not.toBeInTheDocument();
		// Should show time
		expect(screen.getByText(/\d{1,2}:\d{2}/)).toBeInTheDocument();
	});

	it('renders inline feeding summary with separator', () => {
		const entry = {
			id: 'f2',
			timestamp: '2026-03-20T14:30:00Z',
			feed_type: 'formula',
			volume_ml: 90,
			calories: 65
		};

		render(LogEntryRow, { props: { entry, logType: feedingType, ondelete } });

		expect(screen.getByText(/formula/i)).toBeInTheDocument();
		expect(screen.getByText(/90\s*mL/)).toBeInTheDocument();
		expect(screen.getByText(/65\s*kcal/)).toBeInTheDocument();
	});

	it('renders inline stool summary', () => {
		const entry = {
			id: 's1',
			timestamp: '2026-03-20T10:00:00Z',
			color_rating: 3,
			consistency: 'soft',
			volume_estimate: 'medium'
		};

		render(LogEntryRow, { props: { entry, logType: stoolType, ondelete } });

		expect(screen.getByText(/3\/7/)).toBeInTheDocument();
		expect(screen.getByText(/soft/i)).toBeInTheDocument();
		expect(screen.getByText(/medium/i)).toBeInTheDocument();
	});

	it('renders med-log with medication name and status', () => {
		const entry = {
			id: 'm1',
			medication_id: 'med-1',
			given_at: '2026-03-20T14:30:00Z',
			skipped: false,
			created_at: '2026-03-20T14:00:00Z'
		};
		const medNames = { 'med-1': 'Ursodiol 50mg' };

		render(LogEntryRow, { props: { entry, logType: medLogType, ondelete, medNames } });

		expect(screen.getByText(/Ursodiol 50mg/)).toBeInTheDocument();
		expect(screen.getByText(/given/i)).toBeInTheDocument();
	});

	it('renders med-log skipped status with reason', () => {
		const entry = {
			id: 'm2',
			medication_id: 'med-1',
			given_at: null,
			skipped: true,
			skip_reason: 'Vomiting',
			created_at: '2026-03-20T14:00:00Z'
		};
		const medNames = { 'med-1': 'Ursodiol 50mg' };

		render(LogEntryRow, { props: { entry, logType: medLogType, ondelete, medNames } });

		expect(screen.getByText(/skipped/i)).toBeInTheDocument();
		expect(screen.getByText(/vomiting/i)).toBeInTheDocument();
	});

	it('renders note with truncated content', () => {
		const entry = {
			id: 'n1',
			timestamp: '2026-03-20T14:30:00Z',
			category: 'concern',
			content: 'Baby was very fussy today and did not want to eat anything at all during the morning hours which was unusual'
		};

		render(LogEntryRow, { props: { entry, logType: noteType, ondelete } });

		expect(screen.getByText(/concern/i)).toBeInTheDocument();
		// Should be truncated
		expect(screen.getByText(/\.\.\./)).toBeInTheDocument();
	});

	it('renders fluid I/O summary', () => {
		const entry = {
			id: 'fl1',
			timestamp: '2026-03-20T14:30:00Z',
			direction: 'intake',
			method: 'oral',
			volume_ml: 30
		};

		render(LogEntryRow, { props: { entry, logType: fluidType, ondelete } });

		expect(screen.getByText(/intake/i)).toBeInTheDocument();
		expect(screen.getByText(/oral/i)).toBeInTheDocument();
		expect(screen.getByText(/30\s*mL/)).toBeInTheDocument();
	});

	it('Edit button navigates to edit page', async () => {
		const entry = {
			id: 'f3',
			timestamp: '2026-03-20T14:30:00Z',
			feed_type: 'breast_milk'
		};

		render(LogEntryRow, { props: { entry, logType: feedingType, ondelete } });

		const editBtn = screen.getByRole('button', { name: /edit/i });
		await fireEvent.click(editBtn);

		expect(goto).toHaveBeenCalledWith('/log/feeding?edit=f3');
	});

	it('Delete button shows confirmation then calls ondelete', async () => {
		const entry = {
			id: 'f4',
			timestamp: '2026-03-20T14:30:00Z',
			feed_type: 'formula'
		};

		render(LogEntryRow, { props: { entry, logType: feedingType, ondelete } });

		const deleteBtn = screen.getByRole('button', { name: /delete/i });
		await fireEvent.click(deleteBtn);

		// Should show confirmation
		expect(screen.getByText(/delete\?/i)).toBeInTheDocument();

		const confirmBtn = screen.getByRole('button', { name: /confirm/i });
		await fireEvent.click(confirmBtn);

		expect(ondelete).toHaveBeenCalledWith('f4');
	});

	it('Cancel delete restores row', async () => {
		const entry = {
			id: 'f5',
			timestamp: '2026-03-20T14:30:00Z',
			feed_type: 'formula',
			volume_ml: 120
		};

		render(LogEntryRow, { props: { entry, logType: feedingType, ondelete } });

		const deleteBtn = screen.getByRole('button', { name: /delete/i });
		await fireEvent.click(deleteBtn);

		const cancelBtn = screen.getByRole('button', { name: /cancel/i });
		await fireEvent.click(cancelBtn);

		// Summary should be visible again
		expect(screen.getByText(/120\s*mL/)).toBeInTheDocument();
		expect(ondelete).not.toHaveBeenCalled();
	});
});

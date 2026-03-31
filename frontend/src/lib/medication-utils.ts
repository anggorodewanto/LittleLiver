export const FREQUENCY_OPTIONS = [
	{ value: 'once_daily', label: 'Once daily', timeSlots: 1 },
	{ value: 'twice_daily', label: 'Twice daily', timeSlots: 2 },
	{ value: 'three_times_daily', label: 'Three times daily', timeSlots: 3 },
	{ value: 'every_x_days', label: 'Every X days', timeSlots: 0 },
	{ value: 'as_needed', label: 'As needed', timeSlots: 0 },
	{ value: 'custom', label: 'Custom', timeSlots: 0 }
] as const;

export function formatFrequency(freq: string, intervalDays?: number | null): string {
	if (freq === 'every_x_days' && intervalDays) {
		return `Every ${intervalDays} days`;
	}
	return FREQUENCY_OPTIONS.find((o) => o.value === freq)?.label ?? freq;
}

export function getTimeSlotCount(freq: string): number {
	return FREQUENCY_OPTIONS.find((o) => o.value === freq)?.timeSlots ?? 0;
}

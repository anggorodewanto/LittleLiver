export function defaultTimestamp(): string {
	const now = new Date();
	const offset = now.getTimezoneOffset();
	const local = new Date(now.getTime() - offset * 60000);
	return local.toISOString().slice(0, 16);
}

/** Convert a datetime-local value (YYYY-MM-DDTHH:MM) to ISO 8601 UTC (YYYY-MM-DDTHH:MM:SSZ). */
export function toISO8601(datetimeLocal: string): string {
	if (datetimeLocal.endsWith('Z')) {
		return datetimeLocal;
	}
	// datetime-local values are in the user's local timezone.
	// Parse via Date (which interprets without Z as local time) then convert to UTC.
	const d = new Date(datetimeLocal);
	return d.toISOString().replace(/\.\d{3}Z$/, 'Z');
}

export function formatDateISO(date: Date): string {
	return date.toISOString().split('T')[0];
}

export function formatDateTime(dt: string): string {
	return new Date(dt).toLocaleString();
}

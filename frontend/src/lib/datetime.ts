export function defaultTimestamp(): string {
	const now = new Date();
	const offset = now.getTimezoneOffset();
	const local = new Date(now.getTime() - offset * 60000);
	return local.toISOString().slice(0, 16);
}

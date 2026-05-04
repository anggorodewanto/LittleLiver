// Age helpers for chronological and corrected (gestation-adjusted) ages.
// Corrected age subtracts prematurity from chronological age and is meaningful
// only for preterm babies (born < 37 weeks gestation).

const FULL_TERM_DAYS = 40 * 7;
const PRETERM_THRESHOLD_DAYS = 37 * 7;

export interface AgeParts {
	months: number;
	days: number;
	beforeTerm: boolean;
}

/** Days the baby was born before 40w 0d. Returns 0 for full-term/post-term/unknown. */
export function prematurityDays(weeks: number | null, days: number | null): number {
	if (weeks == null) {
		return 0;
	}
	const total = weeks * 7 + (days ?? 0);
	const diff = FULL_TERM_DAYS - total;
	return diff > 0 ? diff : 0;
}

/** True when gestational age is < 37 weeks. */
export function isPreterm(weeks: number | null, days: number | null): boolean {
	if (weeks == null) {
		return false;
	}
	const total = weeks * 7 + (days ?? 0);
	return total < PRETERM_THRESHOLD_DAYS;
}

function diffMonthsDays(fromMs: number, toMs: number): { months: number; days: number } {
	const from = new Date(fromMs);
	from.setHours(0, 0, 0, 0);
	const to = new Date(toMs);
	to.setHours(0, 0, 0, 0);

	let months = (to.getFullYear() - from.getFullYear()) * 12 + (to.getMonth() - from.getMonth());
	let days = to.getDate() - from.getDate();
	if (days < 0) {
		months--;
		const prevMonth = new Date(to.getFullYear(), to.getMonth(), 0);
		days += prevMonth.getDate();
	}
	return { months, days };
}

/** Chronological age (DOB → today) in whole months and remaining days. */
export function chronologicalAge(dobStr: string, nowMs: number): { months: number; days: number } {
	const dob = new Date(dobStr + 'T00:00:00').getTime();
	return diffMonthsDays(dob, nowMs);
}

/**
 * Corrected age: chronological age minus weeks-of-prematurity.
 * Returns null when the baby is not preterm or gestational age is unknown.
 * `beforeTerm` is true when today is before the corrected DOB (the original
 * 40-week due date hasn't arrived yet).
 */
export function correctedAge(
	dobStr: string,
	gestWeeks: number | null,
	gestDays: number | null,
	nowMs: number
): AgeParts | null {
	if (!isPreterm(gestWeeks, gestDays)) {
		return null;
	}
	const offsetDays = prematurityDays(gestWeeks, gestDays);
	const correctedDob = new Date(dobStr + 'T00:00:00').getTime() + offsetDays * 24 * 60 * 60 * 1000;

	const today = new Date(nowMs);
	today.setHours(0, 0, 0, 0);
	if (today.getTime() < correctedDob) {
		return { months: 0, days: 0, beforeTerm: true };
	}
	const { months, days } = diffMonthsDays(correctedDob, nowMs);
	return { months, days, beforeTerm: false };
}

/** Render an AgeParts as "M mo D d", or "not yet at term" when before term. */
export function formatAge(age: AgeParts): string {
	if (age.beforeTerm) {
		return 'not yet at term';
	}
	return `${age.months} mo ${age.days} d`;
}

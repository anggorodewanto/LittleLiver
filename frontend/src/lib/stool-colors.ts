export const COLOR_SWATCHES = [
	{ rating: 1, ref: 'white', label: 'White', color: '#F5F5DC', meaning: 'Acholic — NO bile flow — ALERT' },
	{ rating: 2, ref: 'clay', label: 'Clay', color: '#D2B48C', meaning: 'Acholic — minimal bile — ALERT' },
	{ rating: 3, ref: 'pale_yellow', label: 'Pale Yellow', color: '#FFFACD', meaning: 'Questionable — ALERT' },
	{ rating: 4, ref: 'yellow', label: 'Yellow', color: '#FFD700', meaning: 'Some bile present — monitor closely' },
	{ rating: 5, ref: 'light_green', label: 'Light Green', color: '#90EE90', meaning: 'Good bile flow' },
	{ rating: 6, ref: 'green', label: 'Green', color: '#228B22', textColor: 'white', meaning: 'Good bile flow' },
	{ rating: 7, ref: 'brown', label: 'Brown', color: '#8B4513', textColor: 'white', meaning: 'Normal bile flow' }
] as const;

export function stoolStatusColor(rating: number): string {
	if (rating <= 3) return '#dc2626';
	if (rating <= 5) return '#f59e0b';
	return '#84cc16';
}

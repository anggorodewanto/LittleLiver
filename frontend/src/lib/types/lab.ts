export interface LabResult {
	id: string;
	baby_id: string;
	logged_by: string;
	updated_by?: string;
	timestamp: string;
	test_name: string;
	value: string;
	unit?: string;
	normal_range?: string;
	notes?: string;
	created_at: string;
	updated_at: string;
}

export interface LabResultsPage {
	data: LabResult[];
	next_cursor: string | null;
}

const TEST_LABELS: Record<string, string> = {
	total_bilirubin: 'Total Bilirubin',
	direct_bilirubin: 'Direct Bilirubin',
	ALT: 'ALT',
	AST: 'AST',
	GGT: 'GGT',
	albumin: 'Albumin',
	INR: 'INR',
	platelets: 'Platelets'
};

export function labTestLabel(testName: string): string {
	return TEST_LABELS[testName] ?? testName;
}

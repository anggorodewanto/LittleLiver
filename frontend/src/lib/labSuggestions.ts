import { apiClient } from '$lib/api';

export interface LabTestSuggestion {
	test_name: string;
	unit?: string;
	normal_range?: string;
}

export interface QuickPick {
	label: string;
	testName: string;
	unit: string;
}

export const QUICK_PICKS: readonly QuickPick[] = [
	{ label: 'Total Bilirubin', testName: 'total_bilirubin', unit: 'mg/dL' },
	{ label: 'Direct Bilirubin', testName: 'direct_bilirubin', unit: 'mg/dL' },
	{ label: 'ALT', testName: 'ALT', unit: 'U/L' },
	{ label: 'AST', testName: 'AST', unit: 'U/L' },
	{ label: 'GGT', testName: 'GGT', unit: 'U/L' },
	{ label: 'Albumin', testName: 'albumin', unit: 'g/dL' },
	{ label: 'INR', testName: 'INR', unit: '' },
	{ label: 'Platelets', testName: 'platelets', unit: '\u00d710\u00b3/\u00b5L' }
] as const;

export async function fetchLabSuggestions(babyId: string): Promise<LabTestSuggestion[]> {
	try {
		return await apiClient.get<LabTestSuggestion[]>(`/babies/${babyId}/labs/tests`);
	} catch {
		return [];
	}
}

export function mergeWithQuickPicks(db: LabTestSuggestion[]): LabTestSuggestion[] {
	const map = new Map<string, LabTestSuggestion>();
	for (const s of db) {
		map.set(s.test_name, s);
	}
	for (const pick of QUICK_PICKS) {
		if (!map.has(pick.testName)) {
			map.set(pick.testName, {
				test_name: pick.testName,
				unit: pick.unit || undefined
			});
		}
	}
	return Array.from(map.values());
}

export function getQuickPickLabel(testName: string): string {
	const pick = QUICK_PICKS.find((p) => p.testName === testName);
	return pick ? pick.label : testName;
}

export function normalizeTestName(name: string): string {
	return name.toLowerCase().replace(/[\s/\-_.]+/g, '');
}

export function findSuggestionMatch(
	name: string,
	suggestions: LabTestSuggestion[]
): LabTestSuggestion | undefined {
	if (!name) return undefined;
	const lower = name.toLowerCase();
	const exact = suggestions.find((s) => s.test_name.toLowerCase() === lower);
	if (exact) return exact;

	const norm = normalizeTestName(name);
	if (!norm) return undefined;
	return suggestions.find((s) => {
		const sNorm = normalizeTestName(s.test_name);
		if (!sNorm) return false;
		return sNorm.includes(norm) || norm.includes(sNorm);
	});
}

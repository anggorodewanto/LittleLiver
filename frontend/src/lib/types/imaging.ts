export interface PhotoRef {
	key: string;
	url: string;
	thumbnail_url: string;
}

export interface ImagingStudy {
	id: string;
	baby_id: string;
	logged_by: string;
	updated_by?: string;
	timestamp: string;
	study_date: string; // YYYY-MM-DD
	study_type: string;
	notes?: string;
	photos: PhotoRef[];
	created_at: string;
	updated_at: string;
}

export interface ImagingStudiesPage {
	data: ImagingStudy[];
	next_cursor: string | null;
}

export interface ImagingSuggestion {
	study_type: string;
	study_date: string;
	findings: string;
	notes: string;
}

export interface ImagingExtractResponse {
	suggested: ImagingSuggestion;
	notes?: string;
}

export const IMAGING_QUICK_PICKS = ['CT', 'Ultrasound', 'MRI'] as const;
export type ImagingQuickPick = (typeof IMAGING_QUICK_PICKS)[number];

export function isPDFKey(key: string): boolean {
	return key.toLowerCase().endsWith('.pdf');
}

export type ImmunizationStatus = 'done' | 'due' | 'upcoming';

export interface ImmunizationSlot {
	code: string;
	name: string;
	dose_number: number;
	dose_label: string;
	age_months: number;
	age_label: string;
	mandatory: boolean;
	status: ImmunizationStatus;
	due_date?: string;
	administered_date?: string;
	record_id?: string;
	off_schedule: boolean;
}

export interface ImmunizationScheduleResponse {
	slots: ImmunizationSlot[];
}

export interface ImmunizationRecord {
	id: string;
	baby_id: string;
	logged_by: string;
	updated_by?: string;
	vaccine_code: string;
	vaccine_name: string;
	dose_number?: number;
	administered_date: string;
	provider?: string;
	lot_number?: string;
	notes?: string;
	created_at: string;
	updated_at: string;
}

export interface ImmunizationRecordsPage {
	data: ImmunizationRecord[];
	next_cursor: string | null;
}

export interface ScheduleEntry {
	code: string;
	name: string;
	dose_number: number;
	dose_label: string;
	age_months: number;
	age_label: string;
	mandatory: boolean;
}

export interface ImmunizationReferenceResponse {
	schedule: ScheduleEntry[];
}

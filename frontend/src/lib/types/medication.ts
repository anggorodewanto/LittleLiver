export interface Medication {
	id: string;
	baby_id: string;
	name: string;
	dose: string;
	frequency: string;
	schedule: string | null;
	timezone: string | null;
	interval_days: number | null;
	starts_from: string | null;
	active: boolean;
	created_at: string;
	updated_at: string;
}

export interface MedicationsResponse {
	medications: Medication[];
}

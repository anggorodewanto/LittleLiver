export interface Medication {
	id: string;
	baby_id: string;
	name: string;
	dose: string;
	frequency: string;
	schedule: string | null;
	timezone: string | null;
	interval_days: number | null;
	active: boolean;
	created_at: string;
	updated_at: string;
}

export interface MedicationsResponse {
	medications: Medication[];
}

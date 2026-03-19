export interface Medication {
	id: string;
	baby_id: string;
	name: string;
	dose: string;
	frequency: string;
	schedule: string | null;
	timezone: string | null;
	active: boolean;
	created_at: string;
	updated_at: string;
}

export interface MedicationsResponse {
	medications: Medication[];
}

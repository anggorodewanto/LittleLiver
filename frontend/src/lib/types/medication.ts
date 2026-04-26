export type DoseUnit = 'mg' | 'ml' | 'tablet' | 'packet' | 'dose';
export type ContainerKind = 'bottle' | 'pill_pack' | 'packet' | 'vial' | 'other';

export interface Medication {
	id: string;
	baby_id: string;
	name: string;
	dose: string;
	frequency: string;
	schedule_times: string[];
	timezone: string | null;
	interval_days: number | null;
	starts_from: string | null;
	active: boolean;
	dose_amount?: number | null;
	dose_unit?: DoseUnit | null;
	low_stock_threshold?: number | null;
	expiry_warning_days?: number | null;
	created_at: string;
	updated_at: string;
}

export interface MedicationsResponse {
	medications: Medication[];
}

export interface MedicationContainer {
	id: string;
	medication_id: string;
	baby_id: string;
	kind: ContainerKind;
	unit: DoseUnit;
	quantity_initial: number;
	quantity_remaining: number;
	opened_at: string | null;
	max_days_after_opening: number | null;
	expiration_date: string | null;
	effective_expiry: string | null;
	depleted: boolean;
	notes: string | null;
	created_at: string;
	updated_at: string;
}

export interface StockAdjustment {
	id: string;
	container_id: string;
	delta: number;
	reason: string | null;
	adjusted_by: string;
	adjusted_at: string;
	created_at: string;
}

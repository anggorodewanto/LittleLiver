export interface CarePlanPhase {
	id?: string;
	care_plan_id?: string;
	seq: number;
	label: string;
	start_date: string;
	ends_on?: string | null;
	notes?: string | null;
	created_at?: string;
	updated_at?: string;
}

export interface CarePlan {
	id: string;
	baby_id: string;
	logged_by: string;
	updated_by?: string | null;
	name: string;
	notes?: string | null;
	timezone: string;
	active: boolean;
	created_at: string;
	updated_at: string;
	phases: CarePlanPhase[];
}

export interface CurrentCarePlanPhase {
	plan_id: string;
	plan_name: string;
	phase_id: string;
	label: string;
	ends_on?: string | null;
	days_remaining?: number | null;
}

export interface CarePlanRequest {
	name: string;
	notes?: string | null;
	timezone?: string;
	active?: boolean;
	phases: CarePlanPhase[];
}

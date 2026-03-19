export interface PercentilePoint {
	age_days: number;
	value: number;
}

export interface Percentiles {
	p3: PercentilePoint[];
	p15: PercentilePoint[];
	p50: PercentilePoint[];
	p85: PercentilePoint[];
	p97: PercentilePoint[];
}

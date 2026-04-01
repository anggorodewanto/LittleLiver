export interface LogTypeConfig {
	key: string;
	label: string;
	endpoint: string;
	metricParam: string;
}

export const LOG_TYPES: LogTypeConfig[] = [
	{ key: 'feeding', label: 'Feedings', endpoint: 'feedings', metricParam: 'feeding' },
	{ key: 'stool', label: 'Stools', endpoint: 'stools', metricParam: 'stool' },
	{ key: 'urine', label: 'Urine', endpoint: 'urine', metricParam: 'urine' },
	{ key: 'weight', label: 'Weights', endpoint: 'weights', metricParam: 'weight' },
	{ key: 'temperature', label: 'Temperatures', endpoint: 'temperatures', metricParam: 'temperature' },
	{ key: 'abdomen', label: 'Abdomen', endpoint: 'abdomen', metricParam: 'abdomen' },
	{ key: 'skin', label: 'Skin', endpoint: 'skin', metricParam: 'skin' },
	{ key: 'bruising', label: 'Bruising', endpoint: 'bruising', metricParam: 'bruising' },
	{ key: 'lab', label: 'Labs', endpoint: 'labs', metricParam: 'lab' },
	{ key: 'note', label: 'Notes', endpoint: 'notes', metricParam: 'notes' },
	{ key: 'med-log', label: 'Med Logs', endpoint: 'med-logs', metricParam: 'med' },
	{ key: 'fluid', label: 'Fluid I/O', endpoint: 'fluid-log', metricParam: 'other_intake' },
	{ key: 'head-circumference', label: 'Head Circ.', endpoint: 'head-circumferences', metricParam: 'head_circumference' },
	{ key: 'upper-arm-circumference', label: 'Arm Circ.', endpoint: 'upper-arm-circumferences', metricParam: 'upper_arm_circumference' }
];

import type { Baby } from '$lib/stores/baby';

export const mockBabies: Baby[] = [
	{
		id: 'baby-1',
		name: 'Alice',
		date_of_birth: '2025-06-01',
		sex: 'female',
		diagnosis_date: '2025-06-15',
		kasai_date: '2025-06-20'
	},
	{
		id: 'baby-2',
		name: 'Bob',
		date_of_birth: '2025-09-01',
		sex: 'male',
		diagnosis_date: null,
		kasai_date: null
	}
];

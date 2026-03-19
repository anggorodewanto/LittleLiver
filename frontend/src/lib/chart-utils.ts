export const dateTickCallback = (value: string | number) =>
	new Date(value as number).toLocaleDateString();

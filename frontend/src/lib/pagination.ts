export interface PaginatedResponse<T> {
	items: T[];
	next_cursor: string | null;
}

export interface PaginationState<T> {
	items: T[];
	nextCursor: string | null;
	loading: boolean;
}

export function createPaginationState<T>(): PaginationState<T> {
	return { items: [], nextCursor: null, loading: false };
}

export function appendPage<T>(
	state: PaginationState<T>,
	page: PaginatedResponse<T>
): PaginationState<T> {
	return {
		items: [...state.items, ...page.items],
		nextCursor: page.next_cursor,
		loading: false
	};
}

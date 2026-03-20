import { describe, it, expect } from 'vitest';
import { createPaginationState, appendPage } from '$lib/pagination';

describe('pagination', () => {
	it('createPaginationState returns empty initial state', () => {
		const state = createPaginationState<string>();

		expect(state.items).toEqual([]);
		expect(state.nextCursor).toBeNull();
		expect(state.loading).toBe(false);
	});

	it('appendPage appends items and updates cursor', () => {
		const state = createPaginationState<string>();

		const result = appendPage(state, {
			items: ['a', 'b'],
			next_cursor: 'cursor-1'
		});

		expect(result.items).toEqual(['a', 'b']);
		expect(result.nextCursor).toBe('cursor-1');
		expect(result.loading).toBe(false);

		const result2 = appendPage(result, {
			items: ['c'],
			next_cursor: null
		});

		expect(result2.items).toEqual(['a', 'b', 'c']);
		expect(result2.nextCursor).toBeNull();
	});
});

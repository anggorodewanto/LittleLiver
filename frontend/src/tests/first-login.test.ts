import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import FirstLogin from '$lib/components/FirstLogin.svelte';

describe('FirstLogin', () => {
	let oncreate: ReturnType<typeof vi.fn>;
	let onjoin: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		oncreate = vi.fn();
		onjoin = vi.fn();
	});

	it('renders a welcome heading', () => {
		render(FirstLogin, { props: { oncreate, onjoin } });

		expect(screen.getByRole('heading', { name: /welcome/i })).toBeInTheDocument();
	});

	it('shows create baby and join options', () => {
		render(FirstLogin, { props: { oncreate, onjoin } });

		expect(
			screen.getByRole('button', { name: /create.*baby/i }) ||
				screen.getByText(/create.*baby/i)
		).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: /join.*invite/i }) ||
				screen.getByText(/join.*invite|enter.*invite/i)
		).toBeInTheDocument();
	});

	it('shows create baby form when create option is selected', async () => {
		render(FirstLogin, { props: { oncreate, onjoin } });

		const createBtn = screen.getByRole('button', { name: /create.*baby/i });
		await fireEvent.click(createBtn);

		expect(screen.getByLabelText(/name/i)).toBeInTheDocument();
	});

	it('shows join form when join option is selected', async () => {
		render(FirstLogin, { props: { oncreate, onjoin } });

		const joinBtn = screen.getByRole('button', { name: /join.*invite|enter.*invite/i });
		await fireEvent.click(joinBtn);

		expect(screen.getByLabelText(/invite code/i)).toBeInTheDocument();
	});

	it('does not show forms initially — only the choice buttons', () => {
		render(FirstLogin, { props: { oncreate, onjoin } });

		expect(screen.queryByLabelText(/name/i)).toBeNull();
		expect(screen.queryByLabelText(/invite code/i)).toBeNull();
	});

	it('passes oncreate callback through to CreateBabyForm', async () => {
		render(FirstLogin, { props: { oncreate, onjoin } });

		await fireEvent.click(screen.getByRole('button', { name: /create.*baby/i }));

		await fireEvent.input(screen.getByLabelText(/name/i), { target: { value: 'Alice' } });
		await fireEvent.input(screen.getByLabelText(/date of birth/i), {
			target: { value: '2025-06-01' }
		});
		await fireEvent.change(screen.getByLabelText(/sex/i), { target: { value: 'female' } });
		await fireEvent.click(screen.getByRole('button', { name: /create baby/i }));

		expect(oncreate).toHaveBeenCalledWith({
			name: 'Alice',
			date_of_birth: '2025-06-01',
			sex: 'female',
			diagnosis_date: undefined,
			kasai_date: undefined
		});
	});

	it('passes onjoin callback through to JoinBabyForm', async () => {
		render(FirstLogin, { props: { oncreate, onjoin } });

		await fireEvent.click(screen.getByRole('button', { name: /join.*invite|enter.*invite/i }));

		await fireEvent.input(screen.getByLabelText(/invite code/i), {
			target: { value: 'ABC123' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /join/i }));

		expect(onjoin).toHaveBeenCalledWith('ABC123');
	});
});

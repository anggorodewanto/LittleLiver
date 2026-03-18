import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import CreateBabyForm from '$lib/components/CreateBabyForm.svelte';

describe('CreateBabyForm', () => {
	let oncreate: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		oncreate = vi.fn();
	});

	it('renders name, DOB, sex, diagnosis date, and kasai date fields', () => {
		render(CreateBabyForm, { props: { oncreate } });

		expect(screen.getByLabelText(/name/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/date of birth/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/sex/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/diagnosis date/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/kasai date/i)).toBeInTheDocument();
	});

	it('renders a submit button', () => {
		render(CreateBabyForm, { props: { oncreate } });

		expect(screen.getByRole('button', { name: /create baby/i })).toBeInTheDocument();
	});

	it('sex field has male and female options', () => {
		render(CreateBabyForm, { props: { oncreate } });

		const select = screen.getByLabelText(/sex/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('male');
		expect(options).toContain('female');
	});

	it('shows validation error when submitting without required fields', async () => {
		render(CreateBabyForm, { props: { oncreate } });

		await fireEvent.click(screen.getByRole('button', { name: /create baby/i }));

		expect(screen.getByText(/name is required/i)).toBeInTheDocument();
		expect(oncreate).not.toHaveBeenCalled();
	});

	it('shows validation error when DOB is missing', async () => {
		render(CreateBabyForm, { props: { oncreate } });

		const nameInput = screen.getByLabelText(/name/i);
		await fireEvent.input(nameInput, { target: { value: 'Alice' } });
		await fireEvent.click(screen.getByRole('button', { name: /create baby/i }));

		expect(screen.getByText(/date of birth is required/i)).toBeInTheDocument();
		expect(oncreate).not.toHaveBeenCalled();
	});

	it('shows validation error when sex is not selected', async () => {
		render(CreateBabyForm, { props: { oncreate } });

		const nameInput = screen.getByLabelText(/name/i);
		await fireEvent.input(nameInput, { target: { value: 'Alice' } });
		const dobInput = screen.getByLabelText(/date of birth/i);
		await fireEvent.input(dobInput, { target: { value: '2025-06-01' } });
		await fireEvent.click(screen.getByRole('button', { name: /create baby/i }));

		expect(screen.getByText(/sex is required/i)).toBeInTheDocument();
		expect(oncreate).not.toHaveBeenCalled();
	});

	it('calls oncreate with form data when valid', async () => {
		render(CreateBabyForm, { props: { oncreate } });

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

	it('includes optional fields when provided', async () => {
		render(CreateBabyForm, { props: { oncreate } });

		await fireEvent.input(screen.getByLabelText(/name/i), { target: { value: 'Bob' } });
		await fireEvent.input(screen.getByLabelText(/date of birth/i), {
			target: { value: '2025-09-01' }
		});
		await fireEvent.change(screen.getByLabelText(/sex/i), { target: { value: 'male' } });
		await fireEvent.input(screen.getByLabelText(/diagnosis date/i), {
			target: { value: '2025-09-15' }
		});
		await fireEvent.input(screen.getByLabelText(/kasai date/i), {
			target: { value: '2025-09-20' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /create baby/i }));

		expect(oncreate).toHaveBeenCalledWith({
			name: 'Bob',
			date_of_birth: '2025-09-01',
			sex: 'male',
			diagnosis_date: '2025-09-15',
			kasai_date: '2025-09-20'
		});
	});

	it('disables submit button when submitting prop is true', () => {
		render(CreateBabyForm, { props: { oncreate, submitting: true } });

		expect(screen.getByRole('button', { name: /creating/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(CreateBabyForm, { props: { oncreate, error: 'Something went wrong' } });

		expect(screen.getByText('Something went wrong')).toBeInTheDocument();
	});

	it('calls preventDefault on form submission', async () => {
		render(CreateBabyForm, { props: { oncreate } });

		await fireEvent.input(screen.getByLabelText(/name/i), { target: { value: 'Alice' } });
		await fireEvent.input(screen.getByLabelText(/date of birth/i), {
			target: { value: '2025-06-01' }
		});
		await fireEvent.change(screen.getByLabelText(/sex/i), { target: { value: 'female' } });

		const form = screen.getByRole('button', { name: /create baby/i }).closest('form')!;
		const submitEvent = new Event('submit', { bubbles: true, cancelable: true });
		const prevented = !form.dispatchEvent(submitEvent);

		expect(prevented).toBe(true);
	});
});

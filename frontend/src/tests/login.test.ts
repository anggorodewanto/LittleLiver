import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import LoginPage from '../routes/login/+page.svelte';

describe('Login page', () => {
	it('renders a heading with app name', () => {
		render(LoginPage);
		expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('LittleLiver');
	});

	it('renders a "Sign in with Google" button', () => {
		render(LoginPage);
		const button = screen.getByRole('link', { name: /sign in with google/i });
		expect(button).toBeInTheDocument();
	});

	it('Google sign-in link points to /auth/google/login', () => {
		render(LoginPage);
		const link = screen.getByRole('link', { name: /sign in with google/i });
		expect(link).toHaveAttribute('href', '/auth/google/login');
	});

	it('renders a subtitle describing the app', () => {
		render(LoginPage);
		expect(screen.getByText(/post-kasai baby health tracking/i)).toBeInTheDocument();
	});
});

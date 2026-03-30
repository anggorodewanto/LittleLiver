<div class="login-page">
	<h1>LittleLiver</h1>
	<p>Post-Kasai baby health tracking</p>
	<a href="/auth/google/login" class="login-btn" data-sveltekit-reload>Sign in with Google</a>
</div>

<style>
	.login-page {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		min-height: 60vh;
		text-align: center;
	}

	.login-page h1 {
		font-size: 2rem;
		color: var(--color-primary);
	}

	.login-page p {
		color: var(--color-text-muted);
		margin-bottom: var(--space-8);
	}

	.login-btn {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		background: var(--color-surface);
		border: 1.5px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: var(--space-3) var(--space-6);
		font-weight: 600;
		min-height: 48px;
		box-shadow: var(--shadow-sm);
		color: var(--color-text);
		text-decoration: none;
		transition: box-shadow 0.15s, border-color 0.15s;
	}

	.login-btn:hover {
		border-color: var(--color-primary);
		box-shadow: var(--shadow-md);
		text-decoration: none;
	}
</style>

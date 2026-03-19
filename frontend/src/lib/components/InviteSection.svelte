<script lang="ts">
	import { onDestroy } from 'svelte';

	interface Props {
		ongenerate: () => void;
		inviteCode?: string;
		expiresAt?: string;
		generating?: boolean;
		error?: string;
	}

	let {
		ongenerate,
		inviteCode = '',
		expiresAt = '',
		generating = false,
		error = ''
	}: Props = $props();

	let copied = $state(false);
	let countdown = $state('');
	let countdownInterval: ReturnType<typeof setInterval> | null = null;

	function computeCountdown(iso: string): string {
		if (!iso) {
			return '';
		}
		const now = Date.now();
		const expiry = new Date(iso).getTime();
		const diff = expiry - now;
		if (diff <= 0) {
			return 'Expired';
		}
		const hours = Math.floor(diff / (1000 * 60 * 60));
		const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
		return `${hours}h ${minutes}m`;
	}

	function startCountdown() {
		stopCountdown();
		if (!expiresAt) {
			return;
		}
		countdown = computeCountdown(expiresAt);
		countdownInterval = setInterval(() => {
			countdown = computeCountdown(expiresAt);
		}, 30000);
	}

	function stopCountdown() {
		if (countdownInterval !== null) {
			clearInterval(countdownInterval);
			countdownInterval = null;
		}
	}

	$effect(() => {
		if (expiresAt && inviteCode) {
			startCountdown();
		} else {
			stopCountdown();
			countdown = '';
		}
	});

	onDestroy(() => {
		stopCountdown();
	});

	async function copyCode() {
		if (!inviteCode) {
			return;
		}
		await navigator.clipboard.writeText(inviteCode);
		copied = true;
		setTimeout(() => {
			copied = false;
		}, 2000);
	}

	function formatExpiry(iso: string): string {
		if (!iso) {
			return '';
		}
		const date = new Date(iso);
		return date.toLocaleString();
	}
</script>

<section>
	<h3>Invite Code</h3>

	<button onclick={ongenerate} disabled={generating}>
		{generating ? 'Generating...' : 'Generate Invite Code'}
	</button>

	{#if inviteCode}
		<div>
			<p><strong>{inviteCode}</strong></p>
			<p>Expires {formatExpiry(expiresAt)}{#if countdown} ({countdown} remaining){/if}</p>
			<button onclick={copyCode}>
				{copied ? 'Copied!' : 'Copy Code'}
			</button>
		</div>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}
</section>

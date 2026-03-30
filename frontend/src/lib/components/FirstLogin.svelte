<script lang="ts">
	import CreateBabyForm from './CreateBabyForm.svelte';
	import JoinBabyForm from './JoinBabyForm.svelte';

	interface Props {
		oncreate: (data: {
			name: string;
			date_of_birth: string;
			sex: string;
			diagnosis_date?: string;
			kasai_date?: string;
		}) => void;
		onjoin: (code: string) => void;
		createSubmitting?: boolean;
		joinSubmitting?: boolean;
		createError?: string;
		joinError?: string;
	}

	let {
		oncreate,
		onjoin,
		createSubmitting = false,
		joinSubmitting = false,
		createError = '',
		joinError = ''
	}: Props = $props();

	let mode = $state<'choice' | 'create' | 'join'>('choice');
</script>

<div>
	<h1>Welcome to LittleLiver</h1>

	{#if mode === 'choice'}
		<p>Get started by creating a baby profile or joining an existing one.</p>
		<button onclick={() => (mode = 'create')}>Create a Baby</button>
		<button onclick={() => (mode = 'join')}>Join with Invite Code</button>
	{:else if mode === 'create'}
		<CreateBabyForm {oncreate} submitting={createSubmitting} error={createError} />
		<button onclick={() => (mode = 'choice')}>Back</button>
	{:else if mode === 'join'}
		<JoinBabyForm {onjoin} submitting={joinSubmitting} error={joinError} />
		<button onclick={() => (mode = 'choice')}>Back</button>
	{/if}
</div>

<style>
	div {
		text-align: center;
	}

	h1 {
		margin-bottom: var(--space-2);
	}

	p {
		color: var(--color-text-muted);
		margin-bottom: var(--space-6);
	}

	button {
		display: block;
		width: 100%;
		margin-bottom: var(--space-3);
		min-height: 52px;
		font-size: var(--font-size-lg);
		border-radius: var(--radius-md);
	}
</style>

<script lang="ts">
	import {
		activeBaby,
		babies,
		setActiveBaby,
		updateBaby,
		generateInvite,
		unlinkFromBaby,
		deleteAccount
	} from '$lib/stores/baby';
	import { apiClient } from '$lib/api';
	import type { UpdateBabyInput, InviteResponse } from '$lib/stores/baby';
	import BabySettingsForm from './BabySettingsForm.svelte';
	import BabySelector from './BabySelector.svelte';
	import InviteSection from './InviteSection.svelte';
	import UnlinkSection from './UnlinkSection.svelte';
	import AccountDeletion from './AccountDeletion.svelte';

	let loggingOut = $state(false);

	async function handleLogout(): Promise<void> {
		loggingOut = true;
		try {
			await apiClient.logout();
		} catch {
			// ignore — redirect to login regardless
		}
		window.location.href = '/login';
	}

	function extractError(err: unknown, fallback: string): string {
		return err instanceof Error ? err.message : fallback;
	}

	let baby = $derived($activeBaby);

	let settingsSubmitting = $state(false);
	let settingsError = $state('');

	let inviteCode = $state('');
	let inviteExpiresAt = $state('');
	let inviteGenerating = $state(false);
	let inviteError = $state('');

	let unlinkError = $state('');
	let deleteError = $state('');

	async function handleSave(data: UpdateBabyInput, recalculate: boolean) {
		if (!baby) {
			return;
		}
		settingsSubmitting = true;
		settingsError = '';
		try {
			await updateBaby(baby.id, data, recalculate);
		} catch (err) {
			settingsError = extractError(err, 'Failed to save');
		} finally {
			settingsSubmitting = false;
		}
	}

	async function handleGenerateInvite() {
		if (!baby) {
			return;
		}
		inviteGenerating = true;
		inviteError = '';
		try {
			const result: InviteResponse = await generateInvite(baby.id);
			inviteCode = result.code;
			inviteExpiresAt = result.expires_at;
		} catch (err) {
			inviteError = extractError(err, 'Failed to generate invite');
		} finally {
			inviteGenerating = false;
		}
	}

	async function handleUnlink() {
		if (!baby) {
			return;
		}
		unlinkError = '';
		try {
			await unlinkFromBaby(baby.id);
		} catch (err) {
			unlinkError = extractError(err, 'Failed to unlink');
		}
	}

	async function handleDeleteAccount() {
		deleteError = '';
		try {
			await deleteAccount();
			window.location.href = '/login';
		} catch (err) {
			deleteError = extractError(err, 'Failed to delete account');
		}
	}
</script>

<h1>Settings</h1>

{#if $babies.length > 0}
	<section>
		<h2>Active Baby</h2>
		<BabySelector
			babies={$babies}
			activeBabyId={baby?.id ?? null}
			onselect={setActiveBaby}
		/>
	</section>
{/if}

{#if baby}
	<section>
		<h2>Baby Settings</h2>
		{#key baby.id}
			<BabySettingsForm
				{baby}
				onsave={handleSave}
				submitting={settingsSubmitting}
				error={settingsError}
			/>
		{/key}
	</section>

	<InviteSection
		ongenerate={handleGenerateInvite}
		{inviteCode}
		expiresAt={inviteExpiresAt}
		generating={inviteGenerating}
		error={inviteError}
	/>

	<UnlinkSection babyName={baby.name} onunlink={handleUnlink} error={unlinkError} />
{:else}
	<p>No baby selected. Please create or join a baby profile first.</p>
{/if}

{#if baby}
	<section>
		<h2>Reports</h2>
		<a href="/report" class="settings-link">Generate Clinical Report</a>
	</section>
{/if}

<AccountDeletion ondelete={handleDeleteAccount} error={deleteError} />

<section class="logout-section">
	<button class="logout-btn" onclick={handleLogout} disabled={loggingOut}>
		{loggingOut ? 'Logging out...' : 'Log Out'}
	</button>
</section>

<style>
	.settings-link {
		display: block;
		padding: var(--space-3) var(--space-4);
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		color: var(--color-primary);
		font-weight: 500;
		text-decoration: none;
	}

	.settings-link:hover {
		background: var(--color-primary-light);
		text-decoration: none;
	}

	.logout-section {
		padding-top: var(--space-4);
		border-top: 1px solid var(--color-border);
	}

	.logout-btn {
		width: 100%;
		min-height: 48px;
		background: var(--color-surface);
		color: var(--color-error);
		border: 1.5px solid var(--color-border);
		border-radius: var(--radius-md);
		font-weight: 600;
	}

	.logout-btn:hover {
		background: var(--color-error-bg);
		border-color: var(--color-error);
	}
</style>

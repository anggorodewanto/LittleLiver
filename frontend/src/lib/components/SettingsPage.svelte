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
	import type { UpdateBabyInput, InviteResponse } from '$lib/stores/baby';
	import BabySettingsForm from './BabySettingsForm.svelte';
	import BabySelector from './BabySelector.svelte';
	import InviteSection from './InviteSection.svelte';
	import UnlinkSection from './UnlinkSection.svelte';
	import AccountDeletion from './AccountDeletion.svelte';

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
			settingsError = err instanceof Error ? err.message : 'Failed to save';
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
			inviteError = err instanceof Error ? err.message : 'Failed to generate invite';
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
			unlinkError = err instanceof Error ? err.message : 'Failed to unlink';
		}
	}

	async function handleDeleteAccount() {
		deleteError = '';
		try {
			await deleteAccount();
			window.location.href = '/login';
		} catch (err) {
			deleteError = err instanceof Error ? err.message : 'Failed to delete account';
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
		<BabySettingsForm
			{baby}
			onsave={handleSave}
			submitting={settingsSubmitting}
			error={settingsError}
		/>
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

<AccountDeletion ondelete={handleDeleteAccount} error={deleteError} />

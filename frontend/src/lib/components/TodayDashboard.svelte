<script lang="ts">
	import { untrack } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { goto } from '$app/navigation';
	import { apiClient } from '$lib/api';
	import { stoolStatusColor } from '$lib/stool-colors';
	import type { MetricType } from './QuickLogButtons.svelte';
	import QuickLogButtons from './QuickLogButtons.svelte';

	interface Props {
		babyId: string;
	}

	interface SummaryCards {
		total_feeds: number;
		total_calories: number;
		total_wet_diapers: number;
		total_stools: number;
		worst_stool_color: number | null;
		last_temperature: number | null;
		last_weight: number | null;
	}

	interface StoolColorTrendEntry {
		date: string;
		color: string;
		color_rating: number;
	}

	interface UpcomingMed {
		id: string;
		name: string;
		dose: string;
		frequency: string;
		schedule_times: string[];
		timezone: string | null;
		interval_days: number | null;
		next_dose_at: string | null;
	}

	interface Alert {
		entry_id: string;
		alert_type: string;
		method?: string;
		value: unknown;
		timestamp: string;
		medication_id?: string;
		medication_name?: string;
	}

	interface DashboardResponse {
		summary_cards: SummaryCards;
		stool_color_trend: StoolColorTrendEntry[];
		upcoming_meds: UpcomingMed[];
		active_alerts: Alert[];
		chart_data_series: unknown;
	}

	let { babyId }: Props = $props();

	let loading = $state(true);
	let error = $state<string | null>(null);
	let dashboard = $state<DashboardResponse | null>(null);
	let dismissedAlertIds = new SvelteSet<string>();
	let now = $state(Date.now());

	function loadDismissedAlerts(): SvelteSet<string> {
		const stored = localStorage.getItem('dismissed_alerts');
		if (!stored) {
			return new SvelteSet();
		}
		try {
			const arr = JSON.parse(stored) as string[];
			return new SvelteSet(arr);
		} catch {
			return new SvelteSet();
		}
	}

	function saveDismissedAlerts(ids: SvelteSet<string>): void {
		localStorage.setItem('dismissed_alerts', JSON.stringify([...ids]));
	}

	function cleanupDismissedAlerts(
		activeAlerts: Alert[],
		dismissed: SvelteSet<string>
	): SvelteSet<string> {
		const activeEntryIds = new Set(activeAlerts.map((a) => a.entry_id));
		const cleaned = new SvelteSet<string>();
		for (const id of dismissed) {
			if (activeEntryIds.has(id)) {
				cleaned.add(id);
			}
		}
		return cleaned;
	}

	function dismissAlert(entryId: string): void {
		dismissedAlertIds.add(entryId);
		saveDismissedAlerts(dismissedAlertIds);
	}

	function formatCountdown(nextDoseAt: string, frequency?: string): string {
		const doseTime = new Date(nextDoseAt).getTime();
		const diffMs = doseTime - now;

		// For every_x_days, display in whole-day terms
		if (frequency === 'every_x_days') {
			return formatDayCountdown(doseTime);
		}

		if (diffMs < 0) {
			const overdue = Math.abs(diffMs);
			const hours = Math.floor(overdue / (60 * 60 * 1000));
			const mins = Math.floor((overdue % (60 * 60 * 1000)) / (60 * 1000));
			if (hours > 0) {
				return `overdue by ${hours} h ${mins} min`;
			}
			return `overdue by ${mins} min`;
		}

		const hours = Math.floor(diffMs / (60 * 60 * 1000));
		const mins = Math.floor((diffMs % (60 * 60 * 1000)) / (60 * 1000));
		if (hours > 0) {
			return `in ${hours} h ${mins} min`;
		}
		return `in ${mins} min`;
	}

	function formatDayCountdown(doseTime: number): string {
		const today = new Date(now);
		today.setHours(0, 0, 0, 0);
		const doseDate = new Date(doseTime);
		doseDate.setHours(0, 0, 0, 0);

		const diffDays = Math.round((doseDate.getTime() - today.getTime()) / (24 * 60 * 60 * 1000));

		if (diffDays === 0) {
			return 'Due today';
		}
		if (diffDays < 0) {
			const overdueDays = Math.abs(diffDays);
			return `Overdue by ${overdueDays} day${overdueDays !== 1 ? 's' : ''}`;
		}
		return `Due in ${diffDays} day${diffDays !== 1 ? 's' : ''}`;
	}

	function alertLabel(alertType: string): string {
		switch (alertType) {
			case 'acholic_stool':
				return 'Acholic Stool';
			case 'fever':
				return 'Fever';
			case 'jaundice_worsening':
				return 'Jaundice Worsening';
			case 'missed_medication':
				return 'Missed Medication';
			default:
				return alertType;
		}
	}

	function alertMessage(alert: Alert): string {
		switch (alert.alert_type) {
			case 'acholic_stool':
				return 'Acholic stool detected (color rating: ' + alert.value + '). Contact your hepatology team — this may indicate bile flow failure.';
			case 'fever':
				return 'Fever detected' + (alert.method ? ` (${alert.method})` : '') + '. Contact your hepatology team immediately. Fever after Kasai can indicate cholangitis.';
			case 'jaundice_worsening':
				return 'Worsening jaundice detected. Contact your hepatology team.';
			case 'missed_medication':
				return alert.medication_name
					? `${alert.medication_name} dose was missed. Tap to log.`
					: 'A scheduled medication dose was missed. Tap to log.';
			default:
				return '';
		}
	}

	let visibleAlerts = $derived(
		(dashboard?.active_alerts ?? []).filter((a) => !dismissedAlertIds.has(a.entry_id))
	);

	// Medications due within -60min (overdue) to +30min (upcoming)
	// For every_x_days meds: due if next_dose_at is today or earlier
	let dueNowMeds = $derived(
		(dashboard?.upcoming_meds ?? []).filter((med) => {
			if (!med.next_dose_at) return false;

			if (med.frequency === 'every_x_days') {
				const doseDate = new Date(med.next_dose_at);
				doseDate.setHours(0, 0, 0, 0);
				const today = new Date(now);
				today.setHours(0, 0, 0, 0);
				return doseDate.getTime() <= today.getTime();
			}

			const diffMs = new Date(med.next_dose_at).getTime() - now;
			return diffMs >= -60 * 60 * 1000 && diffMs <= 30 * 60 * 1000;
		})
	);

	function handleQuickLog(type: MetricType): void {
		if (type === 'med_given') {
			void goto('/log/med');
			return;
		}
		void goto(`/log/${type}`);
	}


	async function fetchDashboard(): Promise<void> {
		loading = true;
		error = null;
		try {
			const data = await apiClient.get<DashboardResponse>(`/babies/${babyId}/dashboard`);
			dashboard = data;

			// Load dismissed, clean up stale, save back
			const dismissed = loadDismissedAlerts();
			if (dismissed.size > 0) {
				const cleaned = cleanupDismissedAlerts(data.active_alerts, dismissed);
				dismissedAlertIds.clear();
				for (const id of cleaned) {
					dismissedAlertIds.add(id);
				}
				if (cleaned.size !== dismissed.size) {
					saveDismissedAlerts(dismissedAlertIds);
				}
			}
		} catch {
			error = 'Failed to load dashboard data';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		const _id = babyId;
		untrack(() => { void fetchDashboard(); });
	});

	$effect(() => {
		const interval = setInterval(() => {
			now = Date.now();
		}, 60000);
		return () => clearInterval(interval);
	});

	$effect(() => {
		function onVisibilityChange() {
			if (document.visibilityState === 'visible') {
				void fetchDashboard();
			}
		}
		document.addEventListener('visibilitychange', onVisibilityChange);
		return () => document.removeEventListener('visibilitychange', onVisibilityChange);
	});
</script>

<div class="dashboard">
{#if loading}
	<div class="loading">Loading...</div>
{:else if error}
	<div class="error">{error}</div>
{:else if dashboard}
	<!-- Alert Banners -->
	{#if visibleAlerts.length > 0}
		<div class="alert-banners">
			{#each visibleAlerts as alert (alert.entry_id)}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div
					class="alert-banner"
					class:alert-tappable={alert.alert_type === 'missed_medication'}
					data-alert-type={alert.alert_type}
					onclick={(e: MouseEvent) => {
						if (alert.alert_type === 'missed_medication' && !(e.target as HTMLElement).closest('button')) {
							let url = alert.medication_id ? `/log/med?medicationId=${alert.medication_id}` : '/log/med';
							if (alert.value) {
								const sep = url.includes('?') ? '&' : '?';
								url += `${sep}scheduled_time=${encodeURIComponent(String(alert.value))}`;
							}
							void goto(url);
						}
					}}
					onkeydown={(e: KeyboardEvent) => {
						if (alert.alert_type === 'missed_medication' && (e.key === 'Enter' || e.key === ' ') && !(e.target as HTMLElement).closest('button')) {
							e.preventDefault();
							let url = alert.medication_id ? `/log/med?medicationId=${alert.medication_id}` : '/log/med';
							if (alert.value) {
								const sep = url.includes('?') ? '&' : '?';
								url += `${sep}scheduled_time=${encodeURIComponent(String(alert.value))}`;
							}
							void goto(url);
						}
					}}
				>
					<div class="alert-content">
						<strong class="alert-label">{alertLabel(alert.alert_type)}</strong>
						<span class="alert-message">{alertMessage(alert)}</span>
					</div>
					<button type="button" onclick={() => dismissAlert(alert.entry_id)}>Dismiss</button>
				</div>
			{/each}
		</div>
	{/if}

	<!-- Due Now -->
	{#if dueNowMeds.length > 0}
		{#each dueNowMeds as med (med.id)}
			<!-- svelte-ignore a11y_click_events_have_key_events -->
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div
				class="due-now-banner"
				data-testid="due-now-banner"
				onclick={() => void goto(`/log/med?medicationId=${med.id}`)}
			>
				<div class="due-now-label">Due Now</div>
				<div class="due-now-info">
					<span class="due-now-name">{med.name}</span>
					<span class="due-now-dose">{med.dose}</span>
				</div>
				<div class="due-now-countdown">
					{formatCountdown(med.next_dose_at!, med.frequency)}
				</div>
			</div>
		{/each}
	{/if}

	<!-- Summary Cards -->
	<div class="summary-cards">
		<div class="card">
			<div class="card-value">{dashboard.summary_cards.total_feeds}</div>
			<div class="card-label">Feeds</div>
		</div>
		<div class="card">
			<div class="card-value">{Math.round(dashboard.summary_cards.total_calories)}</div>
			<div class="card-label">Calories</div>
		</div>
		<div class="card">
			<div class="card-value">{dashboard.summary_cards.total_wet_diapers}</div>
			<div class="card-label">Wet Diapers</div>
		</div>
		<div class="card">
			<div class="card-value">
				{dashboard.summary_cards.total_stools}
				{#if dashboard.summary_cards.worst_stool_color !== null}
					<span
						class="stool-color-indicator"
						style="background-color: {stoolStatusColor(dashboard.summary_cards.worst_stool_color)}"
					></span>
				{/if}
			</div>
			<div class="card-label">Stools</div>
		</div>
		<div class="card">
			<div class="card-value">
				{#if dashboard.summary_cards.last_temperature !== null}
					{dashboard.summary_cards.last_temperature} °C
				{:else}
					—
				{/if}
			</div>
			<div class="card-label">Last Temp</div>
		</div>
		<div class="card">
			<div class="card-value">
				{#if dashboard.summary_cards.last_weight !== null}
					{dashboard.summary_cards.last_weight} kg
				{:else}
					—
				{/if}
			</div>
			<div class="card-label">Last Weight</div>
		</div>
	</div>

	<!-- Stool Color Trend -->
	{#if dashboard.stool_color_trend.length > 0}
		<div class="stool-color-trend">
			<h3>Stool Color Trend (7 days)</h3>
			<div class="trend-dots">
				{#each dashboard.stool_color_trend as entry, i (entry.date + '-' + i)}
					<div
						class="stool-trend-dot"
						data-rating={entry.color_rating}
						style="background-color: {stoolStatusColor(entry.color_rating)}"
						title="{entry.date}: {entry.color} ({entry.color_rating})"
					></div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Upcoming Medications -->
	{#if dashboard.upcoming_meds.length > 0}
		<div class="upcoming-meds">
			<h3>Upcoming Medications</h3>
			{#each dashboard.upcoming_meds as med (med.id)}
				<div class="med-item">
					<div class="med-info">
						<span class="med-name">{med.name}</span>
						<span class="med-dose">{med.dose}</span>
					</div>
					<div class="med-countdown">
						{#if med.next_dose_at}
							{formatCountdown(med.next_dose_at, med.frequency)}
						{:else}
							No schedule
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}

	<!-- Quick Log Buttons -->
	<div class="quick-log-section">
		<QuickLogButtons onselect={handleQuickLog} onnavigate={(path) => goto(path)} />
	</div>
{/if}
</div>

<style>
	.dashboard {
		padding-top: var(--space-2);
	}

	.alert-banners {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}

	.alert-banner {
		display: flex;
		align-items: flex-start;
		gap: var(--space-3);
		padding: var(--space-3) var(--space-4);
		border-radius: var(--radius-md);
		border-left: 4px solid;
		background: var(--color-warning-bg);
		border-color: var(--color-warning);
	}

	.alert-banner[data-alert-type="acholic_stool"] {
		background: var(--color-alert-acholic-bg);
		border-color: var(--color-alert-acholic);
	}

	.alert-banner[data-alert-type="fever"] {
		background: var(--color-alert-fever-bg);
		border-color: var(--color-alert-fever);
	}

	.alert-banner[data-alert-type="jaundice_worsening"] {
		background: var(--color-alert-jaundice-bg);
		border-color: var(--color-alert-jaundice);
	}

	.alert-banner[data-alert-type="missed_medication"] {
		background: var(--color-alert-missed-med-bg);
		border-color: var(--color-alert-missed-med);
	}

	.alert-tappable {
		cursor: pointer;
	}

	.alert-content {
		flex: 1;
	}

	.alert-label {
		display: block;
		font-size: var(--font-size-sm);
		margin-bottom: var(--space-1);
	}

	.alert-message {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
		line-height: 1.4;
	}

	.alert-banner button {
		flex-shrink: 0;
		font-size: var(--font-size-xs);
		padding: var(--space-1) var(--space-2);
		min-height: 32px;
		background: transparent;
		color: var(--color-text-muted);
		border: 1px solid var(--color-border);
	}

	.due-now-banner {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3) var(--space-4);
		margin-bottom: var(--space-4);
		border-radius: var(--radius-md);
		background: var(--color-primary);
		color: white;
		cursor: pointer;
		box-shadow: var(--shadow-sm);
	}

	.due-now-label {
		font-size: var(--font-size-xs);
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		background: rgba(255, 255, 255, 0.2);
		padding: var(--space-1) var(--space-2);
		border-radius: var(--radius-sm);
		white-space: nowrap;
	}

	.due-now-info {
		flex: 1;
		display: flex;
		flex-direction: column;
	}

	.due-now-name {
		font-weight: 600;
	}

	.due-now-dose {
		font-size: var(--font-size-sm);
		opacity: 0.85;
	}

	.due-now-countdown {
		font-size: var(--font-size-sm);
		font-weight: 500;
		opacity: 0.9;
		text-align: right;
		white-space: nowrap;
	}

	.summary-cards {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--space-3);
		margin-bottom: var(--space-6);
	}

	.card {
		background: var(--color-surface);
		border-radius: var(--radius-md);
		padding: var(--space-4);
		text-align: center;
		box-shadow: var(--shadow-sm);
		border: 1px solid var(--color-border);
	}

	.card-value {
		font-size: var(--font-size-2xl);
		font-weight: 700;
		color: var(--color-text);
		display: flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
	}

	.card-label {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
		margin-top: var(--space-1);
	}

	.stool-color-indicator {
		display: inline-block;
		width: 14px;
		height: 14px;
		border-radius: var(--radius-full);
		border: 1px solid rgba(0, 0, 0, 0.1);
	}

	.stool-color-trend {
		background: var(--color-surface);
		border-radius: var(--radius-md);
		padding: var(--space-4);
		margin-bottom: var(--space-6);
		border: 1px solid var(--color-border);
		box-shadow: var(--shadow-sm);
	}

	.trend-dots {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		padding-top: var(--space-2);
	}

	.stool-trend-dot {
		width: 24px;
		height: 24px;
		border-radius: var(--radius-full);
		border: 1px solid rgba(0, 0, 0, 0.1);
	}

	.upcoming-meds {
		background: var(--color-surface);
		border-radius: var(--radius-md);
		padding: var(--space-4);
		margin-bottom: var(--space-6);
		border: 1px solid var(--color-border);
		box-shadow: var(--shadow-sm);
	}

	.med-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: var(--space-3) 0;
		border-bottom: 1px solid var(--color-border);
	}

	.med-item:last-child {
		border-bottom: none;
	}

	.med-info {
		display: flex;
		flex-direction: column;
	}

	.med-name {
		font-weight: 600;
	}

	.med-dose {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
	}

	.med-countdown {
		font-size: var(--font-size-sm);
		font-weight: 500;
		color: var(--color-primary);
		text-align: right;
	}

	.quick-log-section {
		margin-bottom: var(--space-4);
	}
</style>

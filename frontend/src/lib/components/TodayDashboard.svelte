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
		next_dose_at: string | null;
	}

	interface Alert {
		entry_id: string;
		alert_type: string;
		method?: string;
		value: unknown;
		timestamp: string;
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

	function formatCountdown(nextDoseAt: string): string {
		const doseTime = new Date(nextDoseAt).getTime();
		const diffMs = doseTime - now;

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

	let visibleAlerts = $derived(
		(dashboard?.active_alerts ?? []).filter((a) => !dismissedAlertIds.has(a.entry_id))
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
</script>

{#if loading}
	<div class="loading">Loading...</div>
{:else if error}
	<div class="error">{error}</div>
{:else if dashboard}
	<!-- Alert Banners -->
	{#if visibleAlerts.length > 0}
		<div class="alert-banners">
			{#each visibleAlerts as alert (alert.entry_id)}
				<div class="alert-banner alert-{alert.alert_type}">
					<span class="alert-message">{alertLabel(alert.alert_type)}</span>
					<button type="button" onclick={() => dismissAlert(alert.entry_id)}>Dismiss</button>
				</div>
			{/each}
		</div>
	{/if}

	<!-- Summary Cards -->
	<div class="summary-cards">
		<div class="card">
			<div class="card-value">{dashboard.summary_cards.total_feeds}</div>
			<div class="card-label">Feeds</div>
		</div>
		<div class="card">
			<div class="card-value">{dashboard.summary_cards.total_calories}</div>
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
				{#each dashboard.stool_color_trend as entry (entry.date)}
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
							{formatCountdown(med.next_dose_at)}
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

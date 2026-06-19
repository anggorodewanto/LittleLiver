// Package immunization holds the static IDAI childhood immunization reference
// schedule used to compute a baby's completed and upcoming vaccinations.
//
// Source: Jadwal Imunisasi Anak IDAI (Ikatan Dokter Anak Indonesia), 2023/2024
// schedule, cross-referenced with the Indonesian national program (Kemenkes)
// for the mandatory-vs-optional classification.
//
//   - Mandatory  = part of the universal national infant program ("imunisasi
//     program / wajib"): HB, BCG, Polio (OPV+IPV), DTP-HB-Hib, PCV, Rotavirus,
//     MR. These are government-funded and recommended for every child.
//   - Optional   = IDAI-recommended additional vaccines ("imunisasi pilihan /
//     dianjurkan"): Influenza, JE, Varicella, Hepatitis A, Typhoid, HPV, Dengue.
//
// NOTE: This is reference data for a personal-use app, not medical advice. The
// wajib/optional line shifts as antigens move into the national program (PCV,
// Rotavirus, HPV, JE were recently added, regionally phased). Verify ages and
// classification against current official IDAI/Kemenkes guidance before relying
// on it. AgeMonths is the canonical due age (0 = at birth); AgeLabel is the
// human-readable range shown in the UI.
package immunization

// ScheduleEntry is a single dose slot in the reference schedule.
type ScheduleEntry struct {
	Code       string `json:"code"`        // vaccine code, e.g. "DTP_HB_HIB"
	Name       string `json:"name"`        // display name
	DoseNumber int    `json:"dose_number"` // 1-based dose index within the vaccine
	DoseLabel  string `json:"dose_label"`  // e.g. "Dose 1", "Booster", "HB-0"
	AgeMonths  int    `json:"age_months"`  // recommended age in months (0 = at birth)
	AgeLabel   string `json:"age_label"`   // human label, e.g. "2 months"
	Mandatory  bool   `json:"mandatory"`   // true = national program; false = optional
}

// schedule is ordered by recommended age, then by vaccine for stable display.
var schedule = []ScheduleEntry{
	// --- At birth ---
	{Code: "HB0", Name: "Hepatitis B (birth dose)", DoseNumber: 1, DoseLabel: "HB-0", AgeMonths: 0, AgeLabel: "At birth (<24h)", Mandatory: true},
	{Code: "BCG", Name: "BCG", DoseNumber: 1, DoseLabel: "Single dose", AgeMonths: 0, AgeLabel: "At birth – 1 month", Mandatory: true},
	{Code: "POLIO", Name: "Polio (OPV)", DoseNumber: 1, DoseLabel: "OPV-0", AgeMonths: 0, AgeLabel: "At birth", Mandatory: true},

	// --- 2 months ---
	{Code: "DTP_HB_HIB", Name: "DTP-HB-Hib (Pentavalent)", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 2, AgeLabel: "2 months", Mandatory: true},
	{Code: "POLIO", Name: "Polio (OPV)", DoseNumber: 2, DoseLabel: "OPV-1", AgeMonths: 2, AgeLabel: "2 months", Mandatory: true},
	{Code: "PCV", Name: "PCV (Pneumococcal)", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 2, AgeLabel: "2 months", Mandatory: true},
	{Code: "RV", Name: "Rotavirus", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 2, AgeLabel: "2 months (from 6 weeks)", Mandatory: true},

	// --- 3 months ---
	{Code: "DTP_HB_HIB", Name: "DTP-HB-Hib (Pentavalent)", DoseNumber: 2, DoseLabel: "Dose 2", AgeMonths: 3, AgeLabel: "3 months", Mandatory: true},
	{Code: "POLIO", Name: "Polio (OPV)", DoseNumber: 3, DoseLabel: "OPV-2", AgeMonths: 3, AgeLabel: "3 months", Mandatory: true},

	// --- 4 months ---
	{Code: "DTP_HB_HIB", Name: "DTP-HB-Hib (Pentavalent)", DoseNumber: 3, DoseLabel: "Dose 3", AgeMonths: 4, AgeLabel: "4 months", Mandatory: true},
	{Code: "POLIO", Name: "Polio (OPV)", DoseNumber: 4, DoseLabel: "OPV-3", AgeMonths: 4, AgeLabel: "4 months", Mandatory: true},
	{Code: "IPV", Name: "Polio (IPV, injectable)", DoseNumber: 1, DoseLabel: "IPV-1", AgeMonths: 4, AgeLabel: "4 months", Mandatory: true},
	{Code: "PCV", Name: "PCV (Pneumococcal)", DoseNumber: 2, DoseLabel: "Dose 2", AgeMonths: 4, AgeLabel: "4 months", Mandatory: true},
	{Code: "RV", Name: "Rotavirus", DoseNumber: 2, DoseLabel: "Dose 2", AgeMonths: 4, AgeLabel: "4 months", Mandatory: true},

	// --- 6 months ---
	{Code: "FLU", Name: "Influenza", DoseNumber: 1, DoseLabel: "First dose", AgeMonths: 6, AgeLabel: "6 months (then yearly)", Mandatory: false},

	// --- 9 months ---
	{Code: "MR", Name: "Measles-Rubella (MR)", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 9, AgeLabel: "9 months", Mandatory: true},
	{Code: "IPV", Name: "Polio (IPV, injectable)", DoseNumber: 2, DoseLabel: "IPV-2", AgeMonths: 9, AgeLabel: "9 months", Mandatory: true},
	{Code: "JE", Name: "Japanese Encephalitis", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 9, AgeLabel: "9–12 months (endemic areas)", Mandatory: false},

	// --- 12 months ---
	{Code: "PCV", Name: "PCV (Pneumococcal)", DoseNumber: 3, DoseLabel: "Booster", AgeMonths: 12, AgeLabel: "12–15 months", Mandatory: true},
	{Code: "HEPA", Name: "Hepatitis A", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 12, AgeLabel: "12–24 months", Mandatory: false},
	{Code: "VAR", Name: "Varicella (Chickenpox)", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 12, AgeLabel: "12–18 months", Mandatory: false},

	// --- 18 months ---
	{Code: "DTP_HB_HIB", Name: "DTP-HB-Hib (Pentavalent)", DoseNumber: 4, DoseLabel: "Booster", AgeMonths: 18, AgeLabel: "18 months", Mandatory: true},
	{Code: "MR", Name: "Measles-Rubella (MR)", DoseNumber: 2, DoseLabel: "Dose 2", AgeMonths: 18, AgeLabel: "18 months", Mandatory: true},
	{Code: "VAR", Name: "Varicella (Chickenpox)", DoseNumber: 2, DoseLabel: "Dose 2", AgeMonths: 18, AgeLabel: "from 6 weeks after dose 1", Mandatory: false},
	{Code: "HEPA", Name: "Hepatitis A", DoseNumber: 2, DoseLabel: "Dose 2", AgeMonths: 18, AgeLabel: "6–18 months after dose 1", Mandatory: false},

	// --- 24 months ---
	{Code: "TYPH", Name: "Typhoid", DoseNumber: 1, DoseLabel: "First dose", AgeMonths: 24, AgeLabel: "2 years (repeat every 3 yr)", Mandatory: false},

	// --- Older children ---
	{Code: "DTP", Name: "DTP (booster)", DoseNumber: 1, DoseLabel: "Booster", AgeMonths: 60, AgeLabel: "5–7 years", Mandatory: true},
	{Code: "MR", Name: "Measles-Rubella (MR)", DoseNumber: 3, DoseLabel: "Dose 3", AgeMonths: 60, AgeLabel: "5–7 years", Mandatory: true},
	{Code: "DENG", Name: "Dengue", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 72, AgeLabel: "from 6 years", Mandatory: false},
	{Code: "DENG", Name: "Dengue", DoseNumber: 2, DoseLabel: "Dose 2", AgeMonths: 75, AgeLabel: "3 months after dose 1", Mandatory: false},
	{Code: "HPV", Name: "HPV", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 108, AgeLabel: "9–14 years (girls)", Mandatory: false},
	{Code: "HPV", Name: "HPV", DoseNumber: 2, DoseLabel: "Dose 2", AgeMonths: 114, AgeLabel: "6–15 months after dose 1", Mandatory: false},
}

// Schedule returns a copy of the reference immunization schedule.
func Schedule() []ScheduleEntry {
	out := make([]ScheduleEntry, len(schedule))
	copy(out, schedule)
	return out
}

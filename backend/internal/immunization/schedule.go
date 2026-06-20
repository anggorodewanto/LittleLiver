// Package immunization holds the static Indonesian childhood immunization
// reference schedule used to compute a baby's completed and upcoming
// vaccinations.
//
// Sources (verified):
//   - Kemenkes national program ("imunisasi rutin lengkap") — KMK No.
//     HK.01.07/MENKES/1098/2024 as amended by KMK No. 35/2025 (HPV single
//     dose); live schedule at ayosehat.kemkes.go.id. Used for the MANDATORY
//     vaccines and their ages, since that is the "wajib" schedule a child
//     actually receives.
//   - IDAI 2024 ("Jadwal Imunisasi Anak Usia 0–18 Tahun") — used for the
//     OPTIONAL ("pilihan" / recommended-additional) vaccines.
//
// Classification:
//   - Mandatory  = universal national program: HB, BCG, Polio (OPV+IPV),
//     DTP-HB-Hib, PCV, Rotavirus, MR, and HPV (girls). Government-funded,
//     for every child.
//   - Optional   = IDAI-recommended additions: Influenza, JE (program but
//     endemic-areas-only, so optional nationally), Varicella, Hepatitis A,
//     Typhoid, the DTaP 5–7y booster, and Dengue.
//
// NOTE: This is reference data for a personal-use app, not medical advice.
// Ages and classification shift as the program changes; verify against current
// official IDAI/Kemenkes guidance. This data table is the single place to
// adjust them. AgeMonths is the canonical due age (0 = at birth); AgeLabel is
// the human-readable range shown in the UI.
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

	// --- 1 month ---
	{Code: "BCG", Name: "BCG", DoseNumber: 1, DoseLabel: "Single dose", AgeMonths: 1, AgeLabel: "1 month", Mandatory: true},
	{Code: "POLIO", Name: "Polio (OPV)", DoseNumber: 1, DoseLabel: "OPV-1", AgeMonths: 1, AgeLabel: "1 month", Mandatory: true},

	// --- 2 months ---
	{Code: "DTP_HB_HIB", Name: "DTP-HB-Hib (Pentavalent)", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 2, AgeLabel: "2 months", Mandatory: true},
	{Code: "POLIO", Name: "Polio (OPV)", DoseNumber: 2, DoseLabel: "OPV-2", AgeMonths: 2, AgeLabel: "2 months", Mandatory: true},
	{Code: "PCV", Name: "PCV (Pneumococcal)", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 2, AgeLabel: "2 months", Mandatory: true},
	{Code: "RV", Name: "Rotavirus", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 2, AgeLabel: "2 months", Mandatory: true},

	// --- 3 months ---
	{Code: "DTP_HB_HIB", Name: "DTP-HB-Hib (Pentavalent)", DoseNumber: 2, DoseLabel: "Dose 2", AgeMonths: 3, AgeLabel: "3 months", Mandatory: true},
	{Code: "POLIO", Name: "Polio (OPV)", DoseNumber: 3, DoseLabel: "OPV-3", AgeMonths: 3, AgeLabel: "3 months", Mandatory: true},
	{Code: "PCV", Name: "PCV (Pneumococcal)", DoseNumber: 2, DoseLabel: "Dose 2", AgeMonths: 3, AgeLabel: "3 months", Mandatory: true},
	{Code: "RV", Name: "Rotavirus", DoseNumber: 2, DoseLabel: "Dose 2", AgeMonths: 3, AgeLabel: "3 months", Mandatory: true},

	// --- 4 months ---
	{Code: "DTP_HB_HIB", Name: "DTP-HB-Hib (Pentavalent)", DoseNumber: 3, DoseLabel: "Dose 3", AgeMonths: 4, AgeLabel: "4 months", Mandatory: true},
	{Code: "POLIO", Name: "Polio (OPV)", DoseNumber: 4, DoseLabel: "OPV-4", AgeMonths: 4, AgeLabel: "4 months", Mandatory: true},
	{Code: "IPV", Name: "Polio (IPV, injectable)", DoseNumber: 1, DoseLabel: "IPV-1", AgeMonths: 4, AgeLabel: "4 months", Mandatory: true},
	{Code: "RV", Name: "Rotavirus", DoseNumber: 3, DoseLabel: "Dose 3", AgeMonths: 4, AgeLabel: "4 months", Mandatory: true},

	// --- 6 months ---
	{Code: "FLU", Name: "Influenza", DoseNumber: 1, DoseLabel: "First dose", AgeMonths: 6, AgeLabel: "6 months (then yearly)", Mandatory: false},

	// --- 9 months ---
	{Code: "MR", Name: "Measles-Rubella (MR)", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 9, AgeLabel: "9 months", Mandatory: true},
	{Code: "IPV", Name: "Polio (IPV, injectable)", DoseNumber: 2, DoseLabel: "IPV-2", AgeMonths: 9, AgeLabel: "9 months", Mandatory: true},

	// --- 10 months ---
	{Code: "JE", Name: "Japanese Encephalitis", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 10, AgeLabel: "10 months (endemic areas only)", Mandatory: false},

	// --- 12 months ---
	{Code: "PCV", Name: "PCV (Pneumococcal)", DoseNumber: 3, DoseLabel: "Booster", AgeMonths: 12, AgeLabel: "12 months", Mandatory: true},
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
	{Code: "DTP", Name: "DTaP (booster)", DoseNumber: 1, DoseLabel: "Booster", AgeMonths: 60, AgeLabel: "5–7 years", Mandatory: false},
	{Code: "DENG", Name: "Dengue", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 72, AgeLabel: "from 6 years", Mandatory: false},
	{Code: "DENG", Name: "Dengue", DoseNumber: 2, DoseLabel: "Dose 2", AgeMonths: 75, AgeLabel: "3 months after dose 1", Mandatory: false},
	{Code: "MR", Name: "Measles-Rubella (MR)", DoseNumber: 3, DoseLabel: "Dose 3", AgeMonths: 84, AgeLabel: "~7 years (school grade 1)", Mandatory: true},
	{Code: "HPV", Name: "HPV", DoseNumber: 1, DoseLabel: "Single dose", AgeMonths: 132, AgeLabel: "~11 years (school grade 5, girls)", Mandatory: true},
}

// Schedule returns a copy of the reference immunization schedule.
func Schedule() []ScheduleEntry {
	out := make([]ScheduleEntry, len(schedule))
	copy(out, schedule)
	return out
}

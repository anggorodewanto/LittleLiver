package report

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"math"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/who"
)

const feverThreshold = 38.0

// reportData holds all queried data for the report.
type reportData struct {
	summary     *store.DashboardSummary
	stools      []store.StoolColorSeriesEntry
	temps       []store.TemperatureSeriesEntry
	feedings    []store.FeedingDailyEntry
	medLogs     []medAdherence
	notes       []noteEntry
	weights     []store.WeightSeriesEntry
	labTrends   map[string][]store.LabTrendEntry
	whoCurves   []who.PercentileCurve
	photoThumbs [][]byte // thumbnail image bytes
}

// medAdherence holds medication adherence info.
type medAdherence struct {
	Name    string
	Dose    string
	Total   int
	Given   int
	Skipped int
}

// noteEntry holds a general note for the report.
type noteEntry struct {
	Timestamp string
	Content   string
	Category  string
}

// Generate produces a PDF report for the given baby within the date range [from, to].
// The PDF is written to w. If objStore is non-nil, photo thumbnails are fetched and embedded.
func Generate(db *sql.DB, objStore storage.ObjectStore, baby *model.Baby, from, to string, w io.Writer) error {
	now := time.Now().UTC()

	data, err := queryReportData(db, objStore, baby, from, to, now)
	if err != nil {
		return fmt.Errorf("query report data: %w", err)
	}

	m := buildPDF(baby, from, to, data, now)

	doc, err := m.Generate()
	if err != nil {
		return fmt.Errorf("generate PDF: %w", err)
	}

	_, err = w.Write(doc.GetBytes())
	return err
}

// queryReportData fetches all data needed for the report.
func queryReportData(db *sql.DB, objStore storage.ObjectStore, baby *model.Baby, from, to string, now time.Time) (*reportData, error) {
	babyID := baby.ID

	summary, err := store.GetDashboardSummary(db, babyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("dashboard summary: %w", err)
	}

	stools, err := store.GetStoolColorSeries(db, babyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("stool color series: %w", err)
	}

	temps, err := store.GetTemperatureSeries(db, babyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("temperature series: %w", err)
	}

	feedings, err := store.GetFeedingDaily(db, babyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("feeding daily: %w", err)
	}

	medLogs, err := queryMedAdherence(db, babyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("med adherence: %w", err)
	}

	notes, err := queryNotes(db, babyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("notes: %w", err)
	}

	weights, err := store.GetWeightSeries(db, babyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("weight series: %w", err)
	}

	labTrends, err := store.GetLabTrends(db, babyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("lab trends: %w", err)
	}

	// Compute WHO percentile curves for the baby's age range
	var whoCurves []who.PercentileCurve
	ageDays := int(now.Sub(baby.DateOfBirth).Hours() / 24)
	if ageDays > 0 && ageDays <= 730 {
		curves, err := who.PercentileCurves(baby.Sex, 0, ageDays)
		if err == nil {
			whoCurves = curves
		}
	}

	// Fetch photo thumbnails from storage
	photoThumbs := fetchPhotoThumbnails(db, objStore, babyID, from, to)

	return &reportData{
		summary:     summary,
		stools:      stools,
		temps:       temps,
		feedings:    feedings,
		medLogs:     medLogs,
		notes:       notes,
		weights:     weights,
		labTrends:   labTrends,
		whoCurves:   whoCurves,
		photoThumbs: photoThumbs,
	}, nil
}

// fetchPhotoThumbnails queries linked photo_uploads for the baby in the date range
// and fetches thumbnail bytes from object storage.
func fetchPhotoThumbnails(db *sql.DB, objStore storage.ObjectStore, babyID, from, to string) [][]byte {
	if objStore == nil {
		return nil
	}

	fromTime, toTime, err := store.ParseDateRange(from, to)
	if err != nil {
		return nil
	}

	// Query for thumbnail keys from stools with photos in this range
	rows, err := db.Query(
		`SELECT DISTINCT p.thumbnail_key
		 FROM photo_uploads p
		 WHERE p.baby_id = ? AND p.linked_at IS NOT NULL
		 AND p.uploaded_at >= ? AND p.uploaded_at < ?
		 AND p.thumbnail_key IS NOT NULL AND p.thumbnail_key != ''
		 ORDER BY p.uploaded_at ASC
		 LIMIT 20`,
		babyID, fromTime, toTime,
	)
	if err != nil {
		log.Printf("query photo thumbnails: %v", err)
		return nil
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			continue
		}
		keys = append(keys, key)
	}

	if len(keys) == 0 {
		return nil
	}

	ctx := context.Background()
	var thumbs [][]byte
	for _, key := range keys {
		data, err := objStore.Get(ctx, key)
		if err != nil {
			log.Printf("fetch thumbnail %s: %v", key, err)
			continue
		}
		thumbs = append(thumbs, data)
	}

	return thumbs
}

// queryMedAdherence computes medication adherence for the date range.
func queryMedAdherence(db *sql.DB, babyID, from, to string) ([]medAdherence, error) {
	fromTime, toTime, err := store.ParseDateRange(from, to)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT m.name, m.dose,
			COUNT(*) as total,
			SUM(CASE WHEN ml.skipped = 0 THEN 1 ELSE 0 END) as given,
			SUM(CASE WHEN ml.skipped = 1 THEN 1 ELSE 0 END) as skipped
		 FROM med_logs ml
		 JOIN medications m ON ml.medication_id = m.id
		 WHERE ml.baby_id = ? AND ml.created_at >= ? AND ml.created_at < ?
		 GROUP BY m.id, m.name, m.dose
		 ORDER BY m.name`,
		babyID, fromTime, toTime,
	)
	if err != nil {
		return nil, fmt.Errorf("query med adherence: %w", err)
	}
	defer rows.Close()

	var result []medAdherence
	for rows.Next() {
		var ma medAdherence
		if err := rows.Scan(&ma.Name, &ma.Dose, &ma.Total, &ma.Given, &ma.Skipped); err != nil {
			return nil, fmt.Errorf("scan med adherence: %w", err)
		}
		result = append(result, ma)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = make([]medAdherence, 0)
	}
	return result, nil
}

// queryNotes fetches general notes for the date range.
func queryNotes(db *sql.DB, babyID, from, to string) ([]noteEntry, error) {
	fromTime, toTime, err := store.ParseDateRange(from, to)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT timestamp, content, COALESCE(category, '')
		 FROM general_notes
		 WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?
		 ORDER BY timestamp ASC`,
		babyID, fromTime, toTime,
	)
	if err != nil {
		return nil, fmt.Errorf("query notes: %w", err)
	}
	defer rows.Close()

	var result []noteEntry
	for rows.Next() {
		var n noteEntry
		if err := rows.Scan(&n.Timestamp, &n.Content, &n.Category); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		result = append(result, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = make([]noteEntry, 0)
	}
	return result, nil
}

// buildPDF constructs the maroto PDF document.
func buildPDF(baby *model.Baby, from, to string, data *reportData, now time.Time) core.Maroto {
	cfg := config.NewBuilder().
		WithDefaultFont(&props.Font{
			Size:   10,
			Style:  fontstyle.Normal,
			Family: "helvetica",
		}).
		Build()

	m := maroto.New(cfg)

	addHeader(m, baby, from, to, now)
	addSummarySection(m, data.summary)
	addStoolColorTable(m, data.stools)
	addTemperatureTable(m, data.temps)
	addFeedingSummary(m, data.feedings)
	addMedicationAdherence(m, data.medLogs)
	addNotableObservations(m, data.notes)

	// Charts
	addChartSection(m, "Stool Color Distribution", data.stools)
	addWeightChartSection(m, data.weights, data.whoCurves, baby.DateOfBirth.Format(model.DateFormat))
	addLabTrendsSection(m, data.labTrends)

	// Photo appendix
	addPhotoAppendix(m, data.photoThumbs)

	return m
}

// embedChartPNG embeds a pre-rendered chart PNG into the PDF with a section title.
func embedChartPNG(m core.Maroto, title string, chartPNG []byte) {
	if chartPNG == nil {
		return
	}
	m.AddRows(text.NewRow(8, title, sectionStyle))
	m.AddRows(row.New(80).Add(
		image.NewFromBytesCol(12, chartPNG, extension.Png, props.Rect{
			Percent: 100,
			Center:  true,
		}),
	))
	m.AddRows(spacerRow(4))
}

// addChartSection renders a stool chart and embeds it.
func addChartSection(m core.Maroto, title string, stools []store.StoolColorSeriesEntry) {
	chartPNG, err := renderStoolChart(stools)
	if err != nil {
		return
	}
	embedChartPNG(m, title, chartPNG)
}

// addWeightChartSection renders a weight chart with WHO percentile bands.
func addWeightChartSection(m core.Maroto, weights []store.WeightSeriesEntry, curves []who.PercentileCurve, dob string) {
	chartPNG, err := renderWeightChart(weights, curves, dob)
	if err != nil {
		return
	}
	embedChartPNG(m, "Weight Chart (WHO Percentiles)", chartPNG)
}

// addLabTrendsSection renders a lab trends chart.
func addLabTrendsSection(m core.Maroto, trends map[string][]store.LabTrendEntry) {
	chartPNG, err := renderLabTrendsChart(trends)
	if err != nil {
		return
	}
	embedChartPNG(m, "Lab Results Trends", chartPNG)
}

// addPhotoAppendix adds a photo appendix section with thumbnails.
func addPhotoAppendix(m core.Maroto, thumbs [][]byte) {
	if len(thumbs) == 0 {
		return
	}

	m.AddRows(text.NewRow(8, "Photo Appendix", sectionStyle))

	// Layout: 3 photos per row
	for i := 0; i < len(thumbs); i += 3 {
		var cols []core.Col
		for j := 0; j < 3 && i+j < len(thumbs); j++ {
			ext := detectImageExtension(thumbs[i+j])
			cols = append(cols, image.NewFromBytesCol(4, thumbs[i+j], ext, props.Rect{
				Percent: 90,
				Center:  true,
			}))
		}
		// Fill remaining columns
		for len(cols) < 3 {
			cols = append(cols, col.New(4))
		}
		m.AddRows(row.New(50).Add(cols...))
	}

	m.AddRows(spacerRow(4))
}

// detectImageExtension detects the image format from bytes.
func detectImageExtension(data []byte) extension.Type {
	if len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n" {
		return extension.Png
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8 {
		return extension.Jpg
	}
	return extension.Png // default
}

// styling helpers
var (
	titleStyle = props.Text{
		Size:  16,
		Style: fontstyle.Bold,
		Align: align.Center,
		Top:   2,
	}
	sectionStyle = props.Text{
		Size:  13,
		Style: fontstyle.Bold,
		Top:   4,
	}
	labelStyle = props.Text{
		Size:  10,
		Style: fontstyle.Bold,
	}
	valueStyle = props.Text{
		Size: 10,
	}
	tableHeaderStyle = props.Text{
		Size:  9,
		Style: fontstyle.Bold,
		Align: align.Center,
	}
	tableCellStyle = props.Text{
		Size:  9,
		Align: align.Center,
	}
	headerBg = &props.Cell{
		BackgroundColor: &props.Color{Red: 220, Green: 220, Blue: 220},
		BorderType:      border.Full,
		BorderThickness: 0.3,
	}
	cellBorder = &props.Cell{
		BorderType:      border.Full,
		BorderThickness: 0.2,
	}
)

// addHeader adds the report header with baby info.
func addHeader(m core.Maroto, baby *model.Baby, from, to string, now time.Time) {
	m.AddRows(text.NewRow(10, "LittleLiver Health Report", titleStyle))
	m.AddRows(spacerRow(3))

	ageDays := int(now.Sub(baby.DateOfBirth).Hours() / 24)
	ageStr := FormatAge(ageDays)

	m.AddRow(7,
		text.NewCol(4, fmt.Sprintf("Name: %s", baby.Name), labelStyle),
		text.NewCol(4, fmt.Sprintf("Sex: %s", baby.Sex), valueStyle),
		text.NewCol(4, fmt.Sprintf("DOB: %s", baby.DateOfBirth.Format(model.DateFormat)), valueStyle),
	)

	m.AddRow(7,
		text.NewCol(4, fmt.Sprintf("Age: %s", ageStr), valueStyle),
		text.NewCol(4, kasaiInfo(baby, now), valueStyle),
		text.NewCol(4, fmt.Sprintf("Period: %s to %s", from, to), valueStyle),
	)

	m.AddRows(spacerRow(4))
}

// kasaiInfo returns Kasai date info string.
func kasaiInfo(baby *model.Baby, now time.Time) string {
	if baby.KasaiDate == nil {
		return "Kasai: N/A"
	}
	days := int(now.Sub(*baby.KasaiDate).Hours() / 24)
	return fmt.Sprintf("Days Post-Kasai: %d", days)
}

// FormatAge formats age in days to a human-readable string.
func FormatAge(days int) string {
	if days < 30 {
		return fmt.Sprintf("%d %s", days, plural(days, "day"))
	}
	months := days / 30
	remainDays := days % 30
	if months < 12 {
		return fmt.Sprintf("%d %s, %d %s", months, plural(months, "month"), remainDays, plural(remainDays, "day"))
	}
	years := months / 12
	remainMonths := months % 12
	return fmt.Sprintf("%d %s, %d %s", years, plural(years, "year"), remainMonths, plural(remainMonths, "month"))
}

// plural returns the singular or plural form of a word.
func plural(n int, singular string) string {
	if n == 1 {
		return singular
	}
	return singular + "s"
}

// addSummarySection adds the summary section.
func addSummarySection(m core.Maroto, summary *store.DashboardSummary) {
	m.AddRows(text.NewRow(8, "Summary", sectionStyle))

	m.AddRow(7,
		text.NewCol(3, fmt.Sprintf("Total Feeds: %d", summary.TotalFeeds), valueStyle),
		text.NewCol(3, fmt.Sprintf("Total Calories: %.0f kcal", summary.TotalCalories), valueStyle),
		text.NewCol(3, fmt.Sprintf("Wet Diapers: %d", summary.TotalWetDiapers), valueStyle),
		text.NewCol(3, fmt.Sprintf("Stools: %d", summary.TotalStools), valueStyle),
	)

	colorStr := "N/A"
	if summary.WorstStoolColor != nil {
		colorStr = fmt.Sprintf("%d", *summary.WorstStoolColor)
	}
	tempStr := "N/A"
	if summary.LastTemperature != nil {
		tempStr = fmt.Sprintf("%.1f°C", *summary.LastTemperature)
	}
	weightStr := "N/A"
	if summary.LastWeight != nil {
		weightStr = fmt.Sprintf("%.2f kg", *summary.LastWeight)
	}

	m.AddRow(7,
		text.NewCol(4, fmt.Sprintf("Stool Color: %s", colorStr), valueStyle),
		text.NewCol(4, fmt.Sprintf("Last Temp: %s", tempStr), valueStyle),
		text.NewCol(4, fmt.Sprintf("Last Weight: %s", weightStr), valueStyle),
	)

	m.AddRows(spacerRow(3))
}

// addStoolColorTable adds the stool color log table.
func addStoolColorTable(m core.Maroto, stools []store.StoolColorSeriesEntry) {
	m.AddRows(text.NewRow(8, "Stool Color Log", sectionStyle))

	if len(stools) == 0 {
		m.AddRows(text.NewRow(6, "No stool entries in this period.", valueStyle))
		m.AddRows(spacerRow(3))
		return
	}

	// Table header
	m.AddRows(row.New(7).Add(
		text.NewCol(4, "Timestamp", tableHeaderStyle),
		text.NewCol(4, "Color Score", tableHeaderStyle),
	).WithStyle(headerBg))

	for _, s := range stools {
		ts := formatTimestamp(s.Timestamp)
		m.AddRows(row.New(6).Add(
			text.NewCol(4, ts, tableCellStyle),
			text.NewCol(4, fmt.Sprintf("%d", s.ColorScore), tableCellStyle),
		).WithStyle(cellBorder))
	}

	m.AddRows(spacerRow(3))
}

// addTemperatureTable adds the temperature log table with fever flags.
func addTemperatureTable(m core.Maroto, temps []store.TemperatureSeriesEntry) {
	m.AddRows(text.NewRow(8, "Temperature Log", sectionStyle))

	if len(temps) == 0 {
		m.AddRows(text.NewRow(6, "No temperature entries in this period.", valueStyle))
		m.AddRows(spacerRow(3))
		return
	}

	// Table header
	m.AddRows(row.New(7).Add(
		text.NewCol(3, "Timestamp", tableHeaderStyle),
		text.NewCol(3, "Value (°C)", tableHeaderStyle),
		text.NewCol(3, "Method", tableHeaderStyle),
		text.NewCol(3, "Fever", tableHeaderStyle),
	).WithStyle(headerBg))

	for _, t := range temps {
		ts := formatTimestamp(t.Timestamp)
		feverFlag := ""
		feverStyle := tableCellStyle
		if t.Value >= feverThreshold {
			feverFlag = "YES"
			feverStyle = props.Text{
				Size:  9,
				Align: align.Center,
				Style: fontstyle.Bold,
				Color: &props.Color{Red: 255, Green: 0, Blue: 0},
			}
		}
		m.AddRows(row.New(6).Add(
			text.NewCol(3, ts, tableCellStyle),
			text.NewCol(3, fmt.Sprintf("%.1f", t.Value), tableCellStyle),
			text.NewCol(3, t.Method, tableCellStyle),
			text.NewCol(3, feverFlag, feverStyle),
		).WithStyle(cellBorder))
	}

	m.AddRows(spacerRow(3))
}

// addFeedingSummary adds the feeding summary section.
func addFeedingSummary(m core.Maroto, feedings []store.FeedingDailyEntry) {
	m.AddRows(text.NewRow(8, "Feeding Summary", sectionStyle))

	if len(feedings) == 0 {
		m.AddRows(text.NewRow(6, "No feeding entries in this period.", valueStyle))
		m.AddRows(spacerRow(3))
		return
	}

	// Calculate averages
	totalVol := 0.0
	totalCal := 0.0
	for _, f := range feedings {
		totalVol += f.TotalVolumeMl
		totalCal += f.TotalCalories
	}
	days := float64(len(feedings))
	avgVol := totalVol / days
	avgCal := totalCal / days

	m.AddRow(7,
		text.NewCol(6, fmt.Sprintf("Avg Daily Volume: %.0f mL", math.Round(avgVol)), valueStyle),
		text.NewCol(6, fmt.Sprintf("Avg Daily Calories: %.0f kcal", math.Round(avgCal)), valueStyle),
	)

	// Daily table
	m.AddRows(row.New(7).Add(
		text.NewCol(3, "Date", tableHeaderStyle),
		text.NewCol(3, "Volume (mL)", tableHeaderStyle),
		text.NewCol(3, "Calories", tableHeaderStyle),
		text.NewCol(3, "Feed Count", tableHeaderStyle),
	).WithStyle(headerBg))

	for _, f := range feedings {
		m.AddRows(row.New(6).Add(
			text.NewCol(3, f.Date, tableCellStyle),
			text.NewCol(3, fmt.Sprintf("%.0f", f.TotalVolumeMl), tableCellStyle),
			text.NewCol(3, fmt.Sprintf("%.0f", f.TotalCalories), tableCellStyle),
			text.NewCol(3, fmt.Sprintf("%d", f.FeedCount), tableCellStyle),
		).WithStyle(cellBorder))
	}

	m.AddRows(spacerRow(3))
}

// addMedicationAdherence adds the medication adherence section.
func addMedicationAdherence(m core.Maroto, meds []medAdherence) {
	m.AddRows(text.NewRow(8, "Medication Adherence", sectionStyle))

	if len(meds) == 0 {
		m.AddRows(text.NewRow(6, "No medication logs in this period.", valueStyle))
		m.AddRows(spacerRow(3))
		return
	}

	// Table header
	m.AddRows(row.New(7).Add(
		text.NewCol(3, "Medication", tableHeaderStyle),
		text.NewCol(2, "Dose", tableHeaderStyle),
		text.NewCol(2, "Given", tableHeaderStyle),
		text.NewCol(2, "Skipped", tableHeaderStyle),
		text.NewCol(3, "Adherence", tableHeaderStyle),
	).WithStyle(headerBg))

	for _, med := range meds {
		ratio := 0.0
		if med.Total > 0 {
			ratio = float64(med.Given) / float64(med.Total) * 100
		}
		m.AddRows(row.New(6).Add(
			text.NewCol(3, med.Name, tableCellStyle),
			text.NewCol(2, med.Dose, tableCellStyle),
			text.NewCol(2, fmt.Sprintf("%d", med.Given), tableCellStyle),
			text.NewCol(2, fmt.Sprintf("%d", med.Skipped), tableCellStyle),
			text.NewCol(3, fmt.Sprintf("%.0f%%", ratio), tableCellStyle),
		).WithStyle(cellBorder))
	}

	m.AddRows(spacerRow(3))
}

// addNotableObservations adds the notable observations section.
func addNotableObservations(m core.Maroto, notes []noteEntry) {
	m.AddRows(text.NewRow(8, "Notable Observations", sectionStyle))

	if len(notes) == 0 {
		m.AddRows(text.NewRow(6, "No observations in this period.", valueStyle))
		return
	}

	for _, n := range notes {
		ts := formatTimestamp(n.Timestamp)
		catStr := ""
		if n.Category != "" {
			catStr = fmt.Sprintf(" [%s]", n.Category)
		}
		m.AddRow(7,
			text.NewCol(3, ts+catStr, labelStyle),
			text.NewCol(9, n.Content, valueStyle),
		)
	}
}

// spacerRow creates an empty spacer row.
func spacerRow(height float64) core.Row {
	return row.New(height).Add(col.New())
}

// formatTimestamp formats a UTC datetime string for display.
func formatTimestamp(ts string) string {
	t, err := time.Parse(model.DateTimeFormat, ts)
	if err != nil {
		return ts
	}
	return t.Format("2006-01-02 15:04")
}

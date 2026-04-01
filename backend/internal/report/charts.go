package report

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/who"
)

const (
	chartWidth  = 6 * vg.Inch
	chartHeight = 3 * vg.Inch
)

// percentileColors for the WHO curves.
var percentileColors = []color.RGBA{
	{R: 200, G: 200, B: 200, A: 255}, // 3rd
	{R: 150, G: 200, B: 150, A: 255}, // 15th
	{R: 50, G: 150, B: 50, A: 255},   // 50th
	{R: 150, G: 200, B: 150, A: 255}, // 85th
	{R: 200, G: 200, B: 200, A: 255}, // 97th
}

// labCategoryOrder defines the rendering order for lab chart categories.
var labCategoryOrder = []string{"Liver Function", "Hematology", "Electrolytes", "Other"}

// labCategories maps category names to the test names that belong in each.
// Matching is case-insensitive.
var labCategories = map[string][]string{
	"Liver Function": {"SGOT/AST", "SGPT/ALT", "Gamma GT", "Bilirubin Total", "Bilirubin Direk", "albumin"},
	"Hematology":     {"Hemoglobin", "Hematokrit", "Eritrosit", "Leukosit", "MCH", "MCHC", "MCV", "RDW-CV", "RDW-SD", "NRBC#", "NRBC%"},
	"Electrolytes":   {"Natrium", "Kalium", "Kalsium", "Klorida", "Magnesium"},
}

// categorizeLabTrends groups lab trend entries by clinical category.
// Tests that don't match any category go into "Other".
func categorizeLabTrends(trends map[string][]store.LabTrendEntry) map[string]map[string][]store.LabTrendEntry {
	if len(trends) == 0 {
		return nil
	}

	// Build a lowercase lookup: lowercase test name -> category
	lookup := make(map[string]string)
	for cat, tests := range labCategories {
		for _, t := range tests {
			lookup[strings.ToLower(t)] = cat
		}
	}

	result := make(map[string]map[string][]store.LabTrendEntry)
	for testName, entries := range trends {
		cat, ok := lookup[strings.ToLower(testName)]
		if !ok {
			cat = "Other"
		}
		if result[cat] == nil {
			result[cat] = make(map[string][]store.LabTrendEntry)
		}
		result[cat][testName] = entries
	}
	return result
}

// renderLabTrendCharts renders separate lab trend charts per clinical category.
// Returns map[categoryName]pngBytes. Categories with no numeric data are omitted.
func renderLabTrendCharts(trends map[string][]store.LabTrendEntry) (map[string][]byte, error) {
	categorized := categorizeLabTrends(trends)
	if len(categorized) == 0 {
		return nil, nil
	}

	result := make(map[string][]byte)
	for _, cat := range labCategoryOrder {
		catTrends, ok := categorized[cat]
		if !ok {
			continue
		}
		png, err := renderSingleLabChart(cat, catTrends)
		if err != nil {
			return nil, fmt.Errorf("render %s chart: %w", cat, err)
		}
		if png != nil {
			result[cat] = png
		}
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// renderSingleLabChart renders a single lab trends chart for one category.
func renderSingleLabChart(category string, trends map[string][]store.LabTrendEntry) ([]byte, error) {
	if len(trends) == 0 {
		return nil, nil
	}

	p := plot.New()
	p.Title.Text = "Lab Trends: " + category
	p.X.Label.Text = "Date"
	p.Y.Label.Text = "Value"

	names := make([]string, 0, len(trends))
	for name := range trends {
		names = append(names, name)
	}
	sort.Strings(names)

	hasData := false
	colorIdx := 0

	for _, name := range names {
		entries := trends[name]
		pts := make(plotter.XYs, 0, len(entries))

		for _, e := range entries {
			val, err := strconv.ParseFloat(e.Value, 64)
			if err != nil {
				continue
			}
			ts, err := store.ParseTime(e.Timestamp)
			if err != nil {
				continue
			}
			pts = append(pts, plotter.XY{
				X: float64(ts.Unix()),
				Y: val,
			})
		}

		if len(pts) == 0 {
			continue
		}

		hasData = true

		line, err := plotter.NewLine(pts)
		if err != nil {
			return nil, fmt.Errorf("create lab trend line: %w", err)
		}
		ci := colorIdx % len(labLineColors)
		line.Color = labLineColors[ci]
		line.Width = vg.Points(2)
		colorIdx++

		p.Add(line)

		unit := ""
		if len(entries) > 0 && entries[0].Unit != nil {
			unit = " (" + *entries[0].Unit + ")"
		}
		p.Legend.Add(name+unit, line)
	}

	if !hasData {
		return nil, nil
	}

	p.X.Tick.Marker = dateTicker{}
	p.Legend.Top = true
	p.Legend.Left = true

	return renderPlotToPNG(p)
}

// labLineColors for different lab test names.
var labLineColors = []color.RGBA{
	{R: 31, G: 119, B: 180, A: 255},
	{R: 255, G: 127, B: 14, A: 255},
	{R: 44, G: 160, B: 44, A: 255},
	{R: 214, G: 39, B: 40, A: 255},
	{R: 148, G: 103, B: 189, A: 255},
	{R: 140, G: 86, B: 75, A: 255},
}

// renderStoolChart renders a stool color distribution chart as PNG bytes.
// Returns nil if there are no stool entries.
func renderStoolChart(stools []store.StoolColorSeriesEntry) ([]byte, error) {
	if len(stools) == 0 {
		return nil, nil
	}

	// Count color score distribution
	counts := make(map[int]int)
	for _, s := range stools {
		counts[s.ColorScore]++
	}

	// Create sorted list of scores
	scores := make([]int, 0, len(counts))
	for s := range counts {
		scores = append(scores, s)
	}
	sort.Ints(scores)

	p := plot.New()
	p.Title.Text = "Stool Color Distribution"
	p.X.Label.Text = "Color Score"
	p.Y.Label.Text = "Count"

	values := make(plotter.Values, len(scores))
	labels := make([]string, len(scores))
	for i, s := range scores {
		values[i] = float64(counts[s])
		labels[i] = fmt.Sprintf("%d", s)
	}

	bars, err := plotter.NewBarChart(values, 20*vg.Millimeter)
	if err != nil {
		return nil, fmt.Errorf("create bar chart: %w", err)
	}

	// Color bars by score
	bars.Color = color.RGBA{R: 100, G: 150, B: 200, A: 255}

	p.Add(bars)
	p.NominalX(labels...)

	return renderPlotToPNG(p)
}

// renderWeightChart renders a weight chart with WHO percentile bands as PNG bytes.
// Returns nil if there are no weights and no WHO curves.
func renderWeightChart(weights []store.WeightSeriesEntry, curves []who.PercentileCurve, dob string) ([]byte, error) {
	if len(weights) == 0 && len(curves) == 0 {
		return nil, nil
	}

	dobTime, err := time.Parse(model.DateFormat, dob)
	if err != nil {
		return nil, fmt.Errorf("parse dob: %w", err)
	}

	p := plot.New()
	p.Title.Text = "Weight vs Age (with WHO Percentiles)"
	p.X.Label.Text = "Age (days)"
	p.Y.Label.Text = "Weight (kg)"

	// Add WHO percentile curves
	for i, curve := range curves {
		pts := make(plotter.XYs, len(curve.Points))
		for j, cp := range curve.Points {
			pts[j].X = float64(cp.AgeDays)
			pts[j].Y = cp.WeightKg
		}

		line, err := plotter.NewLine(pts)
		if err != nil {
			return nil, fmt.Errorf("create percentile line: %w", err)
		}
		ci := i % len(percentileColors)
		line.Color = percentileColors[ci]
		line.Width = vg.Points(1)
		if curve.Percentile == 50 {
			line.Width = vg.Points(2)
		}
		p.Add(line)
		p.Legend.Add(fmt.Sprintf("P%.0f", curve.Percentile), line)
	}

	// Add actual weight data points
	if len(weights) > 0 {
		pts := make(plotter.XYs, 0, len(weights))
		for _, w := range weights {
			ts, parseErr := store.ParseTime(w.Timestamp)
			if parseErr != nil {
				continue
			}
			ageDays := ts.Sub(dobTime).Hours() / 24
			pts = append(pts, plotter.XY{X: ageDays, Y: w.WeightKg})
		}

		if len(pts) > 0 {
			scatter, err := plotter.NewScatter(pts)
			if err != nil {
				return nil, fmt.Errorf("create weight scatter: %w", err)
			}
			scatter.GlyphStyle.Color = color.RGBA{R: 0, G: 0, B: 200, A: 255}
			scatter.GlyphStyle.Radius = vg.Points(3)
			scatter.GlyphStyle.Shape = draw.CircleGlyph{}
			p.Add(scatter)
			p.Legend.Add("Measured", scatter)
		}
	}

	p.Legend.Top = true
	p.Legend.Left = true

	return renderPlotToPNG(p)
}

// renderLabTrendsChart renders lab trends as a single combined chart.
// Kept for backward compatibility with existing tests; delegates to renderSingleLabChart.
func renderLabTrendsChart(trends map[string][]store.LabTrendEntry) ([]byte, error) {
	return renderSingleLabChart("Lab Results Trends", trends)
}

// dateTicker formats unix timestamps as date strings on the X axis.
type dateTicker struct{}

func (dateTicker) Ticks(min, max float64) []plot.Tick {
	n := 5
	step := (max - min) / float64(n)
	if step <= 0 {
		return []plot.Tick{{Value: min, Label: formatUnixDate(min)}}
	}

	ticks := make([]plot.Tick, 0, n+1)
	for i := 0; i <= n; i++ {
		v := min + float64(i)*step
		ticks = append(ticks, plot.Tick{
			Value: v,
			Label: formatUnixDate(v),
		})
	}
	return ticks
}

func formatUnixDate(unix float64) string {
	t := time.Unix(int64(math.Round(unix)), 0).UTC()
	return t.Format("Jan 02")
}

// renderPlotToPNG renders a plot to PNG bytes.
func renderPlotToPNG(p *plot.Plot) ([]byte, error) {
	img := vgimg.New(chartWidth, chartHeight)
	dc := draw.New(img)
	p.Draw(dc)

	var buf bytes.Buffer
	png := vgimg.PngCanvas{Canvas: img}
	if _, err := png.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("write PNG: %w", err)
	}
	return buf.Bytes(), nil
}

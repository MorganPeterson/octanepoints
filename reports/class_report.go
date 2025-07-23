package reports

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"
	"time"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/database"
)

var classReportTmpl = template.Must(
	template.New("class_report.tmpl").
		Funcs(template.FuncMap{
			"fmtDur": fmtDur,
			"pad":    Pad,
			"padNum": PadNum,
		}).
		ParseFS(tmplFS, "templates/class_report.tmpl"),
)

type ClassPointsRow struct {
	RallyID  uint64
	ClassID  uint64
	UserID   uint64
	UserName string
	Time3    time.Duration
	Pos      int64
	Points   int64
}

type ClassTable struct {
	ClassName string
	Rows      []ClassPointsRow
}

type RallySection struct {
	RallyID uint64
	Classes []ClassTable
}

type ChampDriverRow struct {
	UserID      uint64
	UserName    string
	TotalPoints int64
	Pos         int64
}

type ChampSection struct {
	ClassName string
	Rows      []ChampDriverRow
}

type ClassReportData struct {
	Rally        RallySection
	Championship []ChampSection
}

// ExportClassReport generates class tables for a single rally (rallyIDStr)
// AND championship totals across all rallies, then writes class_report.md.
func ExportClassReport(rallyIDStr string, store *database.Store, cfg *configuration.Config) error {
	rallyID := database.ParseStringToUint(rallyIDStr)

	// 1) Class lookup
	classLookup, err := database.GetClasses(store)
	if err != nil {
		return fmt.Errorf("load classes: %w", err)
	}

	// 2) Ranked rows for THIS rally
	rallyRanked, err := database.FetchRankedRows(store, &database.QueryOpts{
		RallyId: rallyID,
	})
	if err != nil {
		return fmt.Errorf("fetch rally ranks: %w", err)
	}
	rallyWithPts := applyPoints(rallyRanked, cfg.General.ClassPoints)

	// Group into tables
	rallySection := RallySection{
		RallyID: rallyID,
		Classes: groupTables(rallyWithPts, classLookup),
	}

	// 3) Ranked rows for ALL rallies (for championship totals)
	allRanked, err := database.FetchRankedRows(store, nil)
	if err != nil {
		return fmt.Errorf("fetch all ranks: %w", err)
	}
	allWithPts := applyPoints(allRanked, cfg.General.ClassPoints)
	champ := buildChampionship(allWithPts, classLookup)

	// 4) Render markdown
	data := ClassReportData{
		Rally:        rallySection,
		Championship: champ,
	}
	return renderMarkdown(data)
}

// ----- Points & grouping logic -----

func applyPoints(ranked []database.RankedRow, scheme []int64) []ClassPointsRow {
	out := make([]ClassPointsRow, len(ranked))
	for i, r := range ranked {
		pts := int64(0)
		if r.Pos-1 < int64(len(scheme)) {
			pts = scheme[r.Pos-1]
		}
		out[i] = ClassPointsRow{
			RallyID:  r.RallyId,
			ClassID:  r.ClassId,
			UserID:   r.UserId,
			UserName: r.UserName,
			Time3:    time.Duration(r.Time3),
			Pos:      r.Pos,
			Points:   pts,
		}
	}
	return out
}

func groupTables(rows []ClassPointsRow, classLookup map[uint64]database.Class) []ClassTable {
	byClass := map[uint64][]ClassPointsRow{}
	for _, r := range rows {
		byClass[r.ClassID] = append(byClass[r.ClassID], r)
	}
	out := make([]ClassTable, 0, len(byClass))
	for cid, slice := range byClass {
		// Already sorted by query, but ensure by Pos
		sort.Slice(slice, func(i, j int) bool { return slice[i].Pos < slice[j].Pos })
		out = append(out, ClassTable{
			ClassName: classLookup[cid].Name,
			Rows:      slice,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ClassName < out[j].ClassName })
	return out
}

func buildChampionship(rows []ClassPointsRow, classLookup map[uint64]database.Class) []ChampSection {
	type key struct {
		ClassID uint64
		UserID  uint64
	}
	acc := map[key]*ChampDriverRow{}
	for _, r := range rows {
		k := key{r.ClassID, r.UserID}
		if _, ok := acc[k]; !ok {
			acc[k] = &ChampDriverRow{
				UserID:   r.UserID,
				UserName: r.UserName,
			}
		}
		acc[k].TotalPoints += r.Points
	}

	// regroup by class
	byClass := map[uint64][]ChampDriverRow{}
	for k, v := range acc {
		byClass[k.ClassID] = append(byClass[k.ClassID], *v)
	}

	out := make([]ChampSection, 0, len(byClass))
	for cid, slice := range byClass {
		sort.Slice(slice, func(i, j int) bool { return slice[i].TotalPoints > slice[j].TotalPoints })
		// set positions
		for i := range slice {
			slice[i].Pos = int64(i + 1)
		}
		out = append(out, ChampSection{
			ClassName: classLookup[cid].Name,
			Rows:      slice,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ClassName < out[j].ClassName })
	return out
}

// ----- Templating -----

func renderMarkdown(data any) error {
	var buf bytes.Buffer
	if err := classReportTmpl.Execute(&buf, data); err != nil {
		return err
	}

	if err := writeMarkdown("class_summaries.md", buf); err != nil {
		return err
	}

	return nil
}

// HH:MM:SS.ss (drop leading 0h)
func fmtDur(d time.Duration) string {
	neg := d < 0
	if neg {
		d = -d
	}
	h := int(d / time.Hour)
	d -= time.Duration(h) * time.Hour
	m := int(d / time.Minute)
	d -= time.Duration(m) * time.Minute
	s := float64(d) / float64(time.Second)

	if h > 0 {
		if neg {
			return fmt.Sprintf("-%d:%02d:%05.2f", h, m, s)
		}
		return fmt.Sprintf("%d:%02d:%05.2f", h, m, s)
	}
	if neg {
		return fmt.Sprintf("-%d:%05.2f", m, s)
	}
	return fmt.Sprintf("%d:%05.2f", m, s)
}

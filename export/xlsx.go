package export

import (
	"fmt"
	"strconv"

	"github.com/kradalby/qlimaster/quiz"
	"github.com/xuri/excelize/v2"
)

// XLSX writes the quiz as an XLSX file at path. The first row is the
// bold-formatted header; the last row is the averages row formatted in
// italic. Numeric cells are written as numbers (not strings) so
// spreadsheet tools can sort and sum them.
func XLSX(path string, q quiz.Quiz) error {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	const sheet = "Quiz"
	if err := createSheet(f, sheet); err != nil {
		return err
	}
	if err := writeHeaderRow(f, sheet, Header(q.Config)); err != nil {
		return err
	}
	rows := BuildRows(q)
	if err := writeBodyRows(f, sheet, rows); err != nil {
		return err
	}
	if err := styleAveragesRow(f, sheet, len(Header(q.Config)), len(rows)); err != nil {
		return err
	}
	if err := f.SaveAs(path); err != nil {
		return fmt.Errorf("save %s: %w", path, err)
	}
	return nil
}

func createSheet(f *excelize.File, sheet string) error {
	idx, err := f.NewSheet(sheet)
	if err != nil {
		return fmt.Errorf("new sheet: %w", err)
	}
	f.SetActiveSheet(idx)
	if _, err := f.GetSheetIndex("Sheet1"); err == nil {
		_ = f.DeleteSheet("Sheet1")
	}
	return nil
}

func writeHeaderRow(f *excelize.File, sheet string, header []string) error {
	headerStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return fmt.Errorf("header style: %w", err)
	}
	for i, h := range header {
		cell := mustCoord(i+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return fmt.Errorf("set header %s: %w", cell, err)
		}
	}
	if err := f.SetCellStyle(sheet, "A1", mustCoord(len(header), 1), headerStyle); err != nil {
		return fmt.Errorf("header style range: %w", err)
	}
	return nil
}

func writeBodyRows(f *excelize.File, sheet string, rows []Row) error {
	for ri, r := range rows {
		for ci, v := range r.Values {
			cell := mustCoord(ci+1, ri+2)
			if err := writeCell(f, sheet, cell, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeCell(f *excelize.File, sheet, cell, v string) error {
	if n, ok := parseNumberCell(v); ok {
		if err := f.SetCellValue(sheet, cell, n); err != nil {
			return fmt.Errorf("set cell %s: %w", cell, err)
		}
		return nil
	}
	if err := f.SetCellValue(sheet, cell, v); err != nil {
		return fmt.Errorf("set cell %s: %w", cell, err)
	}
	return nil
}

func styleAveragesRow(f *excelize.File, sheet string, cols, rowCount int) error {
	if rowCount == 0 {
		return nil
	}
	avgStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Italic: true, Color: "#777777"},
	})
	if err != nil {
		return fmt.Errorf("avg style: %w", err)
	}
	avgRow := rowCount + 1 // +1 for header
	if err := f.SetCellStyle(sheet,
		mustCoord(1, avgRow),
		mustCoord(cols, avgRow),
		avgStyle); err != nil {
		return fmt.Errorf("avg style range: %w", err)
	}
	return nil
}

func mustCoord(col, row int) string {
	c, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return "A1"
	}
	return c
}

// parseNumberCell tries to interpret a cell value as a number. European
// decimal commas are tolerated.
func parseNumberCell(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	// Handle European comma-decimal as well as US dot.
	trimmed := s
	for i := range trimmed {
		if trimmed[i] == ',' {
			trimmed = trimmed[:i] + "." + trimmed[i+1:]
			break
		}
	}
	v, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

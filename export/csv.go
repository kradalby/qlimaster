package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"github.com/kradalby/qlimaster/quiz"
)

// CSV writes the quiz as a standard RFC 4180 CSV to w. Numbers are rendered
// with '.' decimal separator (the programmatic default; spreadsheets
// importing from this file can coerce locale). The header row is always
// the first line; the averages row is always the last.
func CSV(w io.Writer, q quiz.Quiz) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(Header(q.Config)); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	for _, r := range BuildRows(q) {
		if err := cw.Write(r.Values); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}
	return nil
}

// CSVFile is a convenience wrapper around CSV that writes to the named
// path. The file is truncated if it exists.
func CSVFile(path string, q quiz.Quiz) error {
	f, err := os.Create(path) //nolint:gosec // path is user-supplied by design
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return CSV(f, q)
}

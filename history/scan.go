package history

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/kradalby/qlimaster/store"
)

// dateFolderPattern matches folders named "YYYY-MM-DD..." so history scan
// can derive a date even when the folder name has a suffix like
// "2024-07-17-quiz". Non-matching folders still have their quiz file
// considered; the date is then the file's mtime.
var dateFolderPattern = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})`)

// Scan walks root and returns a History built from every
// "*/quiz.hujson" file it finds. Directories whose name starts with a
// YYYY-MM-DD prefix contribute that date as the LastSeen; other quizzes
// fall back to the file modification date. Errors reading individual
// quiz files are skipped so one corrupt file does not block scanning.
func Scan(root string) (History, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return History{}, fmt.Errorf("readdir %s: %w", root, err)
	}
	merged := History{Version: 1}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		qp := filepath.Join(root, e.Name(), "quiz.hujson")
		q, err := store.Load(qp)
		if err != nil {
			continue // missing or unparseable file -> skip silently
		}
		date := scanDate(root, e.Name(), qp)
		merged = RecordQuiz(merged, q, date)
	}
	return merged, nil
}

// scanDate resolves the date associated with a discovered quiz file.
func scanDate(root, folder, quizPath string) time.Time {
	_ = root
	if m := dateFolderPattern.FindStringSubmatch(folder); len(m) == 2 {
		if t, err := time.Parse("2006-01-02", m[1]); err == nil {
			return t
		}
	}
	if info, err := os.Stat(quizPath); err == nil {
		return info.ModTime()
	}
	return time.Now()
}

// Package fuzzy is a thin, deterministic, case-insensitive adapter around
// fzf's FuzzyMatchV2 algorithm.
//
// The exported Match function takes a query and a slice of candidate
// strings and returns scored matches sorted by descending score with
// alphabetical tiebreak on the lower-cased item. An empty query yields
// every candidate in input order with score zero and no highlight, which
// is what the UI wants when the user has not typed anything yet.
package fuzzy

import (
	"sort"
	"strings"
	"sync"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// initOnce ensures the fzf scoring scheme is initialised exactly once
// across the process. The package's behaviour does not depend on any
// particular scheme; "default" is sufficient for team names.
var initOnce sync.Once

// Match represents a single fuzzy-match hit. Score is 0 when the query is
// empty (all-pass mode) and positive otherwise. Positions is the byte
// offsets of matched runes within the original Item; it is nil when the
// query is empty.
type Match struct {
	Item      string
	Score     int
	Positions []int
}

// Do returns the matches for query against items. If query is empty,
// every item is returned with zero score and no positions, preserving
// input order. Matches are case-insensitive.
func Do(query string, items []string) []Match {
	initOnce.Do(func() { algo.Init("default") })

	query = strings.TrimSpace(query)
	if query == "" {
		out := make([]Match, len(items))
		for i, it := range items {
			out[i] = Match{Item: it}
		}
		return out
	}

	queryRunes := []rune(query)
	out := make([]Match, 0, len(items))
	for _, it := range items {
		chars := util.ToChars([]byte(it))
		res, posPtr := algo.FuzzyMatchV2(
			false, // caseSensitive
			true,  // normalize (strip diacritics)
			true,  // forward
			&chars,
			queryRunes,
			true, // withPositions
			nil,
		)
		if res.Score <= 0 {
			continue
		}
		var positions []int
		if posPtr != nil {
			positions = append([]int(nil), (*posPtr)...)
			sort.Ints(positions)
		}
		out = append(out, Match{
			Item:      it,
			Score:     res.Score,
			Positions: positions,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return strings.ToLower(out[i].Item) < strings.ToLower(out[j].Item)
	})
	return out
}

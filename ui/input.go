package ui

// isArrowUp and isArrowDown check for the actual arrow keys and reject
// the hjkl aliases. Text-input fields (score prompts, team-name
// pickers, config form fields) must accept letters verbatim instead of
// treating "k" as "up" or "j" as "down".
func isArrowUp(k string) bool   { return k == "up" }
func isArrowDown(k string) bool { return k == "down" }

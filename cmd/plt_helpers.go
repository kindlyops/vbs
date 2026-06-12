// Copyright © 2026 Kindly Ops, LLC <support@kindlyops.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"strings"
	"unicode"
)

// ticksPerSecond converts .NET-style 100-nanosecond ticks to seconds.
const ticksPerSecond = 10_000_000

// languageNames maps numeric MepsLanguage IDs to their written-language codes.
// The source app identifies languages by integer; only verified IDs are listed.
var languageNames = map[int]string{
	420: "ASL",
}

// ticksToSeconds converts a 100-nanosecond tick count to seconds.
func ticksToSeconds(ticks int64) float64 {
	return float64(ticks) / float64(ticksPerSecond)
}

// resolveLanguage returns the written-language code for a MepsLanguage ID.
// A non-empty override always wins; otherwise the embedded map is consulted
// and an unmapped ID is a fatal error naming the override flag.
func resolveLanguage(id int, override string) (string, error) {
	if override != "" {
		return override, nil
	}

	if code, ok := languageNames[id]; ok {
		return code, nil
	}

	return "", fmt.Errorf("unknown language id %d; set it explicitly with --lang", id)
}

// slugify renders a display name as a filesystem-safe slug: lowercase ASCII
// letters, digits, and hyphens. Apostrophes are dropped (so "don't" becomes
// "dont"); every other non-alphanumeric rune becomes a separator. Runs of
// separators collapse to one hyphen and leading/trailing hyphens are trimmed.
func slugify(s string) string {
	var b strings.Builder

	for _, r := range s {
		r = unicode.ToLower(r)
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		case r == '\'' || r == '’' || r == '‘':
			// drop apostrophes so possessives stay one word
		default:
			b.WriteByte('-')
		}
	}

	return strings.Trim(collapseHyphens(b.String()), "-")
}

// collapseHyphens replaces runs of hyphens with a single hyphen.
func collapseHyphens(s string) string {
	var b strings.Builder

	prevHyphen := false
	for _, r := range s {
		if r == '-' {
			if !prevHyphen {
				b.WriteRune(r)
			}
			prevHyphen = true
			continue
		}

		b.WriteRune(r)
		prevHyphen = false
	}

	return b.String()
}

// uniqueSlug returns base, or base-2/base-3/... when base was already issued.
// seen tracks how many times each base has been requested.
func uniqueSlug(base string, seen map[string]int) string {
	seen[base]++
	if seen[base] == 1 {
		return base
	}

	return fmt.Sprintf("%s-%d", base, seen[base])
}

package ai

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrForbiddenWord = errors.New("ai guardrail: forbidden investment advisory language detected")
	ErrScoreTampered = errors.New("ai guardrail: detected attempt to modify or invent new quant score")
	ErrMissingStale  = errors.New("ai guardrail: stale run data missing mandatory warning prefix")
)

var forbiddenRegex = regexp.MustCompile(`(?i)(pasti\s+naik|pasti\s+untung|jaminan\s+profit|rekomendasi\s+beli|guaranteed\s+return|sure\s+profit|buy\s+signal|target\s+harga\s+pasti)`)

// scorePattern matches patterns like "gl_score = 95" or "gl_score: 99.5"
var scorePattern = regexp.MustCompile(`(?i)(gl_score|graham_score|lynch_score)\s*[:=]\s*([0-9]+(?:\.[0-9]+)?)`)

// ValidateText ensures the output respects guardrails (forbidden words & score modification).
func ValidateText(text string, expectedGL, expectedGraham, expectedLynch float64) error {
	if forbiddenRegex.MatchString(text) {
		match := forbiddenRegex.FindString(text)
		return fmt.Errorf("%w: %q", ErrForbiddenWord, match)
	}

	// Verify any quoted scores match exact expected scores (avoid hallucinations).
	matches := scorePattern.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		var val float64
		fmt.Sscanf(m[2], "%f", &val)
		name := strings.ToLower(m[1])
		switch name {
		case "gl_score":
			if fmt.Sprintf("%.1f", val) != fmt.Sprintf("%.1f", expectedGL) {
				return fmt.Errorf("%w: quoted %s as %.1f instead of %.1f", ErrScoreTampered, name, val, expectedGL)
			}
		case "graham_score":
			if fmt.Sprintf("%.1f", val) != fmt.Sprintf("%.1f", expectedGraham) {
				return fmt.Errorf("%w: quoted %s as %.1f instead of %.1f", ErrScoreTampered, name, val, expectedGraham)
			}
		case "lynch_score":
			if fmt.Sprintf("%.1f", val) != fmt.Sprintf("%.1f", expectedLynch) {
				return fmt.Errorf("%w: quoted %s as %.1f instead of %.1f", ErrScoreTampered, name, val, expectedLynch)
			}
		}
	}
	return nil
}

// ValidateAnalysis checks complete analysis against guardrails.
func ValidateAnalysis(a *Analysis, input CandidateInput) error {
	if input.Stale && !a.IsStale {
		return ErrMissingStale
	}
	if err := ValidateText(a.Summary, input.Stock.GLScore, input.Stock.GrahamScore, input.Stock.LynchScore); err != nil {
		return err
	}
	if err := ValidateText(a.WhyRanked, input.Stock.GLScore, input.Stock.GrahamScore, input.Stock.LynchScore); err != nil {
		return err
	}
	if err := ValidateText(a.BullCase, input.Stock.GLScore, input.Stock.GrahamScore, input.Stock.LynchScore); err != nil {
		return err
	}
	if err := ValidateText(a.BearCase, input.Stock.GLScore, input.Stock.GrahamScore, input.Stock.LynchScore); err != nil {
		return err
	}
	for _, s := range a.Strengths {
		if err := ValidateText(s, input.Stock.GLScore, input.Stock.GrahamScore, input.Stock.LynchScore); err != nil {
			return err
		}
	}
	for _, r := range a.Risks {
		if err := ValidateText(r, input.Stock.GLScore, input.Stock.GrahamScore, input.Stock.LynchScore); err != nil {
			return err
		}
	}
	return nil
}

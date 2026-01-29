package toolset

import (
	"fmt"
	"strconv"
	"strings"
)

// Enumeration value definitions
var (
	ValidSeverities = map[int]bool{1: true, 2: true, 3: true}
	ValidCates      = map[string]bool{"prometheus": true, "host": true, "elasticsearch": true, "loki": true, "$all": true}
	ValidRuleProds  = map[string]bool{"host": true, "metric": true, "loki": true, "anomaly": true}
	ValidRecovered  = map[int]bool{-1: true, 0: true, 1: true}
)

// ValidateTimeRange validates time range
func ValidateTimeRange(hours int64, stime, etime int64) error {
	// hours and stime/etime are mutually exclusive
	if hours > 0 && (stime > 0 || etime > 0) {
		return fmt.Errorf("hours and stime/etime are mutually exclusive, use one or the other")
	}

	// stime must be less than etime
	if stime > 0 && etime > 0 && stime >= etime {
		return fmt.Errorf("stime (%d) must be less than etime (%d)", stime, etime)
	}

	return nil
}

// ValidatePagination validates pagination parameters
func ValidatePagination(limit, page int) error {
	if limit < 0 {
		return fmt.Errorf("limit must be >= 0, got %d", limit)
	}
	if page < 0 {
		return fmt.Errorf("page must be >= 0, got %d", page)
	}
	return nil
}

// ValidateSeverity validates severity enum (comma-separated multiple values)
func ValidateSeverity(severity string) error {
	if severity == "" {
		return nil
	}
	for _, s := range strings.Split(severity, ",") {
		sev, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || !ValidSeverities[sev] {
			return fmt.Errorf("invalid severity value: %s, must be 1, 2, or 3", s)
		}
	}
	return nil
}

// ValidateCate validates cate enum
func ValidateCate(cate string) error {
	if cate == "" {
		return nil
	}
	if !ValidCates[cate] {
		return fmt.Errorf("invalid cate: %s, valid values: prometheus, host, elasticsearch, loki, $all", cate)
	}
	return nil
}

// ValidateRuleProds validates rule_prods enum (comma-separated multiple values)
func ValidateRuleProds(ruleProds string) error {
	if ruleProds == "" {
		return nil
	}
	for _, p := range strings.Split(ruleProds, ",") {
		prod := strings.TrimSpace(p)
		if !ValidRuleProds[prod] {
			return fmt.Errorf("invalid rule_prod: %s, valid values: host, metric, loki, anomaly", prod)
		}
	}
	return nil
}

// ValidateIsRecovered validates is_recovered enum
func ValidateIsRecovered(isRecovered int) error {
	if !ValidRecovered[isRecovered] {
		return fmt.Errorf("invalid is_recovered: %d, valid values: -1 (all), 0 (not recovered), 1 (recovered)", isRecovered)
	}
	return nil
}

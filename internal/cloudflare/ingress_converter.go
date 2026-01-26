package cloudflare

import (
	"github.com/selfhostly/internal/db"
)

// ConvertToCloudflareRules converts db.IngressRule to cloudflare.IngressRule
func ConvertToCloudflareRules(dbRules []db.IngressRule) []IngressRule {
	cfRules := make([]IngressRule, len(dbRules))
	for i, rule := range dbRules {
		cfRule := IngressRule{
			Service:       rule.Service,
			OriginRequest: rule.OriginRequest,
		}
		if rule.Hostname != nil {
			cfRule.Hostname = *rule.Hostname
		}
		if rule.Path != nil {
			cfRule.Path = *rule.Path
		}
		cfRules[i] = cfRule
	}
	return cfRules
}

// ConvertFromCloudflareRules converts cloudflare.IngressRule to db.IngressRule
func ConvertFromCloudflareRules(cfRules []IngressRule) []db.IngressRule {
	dbRules := make([]db.IngressRule, len(cfRules))
	for i, rule := range cfRules {
		dbRule := db.IngressRule{
			Service:       rule.Service,
			OriginRequest: rule.OriginRequest,
		}
		if rule.Hostname != "" {
			dbRule.Hostname = &rule.Hostname
		}
		if rule.Path != "" {
			dbRule.Path = &rule.Path
		}
		dbRules[i] = dbRule
	}
	return dbRules
}

// EnsureCatchAllRule adds a catch-all 404 rule if not present
func EnsureCatchAllRule(rules []IngressRule) []IngressRule {
	if len(rules) == 0 {
		return []IngressRule{{Service: "http_status:404"}}
	}

	// Check if last rule is a catch-all
	lastRule := rules[len(rules)-1]
	if lastRule.Service == "http_status:404" {
		return rules
	}

	// Append catch-all rule
	return append(rules, IngressRule{Service: "http_status:404"})
}

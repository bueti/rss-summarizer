// Package topicnorm is the single authoritative source of topic-name
// normalization. Articles flow through the LLM (returning raw strings) into
// the repository (storing them), and both sides need identical dedupe /
// casing / mapping rules; previously each maintained its own copy and they
// drifted apart.
package topicnorm

import "strings"

// topicMapping folds specific LLM-suggested topics into broad categories so
// we end up with a small, consistent taxonomy in the UI. The key is the
// lowercased input; the value is the canonical display form.
var topicMapping = map[string]string{
	// Programming Languages
	"golang":             "Go",
	"go programming":     "Go",
	"rust programming":   "Rust",
	"python programming": "Python",
	"javascript":         "JavaScript",
	"typescript":         "TypeScript",

	// Cloud & Infrastructure
	"k8s":                     "Kubernetes",
	"kubernetes deployment":   "Kubernetes",
	"container orchestration": "Kubernetes",
	"docker containers":       "Docker",
	"cloud computing":         "Cloud",
	"aws services":            "AWS",
	"amazon web services":     "AWS",
	"google cloud platform":   "GCP",
	"google cloud":            "GCP",
	"azure cloud":             "Azure",
	"cloud infrastructure":    "Cloud",

	// DevOps
	"devops":                 "DevOps",
	"ci/cd":                  "DevOps",
	"continuous integration": "DevOps",
	"infrastructure as code": "DevOps",

	// Security
	"cybersecurity":          "Security",
	"information security":   "Security",
	"application security":   "Security",
	"network security":       "Security",
	"security vulnerability": "Security",

	// AI & ML
	"artificial intelligence": "AI",
	"machine learning":        "AI",
	"deep learning":           "AI",
	"neural networks":         "AI",
	"llm":                     "AI",
	"large language models":   "AI",
	"chatgpt":                 "AI",
	"gpt":                     "AI",

	// Databases
	"postgresql": "Databases",
	"postgres":   "Databases",
	"mysql":      "Databases",
	"mongodb":    "Databases",
	"sql":        "Databases",
	"database":   "Databases",

	// Web & APIs
	"web development":      "Web",
	"frontend":             "Web",
	"backend":              "Web",
	"frontend development": "Web",
	"backend development":  "Web",
	"full stack":           "Web",
	"api development":      "APIs",
	"rest api":             "APIs",
	"rest":                 "APIs",
	"graphql":              "APIs",

	// Engineering
	"software development":  "Engineering",
	"software engineering":  "Engineering",
	"code quality":          "Engineering",
	"software architecture": "Architecture",
	"system design":         "Architecture",
	"microservices":         "Architecture",

	// Other
	"performance optimization": "Performance",
	"software testing":         "Testing",
	"unit testing":             "Testing",
	"integration testing":      "Testing",
	"test automation":          "Testing",
	"version control":          "Git",
	"source control":           "Git",
	"open source":              "Open Source",
	"opensource":               "Open Source",
	"technology":               "Tech",
}

// acronyms preserves the canonical casing of technical acronyms when a topic
// doesn't match a mapping entry but is itself an acronym. Title-case would
// butcher "AI" into "Ai".
var acronyms = map[string]string{
	"ai": "AI", "api": "API", "apis": "APIs", "aws": "AWS", "gcp": "GCP",
	"devops": "DevOps", "cicd": "CI/CD", "ml": "ML", "llm": "LLM", "llms": "LLMs",
	"ui": "UI", "ux": "UX", "css": "CSS", "html": "HTML", "json": "JSON",
	"xml": "XML", "sql": "SQL", "nosql": "NoSQL", "rest": "REST",
	"graphql": "GraphQL", "grpc": "gRPC", "http": "HTTP", "https": "HTTPS",
	"ssh": "SSH", "tcp": "TCP", "udp": "UDP", "dns": "DNS", "cdn": "CDN",
	"saas": "SaaS", "paas": "PaaS", "iaas": "IaaS", "oauth": "OAuth",
	"jwt": "JWT", "tls": "TLS", "ssl": "SSL", "vpn": "VPN",
}

// Normalize trims each input, folds it through the broad-category mapping,
// applies acronym casing or ASCII title-case, and deduplicates
// case-insensitively. Order of first occurrence is preserved.
func Normalize(topics []string) []string {
	if len(topics) == 0 {
		return topics
	}
	seen := make(map[string]struct{}, len(topics))
	result := make([]string, 0, len(topics))
	for _, t := range topics {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		lower := strings.ToLower(t)
		var normalized string
		switch {
		case topicMapping[lower] != "":
			normalized = topicMapping[lower]
		case acronyms[lower] != "":
			normalized = acronyms[lower]
		default:
			normalized = titleCaseASCII(lower)
		}
		key := strings.ToLower(normalized)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

// titleCaseASCII upper-cases the first rune of each whitespace-/hyphen-/slash-
// separated word. strings.Title is deprecated and mishandles non-ASCII
// casing; acronyms and technical terms are handled separately above.
func titleCaseASCII(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	capitalizeNext := true
	for _, r := range s {
		if r == ' ' || r == '-' || r == '/' {
			capitalizeNext = true
			b.WriteRune(r)
			continue
		}
		if capitalizeNext && r >= 'a' && r <= 'z' {
			b.WriteRune(r - ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
		capitalizeNext = false
	}
	return b.String()
}

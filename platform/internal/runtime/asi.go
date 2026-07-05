package aapruntime

// ASITitles are the official OWASP Top 10 for Agentic Applications (2026)
// category titles, ASI01–ASI10, as published by the OWASP GenAI Security
// Project on 2026-07-05 verification. This is the single source of truth:
// both the section registry (ArtifactSections) and the scaffold template
// consume it, so the two cannot drift apart. When OWASP revises the list,
// change it here only.
//
// Source: https://genai.owasp.org/resource/owasp-top-10-for-agentic-applications-for-2026/
var ASITitles = []string{
	"ASI01 Agent Goal Hijack",
	"ASI02 Tool Misuse & Exploitation",
	"ASI03 Agent Identity & Privilege Abuse",
	"ASI04 Agentic Supply Chain Compromise",
	"ASI05 Unexpected Code Execution",
	"ASI06 Memory & Context Poisoning",
	"ASI07 Insecure Inter-Agent Communication",
	"ASI08 Cascading Agent Failures",
	"ASI09 Human-Agent Trust Exploitation",
	"ASI10 Rogue Agents",
}

// securityChecklistSections returns the full required-section list for the
// security-governance checklist: the ten ASI controls plus the fixed
// governance sections.
func securityChecklistSections() []string {
	sections := make([]string, 0, len(ASITitles)+4)
	sections = append(sections, ASITitles...)
	sections = append(sections,
		"RBAC Summary", "Data Classification Summary", "Audit Obligations", "Kill-Switch Path")
	return sections
}

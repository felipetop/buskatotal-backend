package app

import (
	"html"
	"strings"
	"testing"
)

func TestBuildDeletionNotifyHTML_EscapesInput(t *testing.T) {
	maliciousName := `<script>alert("xss")</script>`
	maliciousEmail := `attacker@test.com"><img src=x>`
	maliciousReason := `<b>bold</b> & "quotes"`

	result := buildDeletionNotifyHTML(maliciousName, maliciousEmail, maliciousReason)

	if strings.Contains(result, "<script>") {
		t.Error("HTML should escape script tags in name")
	}
	if strings.Contains(result, "<img") {
		t.Error("HTML should escape img tags in email")
	}
	if !strings.Contains(result, html.EscapeString(maliciousName)) {
		t.Error("name should be HTML-escaped")
	}
	if !strings.Contains(result, html.EscapeString(maliciousEmail)) {
		t.Error("email should be HTML-escaped")
	}
	if !strings.Contains(result, html.EscapeString(maliciousReason)) {
		t.Error("reason should be HTML-escaped")
	}
}

func TestBuildDeletionConfirmHTML_EscapesInput(t *testing.T) {
	maliciousName := `<script>alert(1)</script>`
	dpoEmail := "dpo@buskatotal.com.br"

	result := buildDeletionConfirmHTML(maliciousName, dpoEmail)

	if strings.Contains(result, "<script>") {
		t.Error("HTML should escape script tags in name")
	}
	if !strings.Contains(result, html.EscapeString(maliciousName)) {
		t.Error("name should be HTML-escaped")
	}
	if !strings.Contains(result, dpoEmail) {
		t.Error("DPO email should be present")
	}
}

func TestBuildDeletionConfirmHTML_UsesDpoEmail(t *testing.T) {
	result := buildDeletionConfirmHTML("João", "custom-dpo@company.com")

	if !strings.Contains(result, "custom-dpo@company.com") {
		t.Error("should use the provided DPO email, not hardcoded")
	}
}

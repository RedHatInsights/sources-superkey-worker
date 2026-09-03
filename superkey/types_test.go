package superkey

import (
	"fmt"
	"strings"
	"testing"
)

func TestCreateRequestString_RedactsSensitiveFields(t *testing.T) {
	req := &CreateRequest{
		IdentityHeader:  "base64-encoded-identity-secret",
		OrgIdHeader:     "org-12345-secret",
		TenantID:        "tenant-1",
		SourceID:        "src-42",
		ApplicationID:   "app-99",
		ApplicationType: "/insights/platform/cost-management",
		SuperKey:        "super-secret-key-value",
		Provider:        "amazon",
		Extra:           map[string]string{"account": "123456789", "external_id": "ext-secret"},
		SuperKeySteps:   []Step{{Step: 1, Name: "s3"}, {Step: 2, Name: "iam_role"}},
	}

	result := req.String()

	// Must contain non-sensitive fields
	if !strings.Contains(result, "tenant-1") {
		t.Error("expected TenantID in output")
	}
	if !strings.Contains(result, "src-42") {
		t.Error("expected SourceID in output")
	}
	if !strings.Contains(result, "app-99") {
		t.Error("expected ApplicationID in output")
	}
	if !strings.Contains(result, "amazon") {
		t.Error("expected Provider in output")
	}
	if !strings.Contains(result, "Steps:2") {
		t.Error("expected step count in output")
	}

	// Must NOT contain sensitive fields
	if strings.Contains(result, "base64-encoded-identity-secret") {
		t.Error("IdentityHeader leaked into String() output")
	}
	if strings.Contains(result, "org-12345-secret") {
		t.Error("OrgIdHeader leaked into String() output")
	}
	if strings.Contains(result, "super-secret-key-value") {
		t.Error("SuperKey leaked into String() output")
	}
	if strings.Contains(result, "123456789") {
		t.Error("Extra account leaked into String() output")
	}
	if strings.Contains(result, "ext-secret") {
		t.Error("Extra external_id leaked into String() output")
	}
}

func TestCreateRequestString_NilSafe(t *testing.T) {
	var req *CreateRequest
	result := req.String()
	if result != "<nil>" {
		t.Errorf("expected <nil>, got %s", result)
	}
}

func TestCreateRequestString_UsedByFmt(t *testing.T) {
	req := &CreateRequest{
		IdentityHeader: "secret-identity",
		SuperKey:       "secret-key",
		Provider:       "amazon",
		TenantID:       "t-1",
	}

	// %v and %s should both use the String() method
	vResult := fmt.Sprintf("%v", req)
	sResult := fmt.Sprintf("%s", req)

	if strings.Contains(vResult, "secret-identity") {
		t.Error("fmt.Sprintf with v-verb leaked IdentityHeader")
	}
	if strings.Contains(sResult, "secret-identity") {
		t.Error("fmt.Sprintf with s-verb leaked IdentityHeader")
	}
	if vResult != sResult {
		t.Errorf("%%v and %%s produced different output: %q vs %q", vResult, sResult)
	}
}

func TestDestroyRequestString_RedactsSensitiveFields(t *testing.T) {
	req := &DestroyRequest{
		TenantID:  "tenant-1",
		SuperKey:  "super-secret-key-value",
		GUID:      "abcdef1234567890",
		Provider:  "amazon",
		StepsCompleted: map[string]map[string]string{
			"s3":       {"output": "redhat-cost-mgmt-bucket-abc123"},
			"iam_role": {"output": "arn:aws:iam::123456:role/redhat-role"},
		},
		SuperKeySteps: []Step{{Step: 1, Name: "s3"}},
	}

	result := req.String()

	// Must contain non-sensitive fields
	if !strings.Contains(result, "tenant-1") {
		t.Error("expected TenantID in output")
	}
	if !strings.Contains(result, "abcdef1234567890") {
		t.Error("expected GUID in output")
	}
	if !strings.Contains(result, "amazon") {
		t.Error("expected Provider in output")
	}
	if !strings.Contains(result, "StepsCompleted:2") {
		t.Error("expected steps completed count in output")
	}

	// Must NOT contain sensitive fields
	if strings.Contains(result, "super-secret-key-value") {
		t.Error("SuperKey leaked into String() output")
	}
	if strings.Contains(result, "redhat-cost-mgmt-bucket") {
		t.Error("StepsCompleted values leaked into String() output")
	}
	if strings.Contains(result, "arn:aws:iam") {
		t.Error("StepsCompleted ARN leaked into String() output")
	}
}

func TestDestroyRequestString_NilSafe(t *testing.T) {
	var req *DestroyRequest
	result := req.String()
	if result != "<nil>" {
		t.Errorf("expected <nil>, got %s", result)
	}
}

func TestForgedApplicationString_RedactsSensitiveFields(t *testing.T) {
	app := &ForgedApplication{
		GUID: "test-guid-123",
		StepsCompleted: map[string]map[string]string{
			"s3": {"output": "bucket-name"},
		},
		Request: &CreateRequest{
			IdentityHeader: "secret-identity",
			OrgIdHeader:    "secret-org",
			TenantID:       "tenant-1",
			Provider:       "amazon",
		},
	}

	result := app.String()

	if !strings.Contains(result, "test-guid-123") {
		t.Error("expected GUID in output")
	}
	if !strings.Contains(result, "StepsCompleted:1") {
		t.Error("expected steps completed count")
	}
	if strings.Contains(result, "secret-identity") {
		t.Error("IdentityHeader leaked through ForgedApplication")
	}
	if strings.Contains(result, "secret-org") {
		t.Error("OrgIdHeader leaked through ForgedApplication")
	}
	if strings.Contains(result, "bucket-name") {
		t.Error("StepsCompleted values leaked through ForgedApplication")
	}
}

func TestForgedApplicationString_NilRequest(t *testing.T) {
	app := &ForgedApplication{
		GUID: "test-guid",
	}

	result := app.String()
	if !strings.Contains(result, "Request:<nil>") {
		t.Errorf("expected Request:<nil> in output, got %s", result)
	}
}

func TestForgedApplicationString_NilSafe(t *testing.T) {
	var app *ForgedApplication
	result := app.String()
	if result != "<nil>" {
		t.Errorf("expected <nil>, got %s", result)
	}
}

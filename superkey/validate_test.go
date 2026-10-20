package superkey

import (
	"strings"
	"testing"
)

const testGUID = "a1b2c3d4e5f67890"

func TestValidateDestroyRequest_ValidRequest(t *testing.T) {
	req := &DestroyRequest{
		GUID: testGUID,
		StepsCompleted: map[string]map[string]string{
			"s3":          {"output": "redhat-cost-management-bucket-" + testGUID},
			"role":        {"output": "redhat-cost-management-role-" + testGUID, "arn": "arn:aws:iam::123456789012:role/redhat-cost-management-role-" + testGUID},
			"policy":      {"output": "arn:aws:iam::123456789012:policy/redhat-cost-management-policy-" + testGUID},
			"cost_report": {"output": "my-cost-report-" + testGUID},
			"bind_role":   {},
		},
	}

	if err := ValidateDestroyRequest(req); err != nil {
		t.Errorf("expected valid request to pass, got: %s", err)
	}
}

func TestValidateDestroyRequest_EmptyGUID(t *testing.T) {
	req := &DestroyRequest{
		GUID:           "",
		StepsCompleted: map[string]map[string]string{},
	}

	err := ValidateDestroyRequest(req)
	if err == nil {
		t.Fatal("expected error for empty GUID")
	}
	if !strings.Contains(err.Error(), "missing GUID") {
		t.Errorf("unexpected error message: %s", err)
	}
}

func TestValidateDestroyRequest_InvalidGUIDFormat(t *testing.T) {
	tests := []struct {
		name string
		guid string
	}{
		{"too short", "a1b2c3"},
		{"too long", "a1b2c3d4e5f678901234"},
		{"uppercase", "A1B2C3D4E5F67890"},
		{"non-hex chars", "g1h2i3j4k5l67890"},
		{"with spaces", "a1b2c3d4 5f67890"},
		{"special chars", "a1b2c3d4e5f6789!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &DestroyRequest{
				GUID:           tt.guid,
				StepsCompleted: map[string]map[string]string{},
			}

			err := ValidateDestroyRequest(req)
			if err == nil {
				t.Errorf("expected error for GUID %q", tt.guid)
			}
		})
	}
}

func TestValidateDestroyRequest_EmptyStepsCompleted(t *testing.T) {
	req := &DestroyRequest{
		GUID:           testGUID,
		StepsCompleted: map[string]map[string]string{},
	}

	if err := ValidateDestroyRequest(req); err != nil {
		t.Errorf("expected empty steps to pass, got: %s", err)
	}
}

func TestValidateDestroyRequest_S3BucketValidation(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		expectErr bool
	}{
		{"valid bucket", "redhat-cost-management-bucket-" + testGUID, false},
		{"valid bucket different app", "redhat-cloud-meter-bucket-" + testGUID, false},
		{"arbitrary bucket name", "my-production-data", true},
		{"wrong GUID", "redhat-cost-management-bucket-0000000000000000", true},
		{"no redhat prefix", "custom-bucket-" + testGUID, true},
		{"empty", "", false}, // empty output is skipped
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &DestroyRequest{
				GUID: testGUID,
				StepsCompleted: map[string]map[string]string{
					"s3": {"output": tt.output},
				},
			}

			err := ValidateDestroyRequest(req)
			if tt.expectErr && err == nil {
				t.Errorf("expected error for s3 output %q", tt.output)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error for s3 output %q, got: %s", tt.output, err)
			}
		})
	}
}

func TestValidateDestroyRequest_RoleValidation(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		expectErr bool
	}{
		{"valid role", "redhat-cost-management-role-" + testGUID, false},
		{"arbitrary role", "AdminRole", true},
		{"wrong GUID", "redhat-cost-management-role-ffffffffffffffff", true},
		{"no redhat prefix", "my-role-" + testGUID, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &DestroyRequest{
				GUID: testGUID,
				StepsCompleted: map[string]map[string]string{
					"role": {"output": tt.output},
				},
			}

			err := ValidateDestroyRequest(req)
			if tt.expectErr && err == nil {
				t.Errorf("expected error for role output %q", tt.output)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error for role output %q, got: %s", tt.output, err)
			}
		})
	}
}

func TestValidateDestroyRequest_PolicyArnValidation(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		expectErr bool
	}{
		{"valid ARN", "arn:aws:iam::123456789012:policy/redhat-cost-management-policy-" + testGUID, false},
		{"govcloud ARN", "arn:aws-us-gov:iam::123456789012:policy/redhat-cost-management-policy-" + testGUID, false},
		{"china ARN", "arn:aws-cn:iam::123456789012:policy/redhat-cost-management-policy-" + testGUID, false},
		{"arbitrary policy ARN", "arn:aws:iam::123456789012:policy/AdministratorAccess", true},
		{"wrong GUID in ARN", "arn:aws:iam::123456789012:policy/redhat-cost-management-policy-ffffffffffffffff", true},
		{"not an ARN", "redhat-cost-management-policy-" + testGUID, true},
		{"no redhat prefix in policy", "arn:aws:iam::123456789012:policy/custom-policy-" + testGUID, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &DestroyRequest{
				GUID: testGUID,
				StepsCompleted: map[string]map[string]string{
					"policy": {"output": tt.output},
				},
			}

			err := ValidateDestroyRequest(req)
			if tt.expectErr && err == nil {
				t.Errorf("expected error for policy output %q", tt.output)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error for policy output %q, got: %s", tt.output, err)
			}
		})
	}
}

func TestValidateDestroyRequest_CostReportValidation(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		expectErr bool
	}{
		{"valid cost report", "my-cost-report-" + testGUID, false},
		{"valid with redhat prefix", "redhat-cost-management-report-" + testGUID, false},
		{"arbitrary report name", "production-billing-report", true},
		{"wrong GUID", "my-cost-report-ffffffffffffffff", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &DestroyRequest{
				GUID: testGUID,
				StepsCompleted: map[string]map[string]string{
					"cost_report": {"output": tt.output},
				},
			}

			err := ValidateDestroyRequest(req)
			if tt.expectErr && err == nil {
				t.Errorf("expected error for cost_report output %q", tt.output)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error for cost_report output %q, got: %s", tt.output, err)
			}
		})
	}
}

func TestValidateDestroyRequest_BindRoleNoOutput(t *testing.T) {
	req := &DestroyRequest{
		GUID: testGUID,
		StepsCompleted: map[string]map[string]string{
			"bind_role": {},
		},
	}

	if err := ValidateDestroyRequest(req); err != nil {
		t.Errorf("expected bind_role with no output to pass, got: %s", err)
	}
}

func TestValidateDestroyRequest_UnknownStepRequiresGUID(t *testing.T) {
	// Unknown step types must at least contain the GUID.
	req := &DestroyRequest{
		GUID: testGUID,
		StepsCompleted: map[string]map[string]string{
			"future_step": {"output": "some-resource-" + testGUID},
		},
	}

	if err := ValidateDestroyRequest(req); err != nil {
		t.Errorf("expected unknown step with GUID to pass, got: %s", err)
	}

	// Without GUID should fail.
	req.StepsCompleted["future_step"]["output"] = "arbitrary-resource-name"
	if err := ValidateDestroyRequest(req); err == nil {
		t.Error("expected error for unknown step without GUID")
	}
}

func TestValidateDestroyRequest_AttackVector_ArbitraryBucket(t *testing.T) {
	// Simulates an attacker trying to delete an arbitrary S3 bucket.
	req := &DestroyRequest{
		GUID: testGUID,
		StepsCompleted: map[string]map[string]string{
			"s3": {"output": "production-customer-data"},
		},
	}

	err := ValidateDestroyRequest(req)
	if err == nil {
		t.Fatal("SECURITY: arbitrary bucket name should be rejected")
	}
}

func TestValidateDestroyRequest_AttackVector_ArbitraryRole(t *testing.T) {
	// Simulates an attacker trying to delete an arbitrary IAM role.
	req := &DestroyRequest{
		GUID: testGUID,
		StepsCompleted: map[string]map[string]string{
			"role": {"output": "OrganizationAccountAccessRole"},
		},
	}

	err := ValidateDestroyRequest(req)
	if err == nil {
		t.Fatal("SECURITY: arbitrary role name should be rejected")
	}
}

func TestValidateDestroyRequest_AttackVector_ArbitraryPolicy(t *testing.T) {
	// Simulates an attacker trying to delete a customer's IAM policy.
	req := &DestroyRequest{
		GUID: testGUID,
		StepsCompleted: map[string]map[string]string{
			"policy": {"output": "arn:aws:iam::123456789012:policy/AdministratorAccess"},
		},
	}

	err := ValidateDestroyRequest(req)
	if err == nil {
		t.Fatal("SECURITY: arbitrary policy ARN should be rejected")
	}
}

func TestValidateDestroyRequest_AttackVector_GUIDBruteForce(t *testing.T) {
	// Attacker provides a valid-looking name but with a different GUID.
	req := &DestroyRequest{
		GUID: testGUID,
		StepsCompleted: map[string]map[string]string{
			"s3": {"output": "redhat-cost-management-bucket-0000000000000000"},
		},
	}

	err := ValidateDestroyRequest(req)
	if err == nil {
		t.Fatal("SECURITY: mismatched GUID should be rejected")
	}
}

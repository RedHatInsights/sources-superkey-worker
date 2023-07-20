package provider

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/redhatinsights/sources-superkey-worker/amazon"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/superkey"
)

// AmazonProvider struct for implementing the Amazon Provider interface
type AmazonProvider struct {
	Client *amazon.Client
}

// ForgeApplication transforms a superkey request with the amazon provider into a list
// of resources required for the application, specified by the request.
// returns: the new forged application payload with info on what was processed, in case something went wrong.
func (a *AmazonProvider) ForgeApplication(request *superkey.CreateRequest) (*superkey.ForgedApplication, error) {
	guid, err := generateGUID()
	if err != nil {
		return nil, err
	}

	f := &superkey.ForgedApplication{
		StepsCompleted: make(map[string]map[string]string),
		Request:        request,
		Client:         a,
		GUID:           guid,
	}

	for _, step := range request.SuperKeySteps {
		switch step.Name {
		case "s3":
			name := fmt.Sprintf("%v-bucket-%v", getShortName(f.Request.ApplicationType), f.GUID)
			l.Log.Infof("Creating S3 bucket: %v", name)

			err := a.Client.CreateS3Bucket(name)
			if err != nil {
				l.Log.Errorf("Failed to create S3 bucket %v, rolling back superkey request %v", name, f.Request)
				return f, err
			}

			f.MarkCompleted("s3", map[string]string{"output": name})
			l.Log.Infof("Successfully created S3 bucket %v", name)

			// Cost reporting requires a policy so the Reporting job can
			// put things into the S3 bucket.
			if step.Payload == "\"create_cost_policy\"" {
				l.Log.Infof("Creating Reporting S3 Policy %v", name)
				payload := substiteInPayload(amazon.CostS3Policy, f, step.Substitutions)

				err := a.Client.AttachBucketPolicy(name, payload)
				if err != nil {
					l.Log.Errorf("Failed to create Reporting S3 Policy %v, rolling back superkey request %v", name, f.Request)
					return f, err
				}
			}

		case "cost_report":
			payload := substiteInPayload(step.Payload, f, step.Substitutions)
			costReport := amazon.CostReport{}

			err := json.Unmarshal([]byte(payload), &costReport)
			if err != nil {
				return f, err
			}

			costReport.ReportName = fmt.Sprintf("%v-%v", costReport.ReportName, f.GUID)

			l.Log.Infof("Create Cost and Usage Report: %v", costReport.ReportName)
			err = a.Client.CreateCostAndUsageReport(&costReport)
			if err != nil {
				l.Log.Errorf("Failed to create cost and usage request %v, rolling back superkey request %v", costReport.ReportName, f.Request)
				return f, err
			}

			f.MarkCompleted("cost_report", map[string]string{"output": costReport.ReportName})
			l.Log.Infof("Successfully created Cost and Usage Report %v", costReport.ReportName)

		case "policy":
			name := fmt.Sprintf("%v-policy-%v", getShortName(f.Request.ApplicationType), f.GUID)
			payload := substiteInPayload(step.Payload, f, step.Substitutions)
			l.Log.Infof("Creating Policy %v", name)

			arn, err := a.Client.CreatePolicy(name, payload)
			if err != nil {
				l.Log.Errorf("Failed to create Policy %v, rolling back superkey request %v", name, f.Request)
				return f, err
			}

			f.MarkCompleted("policy", map[string]string{"output": *arn})
			l.Log.Infof("Successfully created policy %v", name)

		case "role":
			name := fmt.Sprintf("%v-role-%v", getShortName(f.Request.ApplicationType), f.GUID)
			payload := substiteInPayload(step.Payload, f, step.Substitutions)
			l.Log.Infof("Creating Role %v", name)

			roleArn, err := a.Client.CreateRole(name, payload)
			if err != nil {
				l.Log.Errorf("Failed to create Role %v, rolling back superkey request %v", name, f.Request)
				return f, err
			}

			// Store the Role ARN since that is what we need to return for the Authentication object.
			f.MarkCompleted("role", map[string]string{"output": name, "arn": *roleArn})
			l.Log.Infof("Successfully created role %v", name)

		case "bind_role":
			roleName := f.StepsCompleted["role"]["output"]
			policyArn := f.StepsCompleted["policy"]["output"]
			l.Log.Infof("Binding role %v with policy arn %v", roleName, policyArn)

			err := a.Client.BindPolicyToRole(policyArn, roleName)
			if err != nil {
				l.Log.Errorf("Failed to bind policy %v to role arn %v, rolling back superkey request %v", policyArn, roleName, f.Request)
				return f, err
			}

			f.MarkCompleted("bind_role", map[string]string{})
			l.Log.Infof("Successfully bound role %v to policy %v", roleName, policyArn)

		default:
			l.Log.Infof("%v not implemented yet!", step.Name)
		}
	}

	// Set the username to the role ARN since that is what is needed for this provider.
	username := f.StepsCompleted["role"]["arn"]
	appType := path.Base(f.Request.ApplicationType)
	// Create the payload struct
	f.CreatePayload(&username, nil, &appType)

	return f, nil
}

// generateGUID() generates a short guid for resources
func generateGUID() (string, error) {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// getShortName(string) generates a name off of the application type
func getShortName(name string) string {
	return fmt.Sprintf("redhat-%s", path.Base(name))
}

func substiteInPayload(payload string, f *superkey.ForgedApplication, substitutions map[string]string) string {
	// these are some special case substitutions, where `get_account` implies we need to fetch
	// the account from the payload, and s3 is just the output from the s3 step
	for name, sub := range substitutions {
		switch sub {
		case "get_account":
			accountNumber := f.Request.Extra["account"]
			payload = strings.ReplaceAll(payload, name, accountNumber)
		case "s3":
			s3name := f.StepsCompleted["s3"]["output"]
			payload = strings.ReplaceAll(payload, name, s3name)
		case "generate_external_id":
			externalID, ok := f.Request.Extra["external_id"]
			if ok {
				payload = strings.ReplaceAll(payload, name, externalID)
			}
		}
	}

	return payload
}

// TearDown - provides amazon logic for tearing down a supported application
// returns: error
//
// Basically the StepsCompleted field keeps track of what parts of the forge operation
// went smoothly, and we just go through them in reverse and handle them.
func (a *AmazonProvider) TearDown(f *superkey.ForgedApplication) []error {
	errors := make([]error, 0)

	// -----------------
	// unbind the role first (if it happened) so we can cleanly delete the policy and the role.
	// -----------------
	if f.StepsCompleted["bind_role"] != nil {
		policyArn := f.StepsCompleted["policy"]["output"]
		role := f.StepsCompleted["role"]["output"]

		err := a.Client.UnBindPolicyToRole(policyArn, role)
		if err != nil {
			l.Log.Warnf("Failed to unbind policy %v from role %v", policyArn, role)
			errors = append(errors, err)
		}

		l.Log.Infof("Un-bound policy %v from role %v", policyArn, role)
	}

	// -----------------
	// role/policy/reporting can be deleted independently of each other.
	// -----------------
	if f.StepsCompleted["policy"] != nil {
		policyArn := f.StepsCompleted["policy"]["output"]

		err := a.Client.DestroyPolicy(policyArn)
		if err != nil {
			l.Log.Warnf("Failed to destroy policy %v", policyArn)
			errors = append(errors, err)
		}

		l.Log.Infof("Destroyed policy %v", policyArn)
	}

	if f.StepsCompleted["role"] != nil {
		roleName := f.StepsCompleted["role"]["output"]

		err := a.Client.DestroyRole(roleName)
		if err != nil {
			l.Log.Warnf("Failed to destroy role %v", roleName)
			errors = append(errors, err)
		}

		l.Log.Infof("Destroyed role %v", roleName)
	}

	if f.StepsCompleted["cost_report"] != nil {
		reportName := f.StepsCompleted["cost_report"]["output"]

		err := a.Client.DestroyCostAndUsageReport(reportName)
		if err != nil {
			l.Log.Warnf("Failed to destroy cost and usage report %v", reportName)
			errors = append(errors, err)
		}

		l.Log.Infof("Destroyed Cost and Usage report %v", reportName)
	}

	// -----------------
	// s3 bucket can probably be deleted earlier, but leave it to last just in case
	// other things depend on it.
	// -----------------
	if f.StepsCompleted["s3"] != nil {
		bucket := f.StepsCompleted["s3"]["output"]

		err := a.Client.DestroyS3Bucket(bucket)
		if err != nil {
			l.Log.Warnf("Failed to destroy s3 bucket %v", bucket)
			errors = append(errors, err)
		}

		l.Log.Infof("Destroyed s3 bucket %v", bucket)
	}

	return errors
}

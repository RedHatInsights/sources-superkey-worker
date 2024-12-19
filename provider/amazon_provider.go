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
	"github.com/sirupsen/logrus"
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
		return nil, fmt.Errorf("unable to generate guid: %w", err)
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

			err := a.Client.CreateS3Bucket(name)
			if err != nil {
				return f, fmt.Errorf(`failed to create S3 bucket "%s": %w`, name, err)
			}

			f.MarkCompleted("s3", map[string]string{"output": name})

			l.Log.WithFields(logrus.Fields{"tenant_id": request.TenantID, "source_id": request.SourceID, "application_id": request.ApplicationID}).Infof(`S3 bucket "%s" created`, name)

			// Cost reporting requires a policy so the Reporting job can
			// put things into the S3 bucket.
			if step.Payload == "\"create_cost_policy\"" {
				l.Log.WithFields(logrus.Fields{"tenant_id": request.TenantID, "source_id": request.SourceID, "application_id": request.ApplicationID}).Debugf(`Creating S3 bucket "%s"`, name)

				payload := substiteInPayload(amazon.CostS3Policy, f, step.Substitutions)

				err := a.Client.AttachBucketPolicy(name, payload)
				if err != nil {
					return f, fmt.Errorf(`failed to attach bucket policy to S3 bucket "%s": %w`, name, err)
				}

				l.Log.WithFields(logrus.Fields{"tenant_id": request.TenantID, "source_id": request.SourceID, "application_id": request.ApplicationID}).Infof(`S3 bucket policy attached to bucket "%s"`, name)
			}

		case "cost_report":
			payload := substiteInPayload(step.Payload, f, step.Substitutions)
			costReport := amazon.CostReport{}

			err := json.Unmarshal([]byte(payload), &costReport)
			if err != nil {
				return f, fmt.Errorf(`failed to build cost report with payload "%s": %w`, payload, err)
			}

			costReport.ReportName = fmt.Sprintf("%v-%v", costReport.ReportName, f.GUID)

			l.Log.WithFields(logrus.Fields{"tenant_id": request.TenantID, "source_id": request.SourceID, "application_id": request.ApplicationID}).Debugf(`Creating cost and usage report "%s"`, costReport.ReportName)

			err = a.Client.CreateCostAndUsageReport(&costReport)
			if err != nil {
				return f, fmt.Errorf(`failed to create cost and usage report "%s": %w`, costReport.ReportName, err)
			}

			f.MarkCompleted("cost_report", map[string]string{"output": costReport.ReportName})

			l.Log.WithFields(logrus.Fields{"tenant_id": request.TenantID, "source_id": request.SourceID, "application_id": request.ApplicationID}).Infof(`Cost and usage report "%s" created`, costReport.ReportName)

		case "policy":
			name := fmt.Sprintf("%v-policy-%v", getShortName(f.Request.ApplicationType), f.GUID)
			payload := substiteInPayload(step.Payload, f, step.Substitutions)

			l.Log.WithFields(logrus.Fields{"tenant_id": request.TenantID, "source_id": request.SourceID, "application_id": request.ApplicationID}).Debugf(`Creating policy "%s"`, name)

			arn, err := a.Client.CreatePolicy(name, payload)
			if err != nil {
				return f, fmt.Errorf(`failed to create policy "%s": %w`, name, err)
			}

			f.MarkCompleted("policy", map[string]string{"output": *arn})

			l.Log.WithFields(logrus.Fields{"tenant_id": request.TenantID, "source_id": request.SourceID, "application_id": request.ApplicationID}).Infof(`Policy "%s" created`, name)

		case "role":
			name := fmt.Sprintf("%v-role-%v", getShortName(f.Request.ApplicationType), f.GUID)
			payload := substiteInPayload(step.Payload, f, step.Substitutions)

			l.Log.WithFields(logrus.Fields{"tenant_id": request.TenantID, "source_id": request.SourceID, "application_id": request.ApplicationID}).Debugf(`Creating role "%s"`, name)

			roleArn, err := a.Client.CreateRole(name, payload)
			if err != nil {
				return f, fmt.Errorf(`failed to create role "%s": %w`, name, err)
			}

			// Store the Role ARN since that is what we need to return for the Authentication object.
			f.MarkCompleted("role", map[string]string{"output": name, "arn": *roleArn})

			l.Log.WithFields(logrus.Fields{"tenant_id": request.TenantID, "source_id": request.SourceID, "application_id": request.ApplicationID}).Infof(`Role "%s" created`, name)

		case "bind_role":
			roleName := f.StepsCompleted["role"]["output"]
			policyArn := f.StepsCompleted["policy"]["output"]

			l.Log.WithFields(logrus.Fields{"tenant_id": request.TenantID, "source_id": request.SourceID, "application_id": request.ApplicationID}).Debugf(`Binding role "%s" to policy "%s"`, roleName, policyArn)

			err := a.Client.BindPolicyToRole(policyArn, roleName)
			if err != nil {
				return f, fmt.Errorf(`failed to bind policy "%s" to role "%s": %w`, policyArn, roleName, err)
			}

			f.MarkCompleted("bind_role", map[string]string{})

			l.Log.WithFields(logrus.Fields{"tenant_id": request.TenantID, "source_id": request.SourceID, "application_id": request.ApplicationID}).Infof(`Bound role "%s" to policy "%s"`, roleName, policyArn)

		default:
			return f, fmt.Errorf(`superkey step "%s" not implemented`, step.Name)
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
			errors = append(errors, fmt.Errorf(`failed to unbind policy "%s" from role "%s": %w`, policyArn, role, err))
		}

		l.Log.WithFields(logrus.Fields{"tenant_id": f.Request.TenantID}).Infof(`Policy "%s" unbound from role "%s"`, policyArn, role)
	}

	// -----------------
	// role/policy/reporting can be deleted independently of each other.
	// -----------------
	if f.StepsCompleted["policy"] != nil {
		policyArn := f.StepsCompleted["policy"]["output"]

		err := a.Client.DestroyPolicy(policyArn)
		if err != nil {
			errors = append(errors, fmt.Errorf(`failed to destroy policy "%s": %w`, policyArn, err))
		}

		l.Log.WithFields(logrus.Fields{"tenant_id": f.Request.TenantID}).Infof(`Policy "%s" destroyed`, policyArn)
	}

	if f.StepsCompleted["role"] != nil {
		roleName := f.StepsCompleted["role"]["output"]

		err := a.Client.DestroyRole(roleName)
		if err != nil {
			errors = append(errors, fmt.Errorf(`failed to destroy role "%s": %w`, roleName, err))
		}

		l.Log.WithFields(logrus.Fields{"tenant_id": f.Request.TenantID}).Infof(`Role "%s" destroyed`, roleName)
	}

	if f.StepsCompleted["cost_report"] != nil {
		reportName := f.StepsCompleted["cost_report"]["output"]

		err := a.Client.DestroyCostAndUsageReport(reportName)
		if err != nil {
			errors = append(errors, fmt.Errorf(`failed to destroy cost and usage report "%s": %w`, reportName, err))
		}

		l.Log.WithFields(logrus.Fields{"tenant_id": f.Request.TenantID}).Infof(`Cost and usage report "%s" destroyed`, reportName)
	}

	// -----------------
	// s3 bucket can probably be deleted earlier, but leave it to last just in case
	// other things depend on it.
	// -----------------
	if f.StepsCompleted["s3"] != nil {
		bucket := f.StepsCompleted["s3"]["output"]

		err := a.Client.DestroyS3Bucket(bucket)
		if err != nil {
			errors = append(errors, fmt.Errorf(`failed to destroy S3 bucket "%s": %w`, bucket, err))
		}

		l.Log.WithFields(logrus.Fields{"tenant_id": f.Request.TenantID}).Infof(`S3 bucket "%s" destroyed`, bucket)
	}

	return errors
}

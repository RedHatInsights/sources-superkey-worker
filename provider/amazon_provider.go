package provider

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path"
	"strings"

	"github.com/redhatinsights/sources-superkey-worker/amazon"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
)

// AmazonProvider struct for implementing the Amazon Provider interface
type AmazonProvider struct {
	Client *amazon.Client
}

// ForgeApplication transforms a superkey request with the amazon provider into a list
// of resources required for the application, specified by the request.
// returns: the new forged application payload with info on what was processed, in case something went wrong.
func (a *AmazonProvider) ForgeApplication(request *SuperKeyRequest) (*ForgedApplication, error) {
	f := &ForgedApplication{
		Request:        request,
		StepsCompleted: make(map[string]map[string]string),
		Client:         a,
		GUID:           generateGUID(),
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

		case "policy":
			name := fmt.Sprintf("%v-policy-%v", getShortName(f.Request.ApplicationType), f.GUID)
			payload := substiteInPayload(step.Payload, f, step.Substitutions)
			l.Log.Infof("Creating Policy %v", name)

			arn, err := a.Client.CreatePolicy(name, payload)
			if err != nil {
				l.Log.Error("Failed to create Policy %v, rolling back superkey request %v", name, f.Request)
				return f, err
			}

			f.MarkCompleted("policy", map[string]string{"output": *arn})
			l.Log.Infof("Successfully created policy %v", name)

		case "role":
			name := fmt.Sprintf("%v-role-%v", getShortName(f.Request.ApplicationType), f.GUID)
			payload := substiteInPayload(step.Payload, f, step.Substitutions)
			l.Log.Infof("Creating Role %v", name)

			err := a.Client.CreateRole(name, payload)
			if err != nil {
				l.Log.Error("Failed to create Role %v, rolling back superkey request %v", name, f.Request)
				return f, err
			}

			f.MarkCompleted("role", map[string]string{"output": name})
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
	return f, nil
}

func generateGUID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func getShortName(name string) string {
	return fmt.Sprintf("redhat-%s", path.Base(name))
}

func substiteInPayload(payload string, f *ForgedApplication, substitutions map[string]string) string {
	for name, sub := range substitutions {
		switch sub {
		case "get_account":
			//TODO: get_account from provider
			payload = strings.Replace(payload, name, sub, 1)
		case "s3":
			s3name := f.StepsCompleted["name"]["output"]
			payload = strings.Replace(payload, name, s3name, -1)
		}
	}

	return payload
}

// TearDown - provides amazon logic for tearing down a supported application
// returns: error
//
// Basically the StepsCompleted field keeps track of what parts of the forge operation
// went smoothly, and we just go through them in reverse and handle them.
func (a *AmazonProvider) TearDown(f *ForgedApplication) []error {
	errors := make([]error, 0)

	// unbind the role first (if it happened) so we can cleanly delete the policy and the role.
	if f.StepsCompleted["bind_role"] != nil {
		policyArn := f.StepsCompleted["policy"]["output"]
		role := f.StepsCompleted["role"]["output"]

		err := a.Client.UnBindPolicyToRole(policyArn, role)
		if err != nil {
			l.Log.Warnf("Failed to unbind policy %v from role %v", policyArn, role)
			errors = append(errors, err)
		}
	}

	if f.StepsCompleted["policy"] != nil {
		policyArn := f.StepsCompleted["policy"]["output"]

		err := a.Client.DestroyPolicy(policyArn)
		if err != nil {
			l.Log.Warnf("Failed to destroy policy %v", policyArn)
			errors = append(errors, err)
		}
	}

	if f.StepsCompleted["role"] != nil {
		roleName := f.StepsCompleted["role"]["output"]

		err := a.Client.DestroyRole(roleName)
		if err != nil {
			l.Log.Warnf("Failed to destroy role %v", roleName)
			errors = append(errors, err)
		}
	}

	if f.StepsCompleted["s3"] != nil {
		bucket := f.StepsCompleted["s3"]["output"]

		err := a.Client.DestroyS3Bucket(bucket)
		if err != nil {
			l.Log.Warnf("Failed to destroy s3 bucket %v", bucket)
			errors = append(errors, err)
		}
	}

	return errors
}

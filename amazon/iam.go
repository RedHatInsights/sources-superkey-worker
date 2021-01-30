package amazon

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
)

// CreateRole - creates a role with name from a json payload
// returns: error
func (a *Client) CreateRole(name, payload string) error {
	_, err := a.Iam.CreateRole(context.Background(), &iam.CreateRoleInput{
		AssumeRolePolicyDocument: &payload,
		RoleName:                 &name,
	})

	if err != nil {
		return err
	}

	return nil
}

// DestroyRole - destroys a role with name
// returns: error
func (a *Client) DestroyRole(name string) error {
	_, err := a.Iam.DeleteRole(context.Background(), &iam.DeleteRoleInput{
		RoleName: &name,
	})

	if err != nil {
		return err
	}

	return nil
}

// CreatePolicy - creates an IAM policy with given name + payload, the payload
// comes from the superkey metadata in the job payload
// returns: (ARN of new policy, error)
func (a *Client) CreatePolicy(name, payload string) (*string, error) {
	out, err := a.Iam.CreatePolicy(context.Background(), &iam.CreatePolicyInput{
		PolicyDocument: &payload,
		PolicyName:     &name,
	})

	if err != nil {
		return nil, err
	}

	return out.Policy.Arn, nil
}

// DestroyPolicy - inverse of CreatePolicy, takes an ARN pointing to a Policy
// and destroys it.
// returns: error
func (a *Client) DestroyPolicy(arn string) error {
	_, err := a.Iam.DeletePolicy(context.Background(), &iam.DeletePolicyInput{
		PolicyArn: &arn,
	})

	if err != nil {
		return err
	}

	return nil
}

// BindPolicyToRole - attaches policy (arn) to role (name)
// returns: error
func (a *Client) BindPolicyToRole(policy, role string) error {
	_, err := a.Iam.AttachRolePolicy(context.Background(), &iam.AttachRolePolicyInput{
		PolicyArn: &policy,
		RoleName:  &role,
	})

	if err != nil {
		return err
	}

	return nil
}

// UnBindPolicyToRole - detaches policy (arn) from role (name)
// returns: error
func (a *Client) UnBindPolicyToRole(role, policy string) error {
	_, err := a.Iam.DetachRolePolicy(context.Background(), &iam.DetachRolePolicyInput{
		PolicyArn: &policy,
		RoleName:  &role,
	})

	if err != nil {
		return err
	}

	return nil
}

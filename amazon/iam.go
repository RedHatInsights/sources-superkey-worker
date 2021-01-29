package amazon

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

// CreateRole - creates a role with name from a json payload
// returns: error
func CreateRole(name, payload string, cfg *aws.Config) error {
	client := iam.NewFromConfig(*cfg)

	_, err := client.CreateRole(context.Background(), &iam.CreateRoleInput{
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
func DestroyRole(name string, cfg *aws.Config) error {
	client := iam.NewFromConfig(*cfg)

	_, err := client.DeleteRole(context.Background(), &iam.DeleteRoleInput{
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
func CreatePolicy(name, payload string, cfg *aws.Config) (*string, error) {
	client := iam.NewFromConfig(*cfg)

	out, err := client.CreatePolicy(context.Background(), &iam.CreatePolicyInput{
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
func DestroyPolicy(arn string, cfg *aws.Config) error {
	client := iam.NewFromConfig(*cfg)

	_, err := client.DeletePolicy(context.Background(), &iam.DeletePolicyInput{
		PolicyArn: &arn,
	})

	if err != nil {
		return err
	}

	return nil
}

// BindPolicyToRole - attaches policy (arn) to role (name)
// returns: error
func BindPolicyToRole(policy, role string, cfg *aws.Config) error {
	client := iam.NewFromConfig(*cfg)

	_, err := client.AttachRolePolicy(context.Background(), &iam.AttachRolePolicyInput{
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
func UnBindPolicyToRole(role, policy string, cfg *aws.Config) error {
	client := iam.NewFromConfig(*cfg)

	_, err := client.DetachRolePolicy(context.Background(), &iam.DetachRolePolicyInput{
		PolicyArn: &policy,
		RoleName:  &role,
	})

	if err != nil {
		return err
	}

	return nil
}

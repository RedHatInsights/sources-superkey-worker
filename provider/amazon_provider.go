package provider

import (
	"github.com/redhatinsights/sources-superkey-worker/amazon"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
)

// AmazonProvider struct for implementing the Amazon Provider interface
type AmazonProvider struct {
	Client *amazon.Client
}

// ForgeApplication -=
func (a *AmazonProvider) ForgeApplication(request *SuperKeyRequest) (*ForgedProduct, error) {
	f := ForgedProduct{
		Request:        request,
		StepsCompleted: make(map[string]map[string]string),
		Client:         a,
	}

	for _, step := range request.SuperKeySteps {
		switch step.Name {
		case "s3":
			l.Log.Info("s3 not implemented yet!")
		case "policy":
			l.Log.Info("policy not implemented yet!")
		case "role":
			l.Log.Info("role not implemented yet!")
		case "bind_role":
			l.Log.Info("bind_role not implemented yet!")
		default:
			l.Log.Infof("%v not implemented yet!", step.Name)
		}
	}
	return &f, nil
}

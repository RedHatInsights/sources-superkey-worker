package provider

import (
	"github.com/redhatinsights/sources-superkey-worker/sources"
)

// SuperKeyRequest - struct representing a request for a superkey
type SuperKeyRequest struct {
	TenantID        string         `json:"tenant_id"`
	Provider        string         `json:"provider"`
	ApplicationType string         `json:"application_type"`
	SuperKeySteps   []SuperKeyStep `json:"superkey_steps"`
}

// SuperKeyStep - struct representing a step for SuperKey
type SuperKeyStep struct {
	Step          int               `json:"step"`
	Name          string            `json:"name"`
	Payload       string            `json:"payload"`
	Substitutions map[string]string `json:"substitutions"`
}

// ForgedApplication - struct to hold the output of a superkey create_application
// request
type ForgedApplication struct {
	Product        *sources.SuperKeyApp
	StepsCompleted map[string]map[string]string
	Request        *SuperKeyRequest
	Client         SuperKeyProvider
	GUID           string
}

// MarkCompleted marks a step as completed, storing the passed in hash of data.
func (f *ForgedApplication) MarkCompleted(name string, data map[string]string) {
	f.StepsCompleted[name] = data
}

// SuperKeyProvider the interface for all of the superkey providers
// currently just a single method is needed (ForgeApplication)
type SuperKeyProvider interface {
	ForgeApplication(*SuperKeyRequest) (*ForgedApplication, error)
	TearDown(*ForgedApplication) error
}

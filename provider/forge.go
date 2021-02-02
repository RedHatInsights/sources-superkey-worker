package provider

import (
	"fmt"

	"github.com/redhatinsights/sources-superkey-worker/amazon"
)

// Forge -
func Forge(request *SuperKeyRequest) (*ForgedApplication, error) {
	client, err := getProvider(request)
	if err != nil {
		return nil, err
	}
	f, err := client.ForgeApplication(request)

	// returning both every time. we need the state of the forged product to know what to
	// tear down.
	return f, err
}

func getProvider(request *SuperKeyRequest) (SuperKeyProvider, error) {
	switch request.Provider {
	case "amazon":
		//TODO: fetch creds from sources api
		client, err := amazon.NewClient("asdf", "asdf", getStepNames(request.SuperKeySteps)...)
		if err != nil {
			return nil, err
		}

		return &AmazonProvider{Client: client}, nil
	default:
		return nil, fmt.Errorf("Unsupported auth provider %v", request.Provider)
	}
}

func getStepNames(steps []SuperKeyStep) []string {
	names := make([]string, 0)
	for _, step := range steps {
		names = append(names, step.Name)
	}

	return names
}

// TearDown -
func TearDown(f *ForgedApplication) {
	if f.Client != nil {
		f.Client.TearDown(f)
	}
}

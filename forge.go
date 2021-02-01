package main

import (
	"fmt"

	"github.com/redhatinsights/sources-superkey-worker/amazon"
	"github.com/redhatinsights/sources-superkey-worker/provider"
)

// Forge -
func Forge(request *provider.SuperKeyRequest) (*provider.ForgedProduct, error) {
	//TODO: fetch creds from sources api
	client, err := getProvider(request)
	if err != nil {
		return nil, err
	}
	f, err := client.ForgeApplication(request)

	// returning both every time. we need the state of the forged product to know what to
	// tear down.
	return f, err
}

func getProvider(request *provider.SuperKeyRequest) (provider.SuperKeyProvider, error) {
	switch request.Provider {
	case "amazon":
		client, err := amazon.NewClient("asdf", "asdf", getApis(request.SuperKeySteps)...)
		if err != nil {
			return nil, err
		}

		return &provider.AmazonProvider{Client: client}, nil
	default:
		return nil, fmt.Errorf("Unsupported auth provider %v", request.Provider)
	}
}

func getApis(steps []provider.SuperKeyStep) []string {
	apis := make([]string, 0)
	for _, step := range steps {
		switch step.Name {
		case "s3":
			apis = append(apis, "s3")
		case "role", "policy", "bind_role":
			apis = append(apis, "iam")
		}
	}

	return apis
}

// TearDown -
func TearDown(f provider.ForgedProduct) {

}

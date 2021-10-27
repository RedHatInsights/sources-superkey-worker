package provider

import (
	"fmt"
	"os"
	"path"

	"github.com/redhatinsights/sources-superkey-worker/azure"
	"github.com/redhatinsights/sources-superkey-worker/superkey"
)

type AzureProvider struct {
	Username string
	Password string
	Tenant   string
}

func (az *AzureProvider) ForgeApplication(request *superkey.CreateRequest) (*superkey.ForgedApplication, error) {
	if !CloudigradeTemplateImported {
		return nil, fmt.Errorf("cloudigrade azure template is not imported - resource creation impossible")
	}

	guid, err := generateGUID()
	if err != nil {
		return nil, err
	}

	tmpdir, err := os.MkdirTemp("/tmp/", "az")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary home directory for az cli: %v", err.Error())
	}

	f := &superkey.ForgedApplication{
		StepsCompleted: make(map[string]map[string]string),
		Request:        request,
		Client:         az,
		GUID:           guid,
	}

	name := fmt.Sprintf("redhat-cloudmeter-%v", guid)
	// keeping the generated name so we can clean it up later even if the deploy fails.
	f.StepsCompleted["az-lighthouse"] = map[string]string{"name": name}

	// Allocate the struct that will manage running the create/destroy commands
	cliInstance := azure.AzCli{HomeDirectory: tmpdir}
	defer cliInstance.Cleanup()

	err = cliInstance.Login(az.Username, az.Password, az.Tenant)
	if err != nil {
		return f, err
	}

	subscriptionID, err := cliInstance.DeploySubscriptionTemplate(name, CLOUDIGRADE_AZURE_TEMPLATE_PATH)
	if err != nil {
		return f, err
	}

	err = cliInstance.Logout()
	if err != nil {
		return f, err
	}

	f.StepsCompleted["az-lighthouse"]["subscriptionID"] = subscriptionID
	appType := path.Base(f.Request.ApplicationType)
	f.CreatePayload(&subscriptionID, nil, &appType)

	return f, nil
}

func (az *AzureProvider) TearDown(f *superkey.ForgedApplication) []error {
	tmpdir, err := os.MkdirTemp("/tmp/", "")
	if err != nil {
		return []error{fmt.Errorf("failed to create temporary home directory for az cli: %v", err.Error())}
	}

	name := f.StepsCompleted["az-lighthouse"]["name"]

	// Create the struct that will manage running the create/destroy commands
	cliInstance := azure.AzCli{HomeDirectory: tmpdir}
	defer cliInstance.Cleanup()

	err = cliInstance.Login(az.Username, az.Password, az.Tenant)
	if err != nil {
		return []error{err}
	}

	err = cliInstance.DeleteSubscription(name)
	if err != nil {
		return []error{err}
	}

	err = cliInstance.Logout()
	if err != nil {
		return []error{err}
	}

	return nil
}

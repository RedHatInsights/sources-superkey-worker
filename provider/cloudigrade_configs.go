package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	l "github.com/redhatinsights/sources-superkey-worker/logger"
)

var (
	CLOUDIGRADE_AZURE_TEMPLATE_PATH       = "/tmp/az_payload.json"
	CLOUDIGRADE_AZURE_TEMPLATE_LOCAL_PATH = os.Getenv("AZURE_TEMPLATE_PATH")

	CLOUDIGRADE_URL            = os.Getenv("CLOUD_METER_URL")
	CLOUDIGRADE_SYSCONFIG_PATH = os.Getenv("CLOUD_METER_SYSCONFIG_PATH")

	CloudigradeTemplateImported = false
	CloudigradeConfig           *cloudigradeConfig
)

type cloudigradeConfig struct {
	SysConfig     CloudigradeSysConfig
	AzureTemplate string
}

type CloudigradeSysConfig struct {
	AwsAccountID string `json:"aws_account_id"`
	AwsPolicies  struct {
		TraditionalInspection map[string]interface{}
	} `json:"aws_policies"`
	AzureOfferTemplatePath string `json:"azure_offer_template_path"`
	Version                string `json:"version"`
}

func FetchCloudigradeConfigs() error {
	l.Log.Infof("Fetching cloudigrade sysconfig from [%v%v]", CLOUDIGRADE_URL, CLOUDIGRADE_SYSCONFIG_PATH)
	resp, err := http.DefaultClient.Get(fmt.Sprintf("%v%v", CLOUDIGRADE_URL, CLOUDIGRADE_SYSCONFIG_PATH))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	sysconfig := CloudigradeSysConfig{}
	err = json.Unmarshal(body, &sysconfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cloudigrade sysconfig")
	}

	azureTemplate, err := fetchAzureOfferTemplate(sysconfig.AzureOfferTemplatePath)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(CLOUDIGRADE_AZURE_TEMPLATE_PATH, []byte(azureTemplate), 0644)
	if err != nil {
		return err
	}
	CloudigradeConfig = &cloudigradeConfig{SysConfig: sysconfig, AzureTemplate: azureTemplate}
	return nil
}

func fetchAzureOfferTemplate(path string) (string, error) {
	l.Log.Infof("Fetching cloudigrade azure template from [%v%v]", CLOUDIGRADE_URL, path)
	resp, err := http.DefaultClient.Get(fmt.Sprintf("%v%v", CLOUDIGRADE_URL, path))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func CopyLocalAzureTemplate() error {
	_, err := os.Stat(CLOUDIGRADE_AZURE_TEMPLATE_PATH)

	// we are fine with the file not existing, but if it isn't that specific
	// error, bubble up.
	if !os.IsNotExist(err) {
		return err
	}

	in, err := os.Open(CLOUDIGRADE_AZURE_TEMPLATE_LOCAL_PATH)
	if err != nil {
		return fmt.Errorf("failed to open default template (not mounted)")
	}
	defer in.Close()

	out, err := os.Create(CLOUDIGRADE_AZURE_TEMPLATE_PATH)
	if err != nil {
		return fmt.Errorf("failed to create destination file")
	}
	defer out.Close()

	l.Log.Infof("Copying [%v] to [%v]", CLOUDIGRADE_AZURE_TEMPLATE_LOCAL_PATH, CLOUDIGRADE_AZURE_TEMPLATE_PATH)
	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("failed to copy template file")
	}

	return nil
}

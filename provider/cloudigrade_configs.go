package provider

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

var (
	CLOUDIGRADE_AZURE_TEMPLATE_PATH = "/tmp/az_payload.json"
	CLOUDIGRADE_AZURE_SECRET_PATH   = os.Getenv("AZURE_TEMPLATE_PATH")
)

var CloudigradeTemplateImported = false

type CloudigradeConfig struct {
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

func FetchCloudigradeConfigs() (*CloudigradeConfig, error) {
	resp, err := http.DefaultClient.Get("http://localhost:8000/api/cloudigrade/v2/sysconfig/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	sysconfig := CloudigradeSysConfig{}
	err = json.Unmarshal(body, &sysconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cloudigrade sysconfig")
	}

	azureTemplate, err := fetchAzureOfferTemplate(sysconfig.AzureOfferTemplatePath)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(CLOUDIGRADE_AZURE_TEMPLATE_PATH, []byte(azureTemplate), 0644)
	if err != nil {
		return nil, err
	}

	return &CloudigradeConfig{SysConfig: sysconfig, AzureTemplate: azureTemplate}, nil
}

func fetchAzureOfferTemplate(path string) (string, error) {
	resp, err := http.DefaultClient.Get(fmt.Sprintf("http://localhost:8000%v", path))
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

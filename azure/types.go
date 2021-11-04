package azure

import "time"

type AzureDeploymentOutput []AzureDeployment
type AzureDeployment struct {
	ID         string `json:"id"`
	Location   string `json:"location"`
	Name       string `json:"name"`
	Properties struct {
		CorrelationID string      `json:"correlationId"`
		DebugSetting  interface{} `json:"debugSetting"`
		Dependencies  []struct {
			DependsOn []struct {
				ID           string `json:"id"`
				ResourceName string `json:"resourceName"`
				ResourceType string `json:"resourceType"`
			} `json:"dependsOn"`
			ID           string `json:"id"`
			ResourceName string `json:"resourceName"`
			ResourceType string `json:"resourceType"`
		} `json:"dependencies"`
		Duration string `json:"duration"`
		Error    struct {
			AdditionalInfo interface{} `json:"additionalInfo"`
			Code           string      `json:"code"`
			Details        []struct {
				AdditionalInfo interface{} `json:"additionalInfo"`
				Code           string      `json:"code"`
				Details        interface{} `json:"details"`
				Message        string      `json:"message"`
				Target         interface{} `json:"target"`
			} `json:"details"`
			Message string      `json:"message"`
			Target  interface{} `json:"target"`
		} `json:"error"`
		Mode              string      `json:"mode"`
		OnErrorDeployment interface{} `json:"onErrorDeployment"`
		OutputResources   interface{} `json:"outputResources"`
		Outputs           interface{} `json:"outputs"`
		Parameters        struct {
			Authorizations struct {
				Type  string `json:"type"`
				Value []struct {
					PrincipalID            string `json:"principalId"`
					PrincipalIDDisplayName string `json:"principalIdDisplayName"`
					RoleDefinitionID       string `json:"roleDefinitionId"`
				} `json:"value"`
			} `json:"authorizations"`
			ManagedByTenantID struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"managedByTenantId"`
			MspOfferDescription struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"mspOfferDescription"`
			MspOfferName struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"mspOfferName"`
		} `json:"parameters"`
		ParametersLink interface{} `json:"parametersLink"`
		Providers      []struct {
			ID                                interface{} `json:"id"`
			Namespace                         string      `json:"namespace"`
			ProviderAuthorizationConsentState interface{} `json:"providerAuthorizationConsentState"`
			RegistrationPolicy                interface{} `json:"registrationPolicy"`
			RegistrationState                 interface{} `json:"registrationState"`
			ResourceTypes                     []struct {
				Aliases           interface{}   `json:"aliases"`
				APIProfiles       interface{}   `json:"apiProfiles"`
				APIVersions       interface{}   `json:"apiVersions"`
				Capabilities      interface{}   `json:"capabilities"`
				DefaultAPIVersion interface{}   `json:"defaultApiVersion"`
				LocationMappings  interface{}   `json:"locationMappings"`
				Locations         []interface{} `json:"locations"`
				Properties        interface{}   `json:"properties"`
				ResourceType      string        `json:"resourceType"`
			} `json:"resourceTypes"`
		} `json:"providers"`
		ProvisioningState  string      `json:"provisioningState"`
		TemplateHash       string      `json:"templateHash"`
		TemplateLink       interface{} `json:"templateLink"`
		Timestamp          time.Time   `json:"timestamp"`
		ValidatedResources interface{} `json:"validatedResources"`
	} `json:"properties"`
	Tags interface{} `json:"tags"`
	Type string      `json:"type"`
}

type AzureDeploymentErrorJson struct {
	OdataError struct {
		Code    string `json:"code"`
		Message struct {
			Lang  string `json:"lang"`
			Value string `json:"value"`
		} `json:"message"`
		RequestID string `json:"requestId"`
		Date      string `json:"date"`
	} `json:"odata.error"`
}

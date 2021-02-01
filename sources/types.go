package sources

type SuperKeyApp struct {
	SourceID    string `json:"source_id"`
	Type        string `json:"application_type"`
	Name        string `json:"application_name"`
	AuthType    string `json:"authentication_type"`
	AuthPayload string `json:"authentication_payload"`
}

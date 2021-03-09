package amazon

import (
	"context"

	cost "github.com/aws/aws-sdk-go-v2/service/costandusagereportservice"
	"github.com/aws/aws-sdk-go-v2/service/costandusagereportservice/types"
)

// CreateCostAndUsageReport - creates a cost report based on input
// returns an error if there was a problem
func (a *Client) CreateCostAndUsageReport(costReport *CostReport) error {
	reportDefinition := cost.PutReportDefinitionInput{
		ReportDefinition: &types.ReportDefinition{
			AdditionalSchemaElements: costReport.AdditionalSchemaElements,
			Compression:              costReport.Compression,
			Format:                   costReport.Format,
			ReportName:               &costReport.ReportName,
			S3Bucket:                 &costReport.S3Bucket,
			S3Prefix:                 &costReport.S3Prefix,
			S3Region:                 costReport.S3Region,
			TimeUnit:                 costReport.TimeUnit,
			AdditionalArtifacts:      costReport.AdditionalArtifacts,
		},
	}

	_, err := a.CostReporting.PutReportDefinition(context.Background(), &reportDefinition)
	if err != nil {
		return err
	}

	return nil
}

// DestroyCostAndUsageReport - creates a cost report based on input
// returns an error if there was a problem
func (a *Client) DestroyCostAndUsageReport(name string) error {
	_, err := a.CostReporting.DeleteReportDefinition(context.Background(), &cost.DeleteReportDefinitionInput{
		ReportName: &name,
	})
	if err != nil {
		return err
	}

	return nil
}

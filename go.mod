module github.com/redhatinsights/sources-superkey-worker

go 1.16

require (
	github.com/RedHatInsights/sources-api-go v0.0.0-20220311182318-754f02997a00
	github.com/aws/aws-sdk-go v1.43.17
	github.com/aws/aws-sdk-go-v2 v1.15.0
	github.com/aws/aws-sdk-go-v2/config v1.15.0
	github.com/aws/aws-sdk-go-v2/credentials v1.10.0
	github.com/aws/aws-sdk-go-v2/service/costandusagereportservice v1.13.0
	github.com/aws/aws-sdk-go-v2/service/iam v1.18.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.26.0
	github.com/klauspost/compress v1.15.1 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/prometheus/client_golang v1.12.1
	github.com/redhatinsights/app-common-go v1.6.0
	github.com/redhatinsights/platform-go-middlewares v0.12.0
	github.com/segmentio/kafka-go v0.4.30
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.10.1
	golang.org/x/crypto v0.0.0-20220307211146-efcb8507fb70 // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/sys v0.0.0-20220310020820-b874c991c1a5 // indirect
)

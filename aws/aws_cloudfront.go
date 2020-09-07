package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/gpkfr/goretdep/config"
)

//ViewListInvalidations List Cloudfront Invalidations.
//cfID: Cloudfront ID
func (awsSession *AWSSession) ViewListInvalidations(cfID string) (*cloudfront.ListInvalidationsOutput, error) {
	svc := cloudfront.New(awsSession.Client)

	resp, err := svc.ListInvalidations(&cloudfront.ListInvalidationsInput{
		DistributionId: aws.String(cfID),
	})

	if err != nil {
		return resp, fmt.Errorf("list.inval. err:%v", err.Error())
	}

	return resp, nil
}

//CreateInvalidationRequest Create an Invalidation Request
//cfID: Cloudfront ID
//pattern: string  to invalidate
func (awsSession *AWSSession) CreateInvalidationRequest(cfID, pattern string) (*cloudfront.CreateInvalidationOutput, error) {
	svc := cloudfront.New(awsSession.Client)

	now := time.Now()
	resp, err := svc.CreateInvalidation(&cloudfront.CreateInvalidationInput{
		DistributionId: aws.String(cfID),
		InvalidationBatch: &cloudfront.InvalidationBatch{
			CallerReference: aws.String(fmt.Sprintf("gtdinvali%s", now.Format("2006/01/02,15:04:05"))),
			Paths: &cloudfront.Paths{
				Quantity: aws.Int64(1),
				Items: []*string{
					aws.String(pattern),
				},
			},
		},
	})

	if err != nil {
		return resp, fmt.Errorf("create.inval. err:%v", err.Error())
	}

	return resp, nil
}

func (awsSession *AWSSession) GetInvalidationRequest(cfID, invalidationID string) (*cloudfront.GetInvalidationOutput, error) {

	svc := cloudfront.New(awsSession.Client)

	resp, err := svc.GetInvalidation(&cloudfront.GetInvalidationInput{
		DistributionId: aws.String(cfID),
		Id:             aws.String(invalidationID),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case cloudfront.ErrCodeNoSuchInvalidation:
				fmt.Println(cloudfront.ErrCodeNoSuchInvalidation, aerr.Error())
			case cloudfront.ErrCodeNoSuchDistribution:
				fmt.Println(cloudfront.ErrCodeNoSuchDistribution, aerr.Error())
			case cloudfront.ErrCodeAccessDenied:
				fmt.Println(cloudfront.ErrCodeAccessDenied, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	return resp, err
}

func (awsSession *AWSSession) GetCloudFronts(cloudfronts *config.CloudFronts, env string) {
	if err := config.LoadCloudFront(cloudfronts, &env); err != nil {
		log.Fatal(err)
	}
}

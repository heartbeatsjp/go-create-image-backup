package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/pkg/errors"
)

// EC2MetadataAPI interface of ec2metadata.EC2Metadata.
type EC2MetadataAPI interface {
	Available() bool
	GetInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error)
	Region() (string, error)
}

// AWS provides methods for AWS operations.
type AWS interface {
	GetRegion() (string, error)
	GetInstanceID() (string, error)
	GetInstanceName(ctx context.Context, instanceID string) (string, error)
	CreateImage(ctx context.Context, instanceID, name, now string) (string, error)
	CreateTags(ctx context.Context, resourceID string, tags []*ec2.Tag) error
	GetImages(ctx context.Context, name, service string) ([]*ec2.Image, error)
	GetImage(ctx context.Context, imageID string) (*ec2.Image, error)
	GetSnapshots(ctx context.Context, imageID string) ([]string, error)
	DeregisterImages(ctx context.Context, images []*ec2.Image) error
}

// AWSClient implements AWS.
type AWSClient struct {
	svcEC2         ec2iface.EC2API
	svcEC2Metadata EC2MetadataAPI
}

// NewAWSClient creates a new AWSClient.
func NewAWSClient() *AWSClient {
	sess := session.Must(session.NewSession())
	return &AWSClient{
		svcEC2:         ec2.New(sess, aws.NewConfig().WithRegion("ap-northeast-1")),
		svcEC2Metadata: ec2metadata.New(session.Must(session.NewSession())),
	}
}

// GetRegion returns region, this method available at AWS EC2 instance.
func (client *AWSClient) GetRegion() (string, error) {
	if client.svcEC2Metadata.Available() {
		region, err := client.svcEC2Metadata.Region()
		if err != nil {
			return "", err
		}
		return region, nil
	}
	return "", errors.New("program is not running with EC2 Instance or metadata service is not available")
}

// GetInstanceID returns instance id, this method available at AWS EC2 instance.
func (client *AWSClient) GetInstanceID() (string, error) {
	if client.svcEC2Metadata.Available() {
		i, err := client.svcEC2Metadata.GetInstanceIdentityDocument()
		if err != nil {
			return "", err
		}
		return i.InstanceID, nil
	}
	return "", errors.New("program is not running with EC2 Instance or metadata service is not available")
}

// GetInstanceName returns value of Name tag attached instance id as instance name.
// If Name tag not found, return instance id instead instance name.
func (client *AWSClient) GetInstanceName(ctx context.Context, instanceID string) (string, error) {
	result, err := client.svcEC2.DescribeTagsWithContext(ctx, &ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("resource-id"), Values: []*string{aws.String(instanceID)}},
			{Name: aws.String("tag-key"), Values: []*string{aws.String("Name")}},
		},
	})
	if err != nil {
		return "", err
	}

	if len(result.Tags) < 1 {
		return instanceID, nil
	}

	name := *result.Tags[0].Value
	if name == "" {
		return instanceID, nil
	}
	return name, nil
}

// CreateImage creates machine image for instance which has instance id.
func (client *AWSClient) CreateImage(ctx context.Context, instanceID, name, now string) (string, error) {
	result, err := client.svcEC2.CreateImageWithContext(ctx, &ec2.CreateImageInput{
		InstanceId:  aws.String(instanceID),
		Description: aws.String("create by go-create-image-backup"),
		Name:        aws.String(fmt.Sprintf("%s-%s", name, now)),
		NoReboot:    aws.Bool(true),
	})
	if err != nil {
		return "", err
	}

	imageID := *result.ImageId

	if err := client.svcEC2.WaitUntilImageAvailableWithContext(
		ctx,
		&ec2.DescribeImagesInput{
			ImageIds: []*string{aws.String(imageID)},
		},
		[]request.WaiterOption{request.WithWaiterMaxAttempts(120)}...,
	); err != nil {
		return "", err
	}

	return imageID, nil
}

// CreateTags creates tags to specified resource id.
func (client *AWSClient) CreateTags(ctx context.Context, resourceID string, tags []*ec2.Tag) error {
	_, err := client.svcEC2.CreateTagsWithContext(ctx, &ec2.CreateTagsInput{
		Resources: []*string{aws.String(resourceID)},
		Tags:      tags,
	})
	if err != nil {
		return err
	}

	// check for create tag complete
	doneCh := make(chan struct{})
	go func() {
		for i := 0; i < 40; i++ {
			if strings.HasPrefix(resourceID, "ami-") {
				result, err := client.svcEC2.DescribeImages(&ec2.DescribeImagesInput{
					ImageIds: []*string{aws.String(resourceID)},
				})
				if err != nil {
					continue
				}
				if len(result.Images[0].Tags) == 3 {
					doneCh <- struct{}{}
					return
				}
				time.Sleep(1 * time.Second)
			} else if strings.HasPrefix(resourceID, "snap-") {
				result, err := client.svcEC2.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
					SnapshotIds: []*string{aws.String(resourceID)},
				})
				if err != nil {
					continue
				}
				if len(result.Snapshots[0].Tags) == 3 {
					doneCh <- struct{}{}
					return
				}
				time.Sleep(1 * time.Second)
			}
		}
	}()
	<-doneCh

	return nil
}

// GetImages return machine images with the specified tag values.
func (client *AWSClient) GetImages(ctx context.Context, name, service string) ([]*ec2.Image, error) {
	result, err := client.svcEC2.DescribeImagesWithContext(ctx, &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("tag:BackupType"), Values: []*string{aws.String("auto")}},
			{Name: aws.String("tag:Name"), Values: []*string{aws.String(name)}},
			{Name: aws.String("tag:Service"), Values: []*string{aws.String(service)}},
		},
	})
	if err != nil {
		return nil, err
	}

	return result.Images, nil
}

// GetImage returns image with the specified image id.
func (client *AWSClient) GetImage(ctx context.Context, imageID string) (*ec2.Image, error) {
	result, err := client.svcEC2.DescribeImagesWithContext(ctx, &ec2.DescribeImagesInput{
		ImageIds: []*string{aws.String(imageID)},
	})
	if err != nil {
		return nil, err
	}

	if len(result.Images) < 1 {
		return nil, fmt.Errorf("can't find image: %s", imageID)
	}

	return result.Images[0], nil
}

// GetSnapshots returns snapshot ids which machine image id related.
func (client *AWSClient) GetSnapshots(ctx context.Context, imageID string) ([]string, error) {
	result, err := client.svcEC2.DescribeImagesWithContext(ctx, &ec2.DescribeImagesInput{
		ImageIds: []*string{aws.String(imageID)},
	})
	if err != nil {
		return nil, err
	}

	var snapshots []string
	for _, b := range result.Images[0].BlockDeviceMappings {
		snapshots = append(snapshots, *b.Ebs.SnapshotId)
	}

	return snapshots, nil
}

// DeregisterImages deregister machine images and related snapshots.
func (client *AWSClient) DeregisterImages(ctx context.Context, images []*ec2.Image) error {
	for _, image := range images {
		client.svcEC2.DeregisterImageWithContext(ctx, &ec2.DeregisterImageInput{
			ImageId: image.ImageId,
		})
		for _, d := range image.BlockDeviceMappings {
			client.svcEC2.DeleteSnapshot(&ec2.DeleteSnapshotInput{
				SnapshotId: d.Ebs.SnapshotId,
			})
		}
	}
	return nil
}

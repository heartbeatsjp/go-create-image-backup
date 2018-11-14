package main

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/mock/gomock"
	"github.com/heartbeatsjp/go-create-image-backup/mock"
)

func TestGetRegion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2Metadata := mock.NewMockEC2MetadataAPI(mockCtrl)
	mockEC2Metadata.EXPECT().Region().Return("ap-northeast-1", nil)
	mockEC2Metadata.EXPECT().Available().Return(true)

	client := AWSClient{
		svcEC2Metadata: mockEC2Metadata,
	}

	got, err := client.GetRegion()
	if err != nil {
		t.Fatal("Region failed: ", err)
	}

	want := "ap-northeast-1"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestGetRegion_EC2Metadata_Unavailable(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2Metadata := mock.NewMockEC2MetadataAPI(mockCtrl)
	mockEC2Metadata.EXPECT().Available().Return(false)

	client := AWSClient{
		svcEC2Metadata: mockEC2Metadata,
	}

	_, got := client.GetRegion()

	want := "program is not running with EC2 Instance or metadata service is not available"
	if got.Error() != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestGetInstanceID(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2Metadata := mock.NewMockEC2MetadataAPI(mockCtrl)
	mockEC2Metadata.EXPECT().GetInstanceIdentityDocument().Return(ec2metadata.EC2InstanceIdentityDocument{
		InstanceID: "i-1234567890abcdef0",
	}, nil)
	mockEC2Metadata.EXPECT().Available().Return(true)

	client := AWSClient{
		svcEC2Metadata: mockEC2Metadata,
	}

	got, err := client.GetInstanceID()
	if err != nil {
		t.Fatal("GetInstanceID failed: ", err)
	}

	want := "i-1234567890abcdef0"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestGetInstanceID_EC2Metadata_Unavailable(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2Metadata := mock.NewMockEC2MetadataAPI(mockCtrl)
	mockEC2Metadata.EXPECT().Available().Return(false)

	client := AWSClient{
		svcEC2Metadata: mockEC2Metadata,
	}

	_, got := client.GetInstanceID()

	want := "program is not running with EC2 Instance or metadata service is not available"
	if got.Error() != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestGetInstanceName(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock.NewMockEC2API(mockCtrl)
	mockEC2.EXPECT().DescribeTagsWithContext(
		context.TODO(),
		&ec2.DescribeTagsInput{
			Filters: []*ec2.Filter{
				{Name: aws.String("resource-id"), Values: []*string{aws.String("ami-1234567890abcdef0")}},
				{Name: aws.String("tag-key"), Values: []*string{aws.String("Name")}},
			},
		}).Return(&ec2.DescribeTagsOutput{
		Tags: []*ec2.TagDescription{
			{Key: aws.String("testkey"), Value: aws.String("testvalue")},
		},
	}, nil)

	client := AWSClient{
		svcEC2: mockEC2,
	}

	got, err := client.GetInstanceName(context.TODO(), "ami-1234567890abcdef0")
	if err != nil {
		t.Fatal("GetInstance Name failed: ", err)
	}

	want := "testvalue"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestGetInstanceName_Tag_NotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock.NewMockEC2API(mockCtrl)
	mockEC2.EXPECT().DescribeTagsWithContext(
		context.TODO(),
		&ec2.DescribeTagsInput{
			Filters: []*ec2.Filter{
				{Name: aws.String("resource-id"), Values: []*string{aws.String("i-1234567890abcdef0")}},
				{Name: aws.String("tag-key"), Values: []*string{aws.String("Name")}},
			},
		}).Return(&ec2.DescribeTagsOutput{}, nil)

	client := AWSClient{
		svcEC2: mockEC2,
	}

	got, err := client.GetInstanceName(context.TODO(), "i-1234567890abcdef0")
	if err != nil {
		t.Fatal("GetInstance Name failed: ", err)
	}

	want := "i-1234567890abcdef0"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestCreateImage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock.NewMockEC2API(mockCtrl)
	mockEC2.EXPECT().CreateImageWithContext(
		context.TODO(),
		&ec2.CreateImageInput{
			InstanceId:  aws.String("i-1234567890abcdef0"),
			Description: aws.String("create by go-create-image-backup"),
			Name:        aws.String("test-200601021504"),
			NoReboot:    aws.Bool(true),
		}).Return(&ec2.CreateImageOutput{
		ImageId: aws.String("ami-1234567890abcdef0"),
	}, nil)
	mockEC2.EXPECT().WaitUntilImageAvailableWithContext(
		context.TODO(),
		&ec2.DescribeImagesInput{
			ImageIds: []*string{aws.String("ami-1234567890abcdef0")},
		},
		gomock.Any(),
	).Return(nil)

	client := AWSClient{
		svcEC2: mockEC2,
	}

	got, err := client.CreateImage(context.TODO(), "i-1234567890abcdef0", "test", "200601021504")
	if err != nil {
		t.Fatal("CreateImage failed: ", err)
	}

	want := "ami-1234567890abcdef0"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestCreateTags_With_AMI(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock.NewMockEC2API(mockCtrl)
	mockEC2.EXPECT().CreateTagsWithContext(
		context.TODO(),
		&ec2.CreateTagsInput{
			Resources: []*string{aws.String("ami-1234567890abcdef0")},
			Tags: []*ec2.Tag{
				{Key: aws.String("key1"), Value: aws.String("value1")},
				{Key: aws.String("key2"), Value: aws.String("value2")},
				{Key: aws.String("key3"), Value: aws.String("value3")},
			},
		}).Return(nil, nil)
	mockEC2.EXPECT().DescribeImages(
		&ec2.DescribeImagesInput{
			ImageIds: []*string{aws.String("ami-1234567890abcdef0")},
		}).Return(&ec2.DescribeImagesOutput{
		Images: []*ec2.Image{
			{
				Tags: []*ec2.Tag{
					{Key: aws.String("key1"), Value: aws.String("value1")},
					{Key: aws.String("key2"), Value: aws.String("value2")},
					{Key: aws.String("key3"), Value: aws.String("value3")},
				},
			},
		},
	}, nil)

	client := AWSClient{
		svcEC2: mockEC2,
	}

	tag := []*ec2.Tag{
		{Key: aws.String("key1"), Value: aws.String("value1")},
		{Key: aws.String("key2"), Value: aws.String("value2")},
		{Key: aws.String("key3"), Value: aws.String("value3")},
	}
	if err := client.CreateTags(context.TODO(), "ami-1234567890abcdef0", tag); err != nil {
		t.Fatal("CreateTags failed: ", err)
	}
}

func TestCreateTags_With_Snapshot(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock.NewMockEC2API(mockCtrl)
	mockEC2.EXPECT().CreateTagsWithContext(
		context.TODO(),
		&ec2.CreateTagsInput{
			Resources: []*string{aws.String("snap-1234567890abcdef0")},
			Tags: []*ec2.Tag{
				{Key: aws.String("key1"), Value: aws.String("value1")},
				{Key: aws.String("key2"), Value: aws.String("value2")},
				{Key: aws.String("key3"), Value: aws.String("value3")},
			},
		}).Return(nil, nil)
	mockEC2.EXPECT().DescribeSnapshots(&ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{aws.String("snap-1234567890abcdef0")},
	}).Return(&ec2.DescribeSnapshotsOutput{
		Snapshots: []*ec2.Snapshot{
			{
				Tags: []*ec2.Tag{
					{Key: aws.String("key1"), Value: aws.String("value1")},
					{Key: aws.String("key2"), Value: aws.String("value2")},
					{Key: aws.String("key3"), Value: aws.String("value3")},
				},
			},
		},
	}, nil)

	client := AWSClient{
		svcEC2: mockEC2,
	}

	tag := []*ec2.Tag{
		{Key: aws.String("key1"), Value: aws.String("value1")},
		{Key: aws.String("key2"), Value: aws.String("value2")},
		{Key: aws.String("key3"), Value: aws.String("value3")},
	}
	if err := client.CreateTags(context.TODO(), "snap-1234567890abcdef0", tag); err != nil {
		t.Fatal("CreateTags failed: ", err)
	}
}

func TestGetImages(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock.NewMockEC2API(mockCtrl)
	mockEC2.EXPECT().DescribeImagesWithContext(
		context.TODO(),
		&ec2.DescribeImagesInput{
			Filters: []*ec2.Filter{
				{Name: aws.String("tag:BackupType"), Values: []*string{aws.String("auto")}},
				{Name: aws.String("tag:Name"), Values: []*string{aws.String("test")}},
				{Name: aws.String("tag:Service"), Values: []*string{aws.String("service")}},
			},
		}).Return(&ec2.DescribeImagesOutput{
		Images: []*ec2.Image{},
	}, nil)

	client := AWSClient{
		svcEC2: mockEC2,
	}

	_, err := client.GetImages(context.TODO(), "test", "service")
	if err != nil {
		t.Fatal("GetImages failed: ", err)
	}
}

func TestGetImage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock.NewMockEC2API(mockCtrl)
	mockEC2.EXPECT().DescribeImagesWithContext(
		context.TODO(),
		&ec2.DescribeImagesInput{
			ImageIds: []*string{aws.String("ami-1234567890abcdef0")},
		}).Return(&ec2.DescribeImagesOutput{
		Images: []*ec2.Image{
			{ImageId: aws.String("ami-1234567890abcdef0")},
		},
	}, nil)

	client := AWSClient{
		svcEC2: mockEC2,
	}

	_, err := client.GetImage(context.TODO(), "ami-1234567890abcdef0")
	if err != nil {
		t.Fatal("GetImage failed: ", err)
	}
}

func TestGetImage_Image_NotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock.NewMockEC2API(mockCtrl)
	mockEC2.EXPECT().DescribeImagesWithContext(
		context.TODO(),
		&ec2.DescribeImagesInput{
			ImageIds: []*string{aws.String("ami-1234567890abcdef0")},
		}).Return(&ec2.DescribeImagesOutput{
		Images: []*ec2.Image{},
	}, nil)

	client := AWSClient{
		svcEC2: mockEC2,
	}

	_, got := client.GetImage(context.TODO(), "ami-1234567890abcdef0")

	want := "can't find image: ami-1234567890abcdef0"
	if got.Error() != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestGetSnapshots(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock.NewMockEC2API(mockCtrl)
	mockEC2.EXPECT().DescribeImagesWithContext(
		context.TODO(),
		&ec2.DescribeImagesInput{
			ImageIds: []*string{aws.String("ami-1234567890abcdef0")},
		}).Return(&ec2.DescribeImagesOutput{
		Images: []*ec2.Image{
			{
				BlockDeviceMappings: []*ec2.BlockDeviceMapping{
					{
						Ebs: &ec2.EbsBlockDevice{
							SnapshotId: aws.String("snap-1234567890abcdef0"),
						},
					},
				},
			},
		},
	}, nil)

	client := AWSClient{
		svcEC2: mockEC2,
	}

	s, err := client.GetSnapshots(context.TODO(), "ami-1234567890abcdef0")
	if err != nil {
		t.Fatal("GetSnapshots: ", err)
	}

	got := len(s)
	want := 1
	if got != want {
		t.Fatalf("got %d, want %d", got, want)
	}
}

func TestGetSnapshots_Multi(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock.NewMockEC2API(mockCtrl)
	mockEC2.EXPECT().DescribeImagesWithContext(
		context.TODO(),
		&ec2.DescribeImagesInput{
			ImageIds: []*string{aws.String("ami-1234567890abcdef0")},
		}).Return(&ec2.DescribeImagesOutput{
		Images: []*ec2.Image{
			{
				BlockDeviceMappings: []*ec2.BlockDeviceMapping{
					{
						Ebs: &ec2.EbsBlockDevice{
							SnapshotId: aws.String("snap-1234567890abcdef0"),
						},
					},
					{
						Ebs: &ec2.EbsBlockDevice{
							SnapshotId: aws.String("snap-1234567890abcdef1"),
						},
					},
				},
			},
		},
	}, nil)

	client := AWSClient{
		svcEC2: mockEC2,
	}

	s, err := client.GetSnapshots(context.TODO(), "ami-1234567890abcdef0")
	if err != nil {
		t.Fatal("GetSnapshots: ", err)
	}

	got := len(s)
	want := 2
	if got != want {
		t.Fatalf("got %d, want %d", got, want)
	}
}

func TestDeregisterImages(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock.NewMockEC2API(mockCtrl)
	mockEC2.EXPECT().DeregisterImageWithContext(
		context.TODO(),
		&ec2.DeregisterImageInput{
			ImageId: aws.String("ami-1234567890abcdef0"),
		}).Return(&ec2.DeregisterImageOutput{}, nil)
	mockEC2.EXPECT().DeleteSnapshot(&ec2.DeleteSnapshotInput{
		SnapshotId: aws.String("snap-1234567890abcdef0"),
	}).Return(&ec2.DeleteSnapshotOutput{}, nil)

	client := AWSClient{
		svcEC2: mockEC2,
	}

	i := []*ec2.Image{
		{
			ImageId: aws.String("ami-1234567890abcdef0"),
			BlockDeviceMappings: []*ec2.BlockDeviceMapping{
				{Ebs: &ec2.EbsBlockDevice{SnapshotId: aws.String("snap-1234567890abcdef0")}},
			},
		},
	}
	if err := client.DeregisterImages(context.TODO(), i); err != nil {
		t.Fatal("DeregisterImages failed: ", err)
	}
}

func TestDeregisterImagesUseEphemeralDisk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockEC2 := mock.NewMockEC2API(mockCtrl)
	mockEC2.EXPECT().DeregisterImageWithContext(
		context.TODO(),
		&ec2.DeregisterImageInput{
			ImageId: aws.String("ami-1234567890abcdef0"),
		}).Return(&ec2.DeregisterImageOutput{}, nil)
	mockEC2.EXPECT().DeleteSnapshot(&ec2.DeleteSnapshotInput{
		SnapshotId: aws.String("snap-1234567890abcdef0"),
	}).Return(&ec2.DeleteSnapshotOutput{}, nil)

	client := AWSClient{
		svcEC2: mockEC2,
	}

	i := []*ec2.Image{
		{
			ImageId: aws.String("ami-1234567890abcdef0"),
			BlockDeviceMappings: []*ec2.BlockDeviceMapping{
				{
					DeviceName: aws.String("/dev/sda"),
					Ebs:        &ec2.EbsBlockDevice{SnapshotId: aws.String("snap-1234567890abcdef0")},
				},
				{
					DeviceName: aws.String("/dev/ephemeral0"),
				},
			},
		},
	}
	if err := client.DeregisterImages(context.TODO(), i); err != nil {
		t.Fatal("DeregisterImages failed: ", err)
	}
}

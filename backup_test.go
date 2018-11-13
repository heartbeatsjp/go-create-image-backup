package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/mock/gomock"
	"github.com/heartbeatsjp/go-create-image-backup/mock"
)

func TestCreate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockAWSClient := mock.NewMockAWS(mockCtrl)
	mockAWSClient.EXPECT().CreateImage(
		context.TODO(),
		"i-1234567890abcdef0",
		"test",
		gomock.Any()).Return("ami-1234567890abcdef0", nil)
	createAMITag := mockAWSClient.EXPECT().CreateTags(
		context.TODO(),
		"ami-1234567890abcdef0",
		[]*ec2.Tag{
			{Key: aws.String("BackupType"), Value: aws.String("auto")},
			{Key: aws.String("Name"), Value: aws.String("test")},
			{Key: aws.String("Service"), Value: aws.String("service")},
		}).Return(nil)
	mockAWSClient.EXPECT().GetSnapshots(context.TODO(), "ami-1234567890abcdef0").Return(
		[]string{"snap-1234567890abcdef0", "snap-1234567890abcdef1", "snap-1234567890abcdef2"},
		nil)
	createSnapTag1 := mockAWSClient.EXPECT().CreateTags(
		context.TODO(),
		"snap-1234567890abcdef0",
		[]*ec2.Tag{
			{Key: aws.String("BackupType"), Value: aws.String("auto")},
			{Key: aws.String("Name"), Value: aws.String("test")},
			{Key: aws.String("Service"), Value: aws.String("service")},
		}).Return(nil).After(createAMITag)
	createSnapTag2 := mockAWSClient.EXPECT().CreateTags(
		context.TODO(),
		"snap-1234567890abcdef1",
		[]*ec2.Tag{
			{Key: aws.String("BackupType"), Value: aws.String("auto")},
			{Key: aws.String("Name"), Value: aws.String("test")},
			{Key: aws.String("Service"), Value: aws.String("service")},
		}).Return(nil).After(createSnapTag1)
	mockAWSClient.EXPECT().CreateTags(
		context.TODO(),
		"snap-1234567890abcdef2",
		[]*ec2.Tag{
			{Key: aws.String("BackupType"), Value: aws.String("auto")},
			{Key: aws.String("Name"), Value: aws.String("test")},
			{Key: aws.String("Service"), Value: aws.String("service")},
		}).Return(nil).After(createSnapTag2)

	backup := &Backup{
		InstanceID: "i-1234567890abcdef0",
		Name:       "test",
		Service:    "service",
		Client:     mockAWSClient,
	}

	got, err := backup.Create(context.TODO())
	if err != nil {
		t.Fatal("Create failed: ", err)
	}

	want := "ami-1234567890abcdef0"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestRotate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockAWSClient := mock.NewMockAWS(mockCtrl)
	mockAWSClient.EXPECT().GetImages(context.TODO(), "test", "service").Return([]*ec2.Image{
		{ImageId: aws.String("ami-1234567890abcdef0"), CreationDate: aws.String("2006-01-02T15:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef1"), CreationDate: aws.String("2006-01-02T16:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef2"), CreationDate: aws.String("2006-01-02T17:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef3"), CreationDate: aws.String("2006-01-02T18:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef4"), CreationDate: aws.String("2006-01-02T19:04:05.000Z"), State: aws.String("available")},
	}, nil)
	mockAWSClient.EXPECT().DeregisterImages(context.TODO(), []*ec2.Image{
		{ImageId: aws.String("ami-1234567890abcdef0"), CreationDate: aws.String("2006-01-02T15:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef1"), CreationDate: aws.String("2006-01-02T16:04:05.000Z"), State: aws.String("available")},
	}).Return(nil)

	backup := &Backup{
		Name:       "test",
		Service:    "service",
		Generation: 3,
		Client:     mockAWSClient,
	}

	got, err := backup.Rotate(context.TODO(), "ami-1234567890abcdef4")
	if err != nil {
		t.Fatal("Rotate failed: ", err)
	}

	want := []string{"ami-1234567890abcdef0", "ami-1234567890abcdef1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestRotate_NotRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockAWSClient := mock.NewMockAWS(mockCtrl)
	mockAWSClient.EXPECT().GetImages(context.TODO(), "test", "service").Return([]*ec2.Image{
		{ImageId: aws.String("ami-1234567890abcdef0"), CreationDate: aws.String("2006-01-02T15:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef1"), CreationDate: aws.String("2006-01-02T16:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef2"), CreationDate: aws.String("2006-01-02T17:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef3"), CreationDate: aws.String("2006-01-02T18:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef4"), CreationDate: aws.String("2006-01-02T19:04:05.000Z"), State: aws.String("available")},
	}, nil)

	backup := &Backup{
		Name:       "test",
		Service:    "service",
		Generation: 5,
		Client:     mockAWSClient,
	}

	got, err := backup.Rotate(context.TODO(), "ami-1234567890abcdef4")
	if err != nil {
		t.Fatal("Rotate failed: ", err)
	}

	var want []string
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestRotate_RecentlyImageID_NotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockAWSClient := mock.NewMockAWS(mockCtrl)
	mockAWSClient.EXPECT().GetImages(context.TODO(), "test", "service").Return([]*ec2.Image{
		{ImageId: aws.String("ami-1234567890abcdef0"), CreationDate: aws.String("2006-01-02T15:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef1"), CreationDate: aws.String("2006-01-02T16:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef2"), CreationDate: aws.String("2006-01-02T17:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef3"), CreationDate: aws.String("2006-01-02T18:04:05.000Z"), State: aws.String("available")},
	}, nil)
	mockAWSClient.EXPECT().GetImage(context.TODO(), "ami-1234567890abcdef4").Return(&ec2.Image{
		ImageId: aws.String("ami-1234567890abcdef4"), CreationDate: aws.String("2006-01-02T19:04:05.000Z"), State: aws.String("available"),
	}, nil)
	mockAWSClient.EXPECT().DeregisterImages(context.TODO(), []*ec2.Image{
		{ImageId: aws.String("ami-1234567890abcdef0"), CreationDate: aws.String("2006-01-02T15:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef1"), CreationDate: aws.String("2006-01-02T16:04:05.000Z"), State: aws.String("available")},
	}).Return(nil)

	backup := &Backup{
		Name:       "test",
		Service:    "service",
		Generation: 3,
		Client:     mockAWSClient,
	}

	got, err := backup.Rotate(context.TODO(), "ami-1234567890abcdef4")
	if err != nil {
		t.Fatal("Rotate failed: ", err)
	}

	want := []string{"ami-1234567890abcdef0", "ami-1234567890abcdef1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestRotate_FailedImage_Found(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockAWSClient := mock.NewMockAWS(mockCtrl)
	mockAWSClient.EXPECT().GetImages(context.TODO(), "test", "service").Return([]*ec2.Image{
		{ImageId: aws.String("ami-1234567890abcdef0"), CreationDate: aws.String("2006-01-02T15:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef1"), CreationDate: aws.String("2006-01-02T16:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef2"), CreationDate: aws.String("2006-01-02T17:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef3"), CreationDate: aws.String("2006-01-02T18:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef4"), CreationDate: aws.String("2006-01-02T19:04:05.000Z"), State: aws.String("failed")},
	}, nil)
	mockAWSClient.EXPECT().DeregisterImages(context.TODO(), []*ec2.Image{
		{ImageId: aws.String("ami-1234567890abcdef4"), CreationDate: aws.String("1970-01-01T00:00:00.000Z"), State: aws.String("failed")},
		{ImageId: aws.String("ami-1234567890abcdef0"), CreationDate: aws.String("2006-01-02T15:04:05.000Z"), State: aws.String("available")},
	}).Return(nil)

	backup := &Backup{
		Name:       "test",
		Service:    "service",
		Generation: 3,
		Client:     mockAWSClient,
	}

	got, err := backup.Rotate(context.TODO(), "ami-1234567890abcdef4")
	if err != nil {
		t.Fatal("Rotate failed: ", err)
	}

	want := []string{"ami-1234567890abcdef4", "ami-1234567890abcdef0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestRotate_RecentlyImageID_Pending(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockAWSClient := mock.NewMockAWS(mockCtrl)
	mockAWSClient.EXPECT().GetImages(context.TODO(), "test", "service").Return([]*ec2.Image{
		{ImageId: aws.String("ami-1234567890abcdef0"), CreationDate: aws.String("2006-01-02T15:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef1"), CreationDate: aws.String("2006-01-02T16:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef2"), CreationDate: aws.String("2006-01-02T17:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef3"), CreationDate: aws.String("2006-01-02T18:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef4"), State: aws.String("pending")},
	}, nil)
	mockAWSClient.EXPECT().DeregisterImages(context.TODO(), []*ec2.Image{
		{ImageId: aws.String("ami-1234567890abcdef0"), CreationDate: aws.String("2006-01-02T15:04:05.000Z"), State: aws.String("available")},
		{ImageId: aws.String("ami-1234567890abcdef1"), CreationDate: aws.String("2006-01-02T16:04:05.000Z"), State: aws.String("available")},
	}).Return(nil)

	backup := &Backup{
		Name:       "test",
		Service:    "service",
		Generation: 3,
		Client:     mockAWSClient,
	}

	got, err := backup.Rotate(context.TODO(), "ami-1234567890abcdef4")
	if err != nil {
		t.Fatal("Rotate failed: ", err)
	}

	want := []string{"ami-1234567890abcdef0", "ami-1234567890abcdef1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

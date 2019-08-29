package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Backup provides methods for backup operations.
type Backup struct {
	InstanceID string
	Name       string
	Generation int
	Service    string
	CustomTags []Tag
	Client     AWS
}

// Tag is key-value formatted metadata for backup
type Tag struct {
	Key   string
	Value string
}

// Create Amazon Machine Image(AMI) as instance's backup.
func (b *Backup) Create(ctx context.Context) (string, error) {
	const layout = "200601021504"
	now := time.Now().Format(layout)

	imageID, err := b.Client.CreateImage(ctx, b.InstanceID, b.Name, now)
	if err != nil {
		return "", err
	}

	tag := []*ec2.Tag{
		{
			Key:   aws.String("BackupType"),
			Value: aws.String("auto"),
		},
		{
			Key:   aws.String("Name"),
			Value: aws.String(b.Name),
		},
		{
			Key:   aws.String("Service"),
			Value: aws.String(b.Service),
		},
	}

	if len(b.CustomTags) > 0 {
		var customTags []*ec2.Tag
		for _, t := range b.CustomTags {
			customTags = append(customTags, &ec2.Tag{
				Key:   aws.String(t.Key),
				Value: aws.String(t.Value),
			})
		}
		tag = append(tag, customTags...)
	}

	if err := b.Client.CreateTags(ctx, imageID, tag); err != nil {
		return "", err
	}

	snapshots, err := b.Client.GetSnapshots(ctx, imageID)
	if err != nil {
		return "", err
	}

	var errList []string
	for _, snapshot := range snapshots {
		if err := b.Client.CreateTags(ctx, snapshot, tag); err != nil {
			errList = append(errList, err.Error())
		}
	}
	if len(errList) > 0 {
		return "", fmt.Errorf(strings.Join(errList, ", "))
	}

	return imageID, nil
}

func convertDate(baseStr string) time.Time {
	dateStr := strings.Split(baseStr, ".")[0]
	layout := "2006-01-02T15:04:05"
	t, err := time.Parse(layout, dateStr)
	if err != nil {
		return time.Time{}
	}
	return t
}

// Rotate deregisters of old machine image which greater than generation.
func (b *Backup) Rotate(ctx context.Context, recentlyImageID string) ([]string, error) {
	var rotateImageIDs []string

	images, err := b.Client.GetImages(ctx, b.Name, b.Service)
	if err != nil {
		return rotateImageIDs, err
	}

	var hasRecentlyImageID bool
	for _, i := range images {
		if *i.ImageId == recentlyImageID {
			hasRecentlyImageID = true
		}
	}

	if !hasRecentlyImageID {
		recentlyImage, err := b.Client.GetImage(ctx, recentlyImageID)
		if err != nil {
			return rotateImageIDs, err
		}
		images = append(images, recentlyImage)
	}

	if len(images) <= b.Generation {
		return rotateImageIDs, nil
	}

	for _, image := range images {
		if *image.State == "failed" {
			image.CreationDate = aws.String("1970-01-01T00:00:00.000Z")
		}
		if *image.State == "pending" {
			const layout = "2006-01-02T15:04:05.000Z"
			now := time.Now().Format(layout)
			image.CreationDate = aws.String(now)
		}
	}

	sort.Slice(images, func(i, j int) bool {
		iDate := convertDate(*images[i].CreationDate)
		jDate := convertDate(*images[j].CreationDate)
		return iDate.Before(jDate)
	})

	rotateIndex := len(images) - b.Generation
	if err := b.Client.DeregisterImages(ctx, images[:rotateIndex]); err != nil {
		return rotateImageIDs, err
	}
	for _, i := range images[:rotateIndex] {
		rotateImageIDs = append(rotateImageIDs, *i.ImageId)
	}

	return rotateImageIDs, err
}

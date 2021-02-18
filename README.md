# go-create-image-backup

[![wercker status](https://app.wercker.com/status/e49ed2149efc24b7a997fd6ee35578bb/s/master "wercker status")](https://app.wercker.com/project/byKey/e49ed2149efc24b7a997fd6ee35578bb)
[![Go Report Card](https://goreportcard.com/badge/github.com/heartbeatsjp/go-create-image-backup)](https://goreportcard.com/report/github.com/heartbeatsjp/go-create-image-backup)
[![Not Maintained](https://img.shields.io/badge/Maintenance%20Level-Abandoned-orange.svg)](https://gist.github.com/cheerfulstoic/d107229326a01ff0f333a1d3476e068d)

**Note: This repository is no longer maintained.**  
**Use [AWS Backup](https://aws.amazon.com/backup/).**

A backup tool with AWS Amazon machine image(AMI) written by Go.  


## Example

Simple usage is below. That creates new machine image `ami-1234567890abcdef0` as a backup of EC2 instance that id is `i-1234567890abcdef0`.  
Also, that deregisters machine image `ami-1234567890abcdef1`, this is backup rotate that deregisters machine images exceeded the number of backup generation specified by `-backup-generation` option that default is 10.  

```
$ go-create-image-backup -instance-id i-1234567890abcdef0
create image: ami-1234567890abcdef0
deregister images: ami-1234567890abcdef1
```


## Features

- Create a backup for Amazon EC2 instance by Amazon machine image
- Manage backup generations per service tag-based logical group
- Add custom tags to AMI and EBS Snapshots
- Notify error by email


### Create a backup for Amazon EC2 instance by Amazon machine image

A primary feature of `go-create-image-backup`.  


### Manage backup generations per service tag-based logical group

`go-create-image-backup` can one or more generation management of backup to a single instance by service tag.  
It's useful when backups for different scheduling like daily and weekly.  

The following commands are backup for the same instance. However, backup rotate is each independent generation management.  

```
# daily backup
0 4 * * * go-create-image-backup -instance-id i-1234567890abcdef0 -service-tag daily -backup-generation 7

# weekly backup
0 4 1 * * go-create-image-backup -instance-id i-1234567890abcdef0 -service-tag weekly -backup-generation 4
```


### Add custom tags to AMI and EBS Snapshots

Custom tags are the feature of add any tags to AMI and EBS Snapshots that related to backup.  
`go-create-image-backup` has `-custom-tags` option that  

```
$ go-create-image-backup -instance-id i-1234567890abcdef0 -service-tag daily -custom-tags key1:val1,key2:val2,...
```

In the above case, add the following tags to AMI of `i-1234567890abcdef0` and EBS Snapshots related that AMI.  

|Key|Value|
|---|---|
|key1|val1|
|key2|val2|

IMPORTANT NOTICE:  

Custom tags are not effecting to generation management of backup.  


### Notify error by email

IMPORTANT NOTICE:  

You should be careful when sending email from Amazon EC2 instance, See also [AWS Documentation](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/limits.html#limits-ec2).  


### IAM requirements

You need to create and use policy which has permissions to `go-create-image-backup` can use following AWS APIs.  

- CreateImage
- CreateTags
- DeleteSnapshot
- DeregisterImage
- DescribeImages
- DescribeSnapshots
- DescribeTags


## Options

```
(-backup-generation | -g) int
 number of backup generation (default 10)
(-instance-id | -i) string
 instance id
(-region | -r) string
 region
(-service-tag | -s) string
 value of Service tag
(-custom-tags | -c) key1:val1,key2:val2,...
 value of Cunstom tags
(-mail-from | -f) string
 from-address of email notification
(-mail-to | -t) string
 to-address of email notification
(-mail-server | -m) string
 mail server address (default localhost)
(-port | -p) int
 mail server's port (default 25)
(-version | -v)
 print version information
```


## Author

[Takatada Yoshima](https://github.com/shiimaxx)  


## License

[Apache License 2.0](https://github.com/heartbeatsjp/go-create-image-backup/blob/master/LICENSE)  

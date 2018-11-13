# go-create-image-backup

backup tool with AWS Amazon machine image(AMI) written by Go.

## Features

- Create backup for Amazon EC2 instance by Amazon machine image
- Manage backup generations per logical group(service tag based).
- Notify error by email
    - You should be careful when send email from Amazon EC2 instance, See also [AWS Documentation](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/limits.html#limits-ec2)

## Usages

```
Usage of go-create-image-backup:
  -backup-generation int
    	number of backup generation (default 10)
  -g int
    	number of backup generation(Short) (default 10)
  -i string
    	instance id(Short)
  -instance-id string
    	instance id
  -r string
    	region(Short)
  -region string
    	region
  -s string
    	value of Service tag(Short)
  -service-tag string
    	value of Service tag
  -t string
      set your mail address , if you want e-mail notification when backup failure(Short)
  -mail-to
      set your mail address , if you want e-mail notification when backup failure
  -m string
      mail server address (default localhost)(Short)
  -mail-server
      mail server address (default localhost)
  -p int
      mail server's port (default 25)(Short)
  -port int
      mail server's port (default 25)
  -v	print version information(Short)
  -version
    	print version information
```

## Author

[Takatada Yoshima](https://github.com/shiimaxx)

## License

[Apache License 2.0](https://github.com/heartbeatsjp/go-create-image-backup/blob/master/LICENSE)

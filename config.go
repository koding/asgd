package asgd

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/koding/ec2dynamicdata"
	"github.com/koding/multiconfig"
)

// Config holds configuration parameters for asgd
type Config struct {
	Name    string `required:"true"`
	Session *awssession.Session

	// required
	AccessKeyID     string `required:"true"`
	SecretAccessKey string `required:"true"`

	// can be overriden
	Region          string
	AutoScalingName string

	// optional
	Debug bool
}

// Configure prepares configuration data for tunnelproxy manager
func Configure() (*Config, error) {
	c := &Config{}

	mc := multiconfig.New()
	mc.Loader = multiconfig.MultiLoader(
		&multiconfig.TagLoader{},
		&multiconfig.EnvironmentLoader{},
		&multiconfig.EnvironmentLoader{Prefix: "ASGD"},
		&multiconfig.FlagLoader{},
	)

	mc.MustLoad(c)

	// decide on region name
	region, err := getRegion(c)
	if err != nil {
		return nil, err
	}

	c.Region = region

	c.Session = awssession.New(&aws.Config{
		Credentials: credentials.NewStaticCredentials(
			c.AccessKeyID,
			c.SecretAccessKey,
			"",
		),
		Region:     aws.String(c.Region),
		MaxRetries: aws.Int(5),
	})

	// decide on autoscaling name
	name, err := getAutoScalingName(c, c.Session)
	if err != nil {
		return nil, err
	}

	c.AutoScalingName = name
	return c, nil
}

// getRegion checks if region name is given in config, if not tries to get it
// from ec2dynamicdata endpoint
func getRegion(conf *Config) (string, error) {
	if conf.Region != "" {
		return conf.Region, nil
	}

	info, err := ec2dynamicdata.Get()
	if err != nil {
		return "", fmt.Errorf("couldn't get region. Err: %s", err.Error())
	}

	if info.Region == "" {
		return "", fmt.Errorf("malformed ec2dynamicdata response: %#v", info)
	}
	return info.Region, nil
}

// getAutoScalingName tries to get autoscaling name from system, first gets from
// config var, if not set then tries ec2dynamicdata service
func getAutoScalingName(conf *Config, session *awssession.Session) (string, error) {
	if conf.AutoScalingName != "" {
		return conf.AutoScalingName, nil
	}

	info, err := ec2dynamicdata.Get()
	if err != nil {
		return "", fmt.Errorf("couldn't get info. Err: %s", err.Error())
	}

	instanceID := info.InstanceID

	asg := autoscaling.New(session)

	resp, err := asg.DescribeAutoScalingInstances(
		&autoscaling.DescribeAutoScalingInstancesInput{
			InstanceIds: []*string{
				aws.String(instanceID),
			},
		},
	)
	if err != nil {
		return "", err
	}

	for _, instance := range resp.AutoScalingInstances {
		if *instance.InstanceId == instanceID {
			return *instance.AutoScalingGroupName, nil
		}
	}
	return "", errors.New("couldn't find autoscaling name")
}

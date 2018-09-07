package rhcos

import (
	"fmt"
)

// AMI calculates a Red Hat CoreOS AMI.
func AMI(channel, region string) (ami string, err error) {
	if channel != "tested" {
		return "", fmt.Errorf("channel %q is not yet supported", channel)
	}

	if region != "us-east-1" {
		return "", fmt.Errorf("region %q is not yet supported", region)
	}

	return "ami-07307c397daf4d02e", nil
}

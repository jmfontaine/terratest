package test

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	terratest_aws "github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/packer"
	"github.com/stretchr/testify/assert"
)

// An example of how to test the Packer template in examples/packer-basic-example using Terratest.
func TestPackerBasicExample(t *testing.T) {
	t.Parallel()

	// Pick a random AWS region to test in. This helps ensure your code works in all regions.
	awsRegion := terratest_aws.GetRandomRegion(t, nil, nil)

	packerOptions := &packer.Options{
		// The path to where the Packer template is located
		Template: "../examples/packer-basic-example/build.json",

		// Variables to pass to our Packer build using -var options
		Vars: map[string]string{
			"aws_region": awsRegion,
		},

		// Only build the AWS AMI
		Only: "amazon-ebs",
	}

	// Make sure the Packer build completes successfully
	amiID := packer.BuildArtifact(t, packerOptions)

	// Clean up the AMI after we're done
	defer terratest_aws.DeleteAmiAndAllSnapshots(t, awsRegion, amiID)

	// Check if AMI is shared/not shared with account
	requestingAccount := terratest_aws.CanonicalAccountId
	randomAccount := "123456789012" // Random Account
	ec2Client := terratest_aws.NewEc2Client(t, awsRegion)
	ShareAmi(t, amiID, requestingAccount, ec2Client)
	accountsWithLaunchPermissions := terratest_aws.GetAccountsWithLaunchPermissionsForAmi(t, awsRegion, amiID)
	assert.NotContains(t, accountsWithLaunchPermissions, randomAccount)
	assert.Contains(t, accountsWithLaunchPermissions, requestingAccount)

	// Check if AMI is public
	MakeAmiPublic(t, amiID, ec2Client)
	amiIsPublic := terratest_aws.GetAmiPubliclyAccessible(t, awsRegion, amiID)
	assert.True(t, amiIsPublic)
}

func ShareAmi(t *testing.T, amiID string, accountID string, ec2Client *ec2.EC2) {
	input := &ec2.ModifyImageAttributeInput{
		ImageId: aws.String(amiID),
		LaunchPermission: &ec2.LaunchPermissionModifications{
			Add: []*ec2.LaunchPermission{
				{
					UserId: aws.String(accountID),
				},
			},
		},
	}
	_, err := ec2Client.ModifyImageAttribute(input)
	if err != nil {
		t.Fatal(err)
	}
}

func MakeAmiPublic(t *testing.T, amiID string, ec2Client *ec2.EC2) {
	input := &ec2.ModifyImageAttributeInput{
		ImageId: aws.String(amiID),
		LaunchPermission: &ec2.LaunchPermissionModifications{
			Add: []*ec2.LaunchPermission{
				{
					Group: aws.String("all"),
				},
			},
		},
	}
	_, err := ec2Client.ModifyImageAttribute(input)
	if err != nil {
		t.Fatal(err)
	}
}

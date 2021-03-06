package gcp

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/random"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
)

// Corresponds to a GCP Compute Instance (https://cloud.google.com/compute/docs/instances/)
type Instance struct {
	projectID string
	*compute.Instance
}

// Corresponds to a GCP Image (https://cloud.google.com/compute/docs/images)
type Image struct {
	projectID string
	*compute.Image
}

// Corresponds to a GCP Zonal Instance Group (https://cloud.google.com/compute/docs/instance-groups/)
type ZonalInstanceGroup struct {
	projectID string
	*compute.InstanceGroup
}

// Corresponds to a GCP Regional Instance Group (https://cloud.google.com/compute/docs/instance-groups/)
type RegionalInstanceGroup struct {
	projectID string
	*compute.InstanceGroup
}

type InstanceGroup interface {
	GetInstanceIds(t *testing.T) []string
	GetInstanceIdsE(t *testing.T) ([]string, error)
}

// FetchInstance queries GCP to return an instance of the (GCP Compute) Instance type
func FetchInstance(t *testing.T, projectID string, name string) *Instance {
	instance, err := FetchInstanceE(t, projectID, name)
	if err != nil {
		t.Fatal(err)
	}

	return instance
}

// FetchInstance queries GCP to return an instance of the (GCP Compute) Instance type
func FetchInstanceE(t *testing.T, projectID string, name string) (*Instance, error) {
	logger.Logf(t, "Getting Compute Instance %s", name)

	ctx := context.Background()
	service, err := NewComputeServiceE(t)
	if err != nil {
		t.Fatal(err)
	}

	// If we want to fetch an Instance without knowing its Zone, we have to query GCP for all Instances in the project
	// and match on name.
	instanceAggregatedList, err := service.Instances.AggregatedList(projectID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("Instances.AggregatedList(%s) got error: %v", projectID, err)
	}

	for _, instanceList := range instanceAggregatedList.Items {
		for _, instance := range instanceList.Instances {
			if name == instance.Name {
				return &Instance{projectID, instance}, nil
			}
		}
	}

	return nil, fmt.Errorf("Compute Instance %s could not be found in project %s", name, projectID)
}

// FetchImage queries GCP to return a new instance of the (GCP Compute) Image type
func FetchImage(t *testing.T, projectID string, name string) *Image {
	image, err := FetchImageE(t, projectID, name)
	if err != nil {
		t.Fatal(err)
	}

	return image
}

// FetchImage queries GCP to return a new instance of the (GCP Compute) Image type
func FetchImageE(t *testing.T, projectID string, name string) (*Image, error) {
	logger.Logf(t, "Getting Image %s", name)

	ctx := context.Background()
	service, err := NewComputeServiceE(t)
	if err != nil {
		return nil, err
	}

	req := service.Images.Get(projectID, name)
	image, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return &Image{projectID, image}, nil
}

// FetchRegionalInstanceGroup queries GCP to return a new instance of the Regional Instance Group type
func FetchRegionalInstanceGroup(t *testing.T, projectID string, region string, name string) *RegionalInstanceGroup {
	instanceGroup, err := FetchRegionalInstanceGroupE(t, projectID, region, name)
	if err != nil {
		t.Fatal(err)
	}

	return instanceGroup
}

// FetchRegionalInstanceGroup queries GCP to return a new instance of the Regional Instance Group type
func FetchRegionalInstanceGroupE(t *testing.T, projectID string, region string, name string) (*RegionalInstanceGroup, error) {
	logger.Logf(t, "Getting Regional Instance Group %s", name)

	ctx := context.Background()
	service, err := NewComputeServiceE(t)
	if err != nil {
		return nil, err
	}

	req := service.RegionInstanceGroups.Get(projectID, region, name)
	instanceGroup, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return &RegionalInstanceGroup{projectID, instanceGroup}, nil
}

// FetchZonalInstanceGroup queries GCP to return a new instance of the Regional Instance Group type
func FetchZonalInstanceGroup(t *testing.T, projectID string, zone string, name string) *ZonalInstanceGroup {
	instanceGroup, err := FetchZonalInstanceGroupE(t, projectID, zone, name)
	if err != nil {
		t.Fatal(err)
	}

	return instanceGroup
}

// FetchZonalInstanceGroup queries GCP to return a new instance of the Regional Instance Group type
func FetchZonalInstanceGroupE(t *testing.T, projectID string, zone string, name string) (*ZonalInstanceGroup, error) {
	logger.Logf(t, "Getting Zonal Instance Group %s", name)

	ctx := context.Background()
	service, err := NewComputeServiceE(t)
	if err != nil {
		return nil, err
	}

	req := service.InstanceGroups.Get(projectID, zone, name)
	instanceGroup, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return &ZonalInstanceGroup{projectID, instanceGroup}, nil
}

// GetPublicIP gets the public IP address of the given Compute Instance.
func (i *Instance) GetPublicIp(t *testing.T) string {
	ip, err := i.GetPublicIpE(t)
	if err != nil {
		t.Fatal(err)
	}
	return ip
}

// GetPublicIpE gets the public IP address of the given Compute Instance.
func (i *Instance) GetPublicIpE(t *testing.T) (string, error) {
	// If there are no accessConfigs specified, then this instance will have no external internet access:
	// https://cloud.google.com/compute/docs/reference/rest/v1/instances.
	if len(i.NetworkInterfaces[0].AccessConfigs) == 0 {
		return "", fmt.Errorf("Attempted to get public IP of Compute Instance %s, but that Compute Instance does not have a public IP address", i.Name)
	}

	ip := i.NetworkInterfaces[0].AccessConfigs[0].NatIP

	return ip, nil
}

// GetLabels returns all the tags for the given Compute Instance.
func (i *Instance) GetLabels(t *testing.T) map[string]string {
	return i.Labels
}

// GetZone returns the Zone in which the Compute Instance is located.
func (i *Instance) GetZone(t *testing.T) string {
	return ZoneUrlToZone(i.Zone)
}

// SetLabels adds the tags to the given Compute Instance.
func (i *Instance) SetLabels(t *testing.T, labels map[string]string) {
	err := i.SetLabelsE(t, labels)
	if err != nil {
		t.Fatal(err)
	}
}

// SetLabelsE adds the tags to the given Compute Instance.
func (i *Instance) SetLabelsE(t *testing.T, labels map[string]string) error {
	logger.Logf(t, "Adding labels to instance %s in zone %s", i.Name, i.Zone)

	ctx := context.Background()
	service, err := NewComputeServiceE(t)
	if err != nil {
		return err
	}

	req := compute.InstancesSetLabelsRequest{Labels: labels, LabelFingerprint: i.LabelFingerprint}
	if _, err := service.Instances.SetLabels(i.projectID, i.GetZone(t), i.Name, &req).Context(ctx).Do(); err != nil {
		return fmt.Errorf("Instances.SetLabels(%s) got error: %v", i.Name, err)
	}

	return nil
}

// DeleteImage deletes the given Compute Image.
func (i *Image) DeleteImage(t *testing.T) {
	err := i.DeleteImageE(t)
	if err != nil {
		t.Fatal(err)
	}
}

// DeleteImageE deletes the given Compute Image.
func (i *Image) DeleteImageE(t *testing.T) error {
	logger.Logf(t, "Destroying Image %s", i.Name)

	ctx := context.Background()
	service, err := NewComputeServiceE(t)
	if err != nil {
		return err
	}

	if _, err := service.Images.Delete(i.projectID, i.Name).Context(ctx).Do(); err != nil {
		return fmt.Errorf("Images.Delete(%s) got error: %v", i.Name, err)
	}

	return nil
}

// GetInstanceIds gets the IDs of Instances in the given Instance Group.
func (ig *ZonalInstanceGroup) GetInstanceIds(t *testing.T) []string {
	ids, err := ig.GetInstanceIdsE(t)
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

// GetInstanceIdsE gets the IDs of Instances in the given Zonal Instance Group.
func (ig *ZonalInstanceGroup) GetInstanceIdsE(t *testing.T) ([]string, error) {
	logger.Logf(t, "Get instances for Zonal Instance Group %s", ig.Name)

	ctx := context.Background()
	service, err := NewComputeServiceE(t)
	if err != nil {
		return nil, err
	}

	requestBody := &compute.InstanceGroupsListInstancesRequest{
		InstanceState: "ALL",
	}

	instanceIDs := []string{}
	zone := ZoneUrlToZone(ig.Zone)

	req := service.InstanceGroups.ListInstances(ig.projectID, zone, ig.Name, requestBody)

	err = req.Pages(ctx, func(page *compute.InstanceGroupsListInstances) error {
		for _, instance := range page.Items {
			// For some reason service.InstanceGroups.ListInstances returns us a collection
			// with Instance URLs and we need only the Instance ID for the next call. Use
			// the path functions to chop the Instance ID off the end of the URL.
			instanceID := path.Base(instance.Instance)
			instanceIDs = append(instanceIDs, instanceID)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("InstanceGroups.ListInstances(%s) got error: %v", ig.Name, err)
	}

	return instanceIDs, nil
}

// GetInstanceIds gets the IDs of Instances in the given Regional Instance Group.
func (ig *RegionalInstanceGroup) GetInstanceIds(t *testing.T) []string {
	ids, err := ig.GetInstanceIdsE(t)
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

// GetInstanceIdsE gets the IDs of Instances in the given Regional Instance Group.
func (ig *RegionalInstanceGroup) GetInstanceIdsE(t *testing.T) ([]string, error) {
	logger.Logf(t, "Get instances for Regional Instance Group %s", ig.Name)

	ctx := context.Background()

	service, err := NewComputeServiceE(t)
	if err != nil {
		return nil, err
	}

	requestBody := &compute.RegionInstanceGroupsListInstancesRequest{
		InstanceState: "ALL",
	}

	instanceIDs := []string{}
	region := RegionUrlToRegion(ig.Region)

	req := service.RegionInstanceGroups.ListInstances(ig.projectID, region, ig.Name, requestBody)

	err = req.Pages(ctx, func(page *compute.RegionInstanceGroupsListInstances) error {
		for _, instance := range page.Items {
			// For some reason service.InstanceGroups.ListInstances returns us a collection
			// with Instance URLs and we need only the Instance ID for the next call. Use
			// the path functions to chop the Instance ID off the end of the URL.
			instanceID := path.Base(instance.Instance)
			instanceIDs = append(instanceIDs, instanceID)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("InstanceGroups.ListInstances(%s) got error: %v", ig.Name, err)
	}

	return instanceIDs, nil
}

// getRandomInstance returns a randomly selected Instance from the Regional Instance Group
func (ig *ZonalInstanceGroup) GetRandomInstance(t *testing.T) *Instance {
	return getRandomInstance(t, ig, ig.Name, ig.Region, ig.Size, ig.projectID)
}

// getRandomInstanceE returns a randomly selected Instance from the Regional Instance Group
func (ig *ZonalInstanceGroup) GetRandomInstanceE(t *testing.T) (*Instance, error) {
	return getRandomInstanceE(t, ig, ig.Name, ig.Region, ig.Size, ig.projectID)
}

// getRandomInstance returns a randomly selected Instance from the Regional Instance Group
func (ig *RegionalInstanceGroup) GetRandomInstance(t *testing.T) *Instance {
	return getRandomInstance(t, ig, ig.Name, ig.Region, ig.Size, ig.projectID)
}

// getRandomInstanceE returns a randomly selected Instance from the Regional Instance Group
func (ig *RegionalInstanceGroup) GetRandomInstanceE(t *testing.T) (*Instance, error) {
	return getRandomInstanceE(t, ig, ig.Name, ig.Region, ig.Size, ig.projectID)
}

func getRandomInstance(t *testing.T, ig InstanceGroup, name string, region string, size int64, projectID string) *Instance {
	instance, err := getRandomInstanceE(t, ig, name, region, size, projectID)
	if err != nil {
		t.Fatal(err)
	}

	return instance
}

func getRandomInstanceE(t *testing.T, ig InstanceGroup, name string, region string, size int64, projectID string) (*Instance, error) {
	instanceIDs := ig.GetInstanceIds(t)
	if len(instanceIDs) == 0 {
		return nil, fmt.Errorf("Could not find any instances in Regional Instance Group %s in Region %s", name, region)
	}

	clusterSize := int(size)
	if len(instanceIDs) != clusterSize {
		return nil, fmt.Errorf("Expected Regional Instance Group %s in Region %s to have %d instances, but found %d", name, region, clusterSize, len(instanceIDs))
	}

	randIndex := random.Random(0, clusterSize-1)
	instanceID := instanceIDs[randIndex]
	instance := FetchInstance(t, projectID, instanceID)

	return instance, nil
}

// NewComputeService creates a new Compute service, which is used to make GCP API calls.
func NewComputeService(t *testing.T) *compute.Service {
	client, err := NewComputeServiceE(t)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// NewComputeServiceE creates a new Compute service, which is used to make GCP API calls.
func NewComputeServiceE(t *testing.T) (*compute.Service, error) {
	ctx := context.Background()

	client, err := google.DefaultClient(ctx, compute.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("Failed to get default client: %v", err)
	}

	service, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	return service, nil
}

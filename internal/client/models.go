package client

// Blueprint describes how a replicated machine is launched in the target cloud.
//
// Optional scalar and list fields are pointers so that PATCH requests only
// send the attributes that were actually configured, while still allowing an
// explicit empty list to be sent to clear a value.
type Blueprint struct {
	ID        string `json:"id,omitempty"`
	MachineID string `json:"machineId,omitempty"`

	MachineName             string `json:"machineName,omitempty"`
	Region                  string `json:"region,omitempty"`
	InstanceType            string `json:"instanceType,omitempty"`
	RecommendedInstanceType string `json:"recommendedInstanceType,omitempty"`
	RecommendedPrivateIP    string `json:"recommendedPrivateIP,omitempty"`
	IAMRole                 string `json:"iamRole,omitempty"`
	PlacementGroup          string `json:"placementGroup,omitempty"`
	Tenancy                 string `json:"tenancy,omitempty"`
	DedicatedHostIdentifier string `json:"dedicatedHostIdentifier,omitempty"`
	LaunchOnInstanceID      string `json:"launchOnInstanceId,omitempty"`

	ByolOnDedicatedInstance *bool `json:"byolOnDedicatedInstance,omitempty"`
	ForceUEFI               *bool `json:"forceUEFI,omitempty"`
	RunAfterLaunch          *bool `json:"runAfterLaunch,omitempty"`
	UseSharedRAM            *bool `json:"useSharedRam,omitempty"`

	SubnetIDs           *[]string `json:"subnetIDs,omitempty"`
	SecurityGroupIDs    *[]string `json:"securityGroupIDs,omitempty"`
	SecurityGroupAction string    `json:"securityGroupAction,omitempty"`
	PrivateIPAction     string    `json:"privateIPAction,omitempty"`
	PrivateIPs          *[]string `json:"privateIPs,omitempty"`
	PublicIPAction      string    `json:"publicIPAction,omitempty"`
	StaticIP            string    `json:"staticIp,omitempty"`
	StaticIPAction      string    `json:"staticIpAction,omitempty"`
	NetworkInterface    string    `json:"networkInterface,omitempty"`
	SubnetsHostProject  string    `json:"subnetsHostProject,omitempty"`

	// vCenter-only settings.
	ScsiAdapterType    string `json:"scsiAdapterType,omitempty"`
	NetworkAdapterType string `json:"networkAdapterType,omitempty"`
	Cpus               *int64 `json:"cpus,omitempty"`
	CoresPerCPU        *int64 `json:"coresPerCpu,omitempty"`
	MbRAM              *int64 `json:"mbRam,omitempty"`
	ComputeLocationID  string `json:"computeLocationId,omitempty"`
	LogicalLocationID  string `json:"logicalLocationId,omitempty"`

	Tags  *[]Tag  `json:"tags,omitempty"`
	Disks *[]Disk `json:"disks,omitempty"`
}

// Tag is a key/value tag applied to the launched target machine.
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Disk describes target machine disk properties (AWS only).
type Disk struct {
	Name       string `json:"name"`
	Type       string `json:"type,omitempty"`
	Iops       *int64 `json:"iops,omitempty"`
	Throughput *int64 `json:"throughput,omitempty"`
}

// Machine is a source machine registered in a CloudEndure project.
// Only the fields used by the provider are modelled.
type Machine struct {
	ID                string                  `json:"id"`
	ReplicationStatus string                  `json:"replicationStatus,omitempty"`
	Replica           string                  `json:"replica,omitempty"`
	IsAgentInstalled  bool                    `json:"isAgentInstalled,omitempty"`
	SourceProperties  MachineSourceProperties `json:"sourceProperties"`
}

// MachineSourceProperties holds properties of the source machine.
type MachineSourceProperties struct {
	Name           string `json:"name"`
	OS             string `json:"os,omitempty"`
	MachineCloudID string `json:"machineCloudId,omitempty"`
}

// RecoveryPlan defines the order in which groups of machines are launched.
type RecoveryPlan struct {
	ID    string            `json:"id,omitempty"`
	Name  string            `json:"name"`
	Steps RecoveryPlanSteps `json:"steps"`
}

// RecoveryPlanSteps wraps the ordered list of recovery plan steps.
type RecoveryPlanSteps struct {
	Items []RecoveryPlanStep `json:"items"`
}

// RecoveryPlanStep is a single launch step in a recovery plan.
type RecoveryPlanStep struct {
	Name       string   `json:"name,omitempty"`
	MachineIDs []string `json:"machineIDs"`
	// WaitBefore is the delay before launching this step, as returned by the API.
	WaitBefore string `json:"waitBefore,omitempty"`
}

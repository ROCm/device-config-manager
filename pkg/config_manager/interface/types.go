package types

type PartitionStatus struct {
	SelectedProfile string
	FinalStatus     string
	Reason          string
	GPUStatus       []GPUPartitionStatus
}

type GPUPartitionStatus struct {
	GpuID         int
	PartitionType string
	Status        string
	Message       string
}

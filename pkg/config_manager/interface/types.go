package types

type PartitionStatus struct {
	SelectedProfile string               `json:"SelectedProfile"`
	FinalStatus     string               `json:"FinalStatus"`
	Reason          string               `json:"Reason"`
	GPUStatus       []GPUPartitionStatus `json:"GPUStatus,omitempty"`
}

type GPUPartitionStatus struct {
	GpuID         int
	PartitionType string
	Status        string
	Message       string
}

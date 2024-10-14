package models

// CPUType represents allowed CPU allocation options
type CPUType string

const (
    CPU100m CPUType = "100m"
    CPU250m CPUType = "250m"
    CPU500m CPUType = "500m"
    CPU1    CPUType = "1"    // Represents "1" CPU core
    CPU2    CPUType = "2"    // Represents "2" CPU cores
)

// MemoryType represents allowed Memory allocation options
type MemoryType string

const (
    Memory128Mi MemoryType = "128Mi"
    Memory256Mi MemoryType = "256Mi"
    Memory512Mi MemoryType = "512Mi"
    Memory1Gi   MemoryType = "1Gi"
    Memory2Gi   MemoryType = "2Gi"
)
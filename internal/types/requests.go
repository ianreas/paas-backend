package types

type BuildAndPushRequest struct {
	RepoFullName    string  `json:"repoFullName"`
	AccessToken     string  `json:"accessToken"`
	UserId          string  `json:"userId"`
	GithubUsername  string  `json:"githubUsername"`
	ContainerPort   int32   `json:"containerPort,omitempty"`
	Replicas        *int32  `json:"replicas,omitempty"`
	CPU             *string `json:"cpuAllocation,omitempty"`
	Memory          *string `json:"memoryAllocation,omitempty"`
}
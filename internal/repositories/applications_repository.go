package repositories

import (
	"context"
)

// Application represents a record in the applications table.
type Application struct {
	ID             int
	GithubRepoName string
	GithubUsername string
	UserID         string
	ProjectName    string
	ContainerPort  int32   // Optional field
	Replicas       *int32  // Optional field
	CPU            *string // Optional field
	Memory         *string // Optional field
}

// ApplicationsRepository defines methods for interacting with the applications table.
type ApplicationsRepository interface {
	CreateOrUpdateApplication(ctx context.Context, app *Application) (int, error)
}

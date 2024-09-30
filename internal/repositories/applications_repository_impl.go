package repositories

import (
	"context"
	"database/sql"
	"fmt"
)

type ApplicationsRepositoryImpl struct {
	db *sql.DB
}

// NewApplicationsRepository creates a new ApplicationsRepository.
func NewApplicationsRepository(db *sql.DB) ApplicationsRepository {
	return &ApplicationsRepositoryImpl{
		db: db,
	}
}

func (repo *ApplicationsRepositoryImpl) CreateOrUpdateApplication(ctx context.Context, app *Application) (int, error) {
	if repo.db == nil {
		return 0, fmt.Errorf("database connection is not initialized")
	}

	query := `
		INSERT INTO applications (github_repo_name, github_username, user_id, project_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, project_name)
		DO UPDATE SET
			github_repo_name = EXCLUDED.github_repo_name,
			github_username = EXCLUDED.github_username
		RETURNING id
	`

	var id int
	err := repo.db.QueryRowContext(ctx, query, app.GithubRepoName, app.GithubUsername, app.UserID, app.ProjectName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert application: %w", err)
	}
	return id, nil
}
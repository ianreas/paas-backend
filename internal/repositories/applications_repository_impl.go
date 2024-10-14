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

// sqlNullInt32 converts a *int32 to sql.NullInt32
func sqlNullInt32(i *int32) sql.NullInt32 {
	if i != nil {
		return sql.NullInt32{Int32: *i, Valid: true}
	}
	return sql.NullInt32{Int32: 0, Valid: false}
}

// sqlNullString converts a *string to sql.NullString
func sqlNullString(s *string) sql.NullString {
	if s != nil {
		return sql.NullString{String: *s, Valid: true}
	}
	return sql.NullString{String: "", Valid: false}
}

func (repo *ApplicationsRepositoryImpl) CreateOrUpdateApplication(ctx context.Context, app *Application) (int, error) {
	if repo.db == nil {
		return 0, fmt.Errorf("database connection is not initialized")
	}

	fmt.Printf("Application object: %v\n", app)

	query := `
        INSERT INTO applications (
            github_repo_name,
            github_username,
            user_id,
            project_name,
            replicas,
            requested_cpu,
            requested_memory,
            container_port
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (user_id, project_name)
        DO UPDATE SET
            github_repo_name = EXCLUDED.github_repo_name,
            github_username = EXCLUDED.github_username,
            replicas = EXCLUDED.replicas,
            requested_cpu = EXCLUDED.requested_cpu,
            requested_memory = EXCLUDED.requested_memory,
            container_port = EXCLUDED.container_port
        RETURNING id
    `

	// Prepare the arguments, handling nil values for optional fields
	args := []interface{}{
		app.GithubRepoName,         // $1
		app.GithubUsername,         // $2
		app.UserID,                 // $3
		app.ProjectName,            // $4
		sqlNullInt32(app.Replicas), // $5
		sqlNullString(app.CPU),     // $6
		sqlNullString(app.Memory),  // $7
		app.ContainerPort,          // $8
	}

	var id int
	err := repo.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert application: %w", err)
	}
	return id, nil
}

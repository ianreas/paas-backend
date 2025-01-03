package services

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"text/template"
	"time"

	"github.com/google/go-github/v67/github"
	"golang.org/x/oauth2"

	"paas-backend/internal/types"

	"database/sql"
)

// WorkflowRequest represents the data required to create the workflow
type WorkflowRequest struct {
	ApplicationId string `json:"applicationId"`
	RepoOwner   string `json:"repoOwner"`
	RepoName    string `json:"repoName"`
	AccessToken string `json:"accessToken"`
	BaseBranch  string `json:"baseBranch"`
}

// WorkflowResponse represents the response after creating the workflow
type WorkflowResponse struct {
	PRNumber int    `json:"prNumber"`
	PRURL    string `json:"prUrl"`
}

// GitHubService defines the interface for GitHub operations
type GitHubService interface {
	CreateWorkflow(ctx context.Context, req WorkflowRequest) (WorkflowResponse, error)
}

type gitHubService struct {
    db *sql.DB
}

func NewGitHubService(db *sql.DB) GitHubService {
    return &gitHubService{
        db: db,
    }
}

// CreateWorkflow creates a GitHub Actions workflow in the repository and opens a PR
func (s *gitHubService) CreateWorkflow(ctx context.Context, req WorkflowRequest) (WorkflowResponse, error) {
	log.Printf("Starting CreateWorkflow for %s/%s/%s", req.RepoOwner, req.RepoName, req.AccessToken)

	// Create GitHub client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: req.AccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	buildAndPushRequest, err := s.fetchRepo(ctx, req)
	if err != nil {
		log.Printf("Error fetching repository: %v", err)
		return WorkflowResponse{}, fmt.Errorf("error fetching repository: %v", err)
	}

	// Generate branch name
	branchName := fmt.Sprintf("feat/add-paas-workflow-%s", time.Now().Format("20060102150405"))
	log.Printf("Generated branch name: %s", branchName)


	// Generate workflow content
	workflowContent, err := generateWorkflowContent(buildAndPushRequest)
	if err != nil {
		log.Printf("Error generating workflow content: %v", err)
		return WorkflowResponse{}, fmt.Errorf("error generating workflow: %v", err)
	}

	// Create branch
	if err := createBranch(ctx, client, req, branchName); err != nil {
		log.Printf("Error creating branch: %v", err)
		return WorkflowResponse{}, fmt.Errorf("error creating branch: %v", err)
	}
	log.Printf("Branch %s created successfully", branchName)

	// Create workflow file
	if err := createWorkflowFile(ctx, client, req, branchName, workflowContent); err != nil {
		log.Printf("Error creating workflow file: %v", err)
		return WorkflowResponse{}, fmt.Errorf("error creating workflow file: %v", err)
	}
	log.Printf("Workflow file created in branch %s", branchName)

	// Create pull request
	pr, err := createPullRequest(ctx, client, req, branchName)
	if err != nil {
		log.Printf("Error creating pull request: %v", err)
		return WorkflowResponse{}, fmt.Errorf("error creating PR: %v", err)
	}
	log.Printf("Pull request #%d created successfully: %s", pr.GetNumber(), pr.GetHTMLURL())

	// Return the PR details
	return WorkflowResponse{
		PRNumber: pr.GetNumber(),
		PRURL:    pr.GetHTMLURL(),
	}, nil

}



func (s *gitHubService) fetchRepo(ctx context.Context, workflowRequestObject WorkflowRequest) (types.BuildAndPushRequest, error){
	if s.db == nil {
        return types.BuildAndPushRequest{}, fmt.Errorf("database connection is not initialized")
    }

	query := `
        SELECT github_repo_name, github_username, project_name, container_port
        FROM applications 
        WHERE id = $1
    `

	var app struct {
        GithubRepoName string
        GithubUsername string
        ProjectName    string
        ContainerPort  int32
    }

	err := s.db.QueryRowContext(ctx, query, workflowRequestObject.ApplicationId).Scan(
        &app.GithubRepoName,
        &app.GithubUsername,
        &app.ProjectName,
        &app.ContainerPort,
    )

	if err != nil {
        if err == sql.ErrNoRows {
            return types.BuildAndPushRequest{}, fmt.Errorf("application not found with ID: %s", workflowRequestObject.ApplicationId)
        }
        return types.BuildAndPushRequest{}, fmt.Errorf("error fetching application: %v", err)
    }

    return types.BuildAndPushRequest{
        RepoFullName: fmt.Sprintf("%s/%s", app.GithubUsername, app.GithubRepoName),
        AccessToken:  workflowRequestObject.AccessToken,
    }, nil
}

// generateWorkflowContent generates the content of the GitHub Actions workflow file
func generateWorkflowContent(buildAndPushRequestObject types.BuildAndPushRequest) (string, error) {
	log.Println("Generating workflow content")
	tmpl, err := template.New("workflow").Delims("[[", "]]").Parse(workflowTemplate)
	if err != nil {
		log.Printf("Error parsing workflow template: %v", err)
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		log.Printf("Error executing workflow template: %v", err)
		return "", err
	}

	log.Println("Workflow content generated successfully")
	return buf.String(), nil
}

// createBranch creates a new branch in the repository
func createBranch(ctx context.Context, client *github.Client, req WorkflowRequest, branchName string) error {
	log.Printf("Fetching reference for base branch %s", req.BaseBranch)

	ref, _, err := client.Git.GetRef(ctx, req.RepoOwner, req.RepoName, fmt.Sprintf("refs/heads/%s", req.BaseBranch))
	if err != nil {
		log.Printf("Error fetching base branch reference: %v", err)
		return err
	}

	log.Printf("Creating new branch %s from base branch %s", branchName, req.BaseBranch)
	newRef := &github.Reference{
		Ref: github.String(fmt.Sprintf("refs/heads/%s", branchName)),
		Object: &github.GitObject{
			SHA: ref.Object.SHA,
		},
	}

	_, _, err = client.Git.CreateRef(ctx, req.RepoOwner, req.RepoName, newRef)
	if err != nil {
		log.Printf("Error creating new branch reference: %v", err)
		return err
	}

	log.Printf("Branch %s created successfully", branchName)
	return nil
}

func createWorkflowFile(ctx context.Context, client *github.Client, req WorkflowRequest, branchName string, content string) error {
    path := "github-workflows-draft/workflows/paas-deploy.yml"
    message := "Add PaaS deployment workflow"
    log.Printf("Creating workflow file at %s in branch %s", path, branchName)
    log.Printf("Workflow content length: %d bytes", len(content))

    opts := &github.RepositoryContentFileOptions{
        Message: &message,
        Content: []byte(content),
        Branch:  &branchName,
    }

    fileResponse, resp, err := client.Repositories.CreateFile(ctx, req.RepoOwner, req.RepoName, path, opts)
    if err != nil {
        log.Printf("Error creating workflow file: %v", err)
        if ghErr, ok := err.(*github.ErrorResponse); ok {
            log.Printf("GitHub Error Message: %s", ghErr.Message)
            for _, e := range ghErr.Errors {
                log.Printf("Error detail: %s", e.Message)
            }
            if resp != nil {
                log.Printf("HTTP Status: %d", resp.StatusCode)
            }
        }
        return err
    }

    log.Printf("Workflow file created successfully: %+v", fileResponse)
    return nil
}



// createPullRequest opens a new pull request for the workflow
func createPullRequest(ctx context.Context, client *github.Client, req WorkflowRequest, branchName string) (*github.PullRequest, error) {
	title := "Add PaaS Deployment Workflow"
	body := prTemplate
	base := req.BaseBranch
	log.Printf("Creating pull request from branch %s to base %s", branchName, base)

	newPR := &github.NewPullRequest{
		Title: &title,
		Head:  &branchName,
		Base:  &base,
		Body:  &body,
	}

	pr, _, err := client.PullRequests.Create(ctx, req.RepoOwner, req.RepoName, newPR)
	if err != nil {
		log.Printf("Error creating pull request: %v", err)
		return nil, err
	}

	log.Printf("Pull request #%d created successfully", pr.GetNumber())

	// Add labels to the PR
	labels := []string{"automation", "ci-cd", "paas"}
	_, _, err = client.Issues.AddLabelsToIssue(ctx, req.RepoOwner, req.RepoName, pr.GetNumber(), labels)
	if err != nil {
		log.Printf("Warning: Failed to add labels to PR: %v", err)
	} else {
		log.Printf("Labels added to pull request #%d", pr.GetNumber())
	}

	return pr, nil
}

const workflowTemplate = 
`name: Deploy to PaaS

on:
  push:
    branches: [ main, staging ]
  pull_request:
    branches: [ main, staging ]

env:
  PAAS_API_URL: ${{ secrets.PAAS_API_URL }}
  PAAS_ACCESS_TOKEN: ${{ secrets.PAAS_ACCESS_TOKEN }}

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Detect package manager
        id: detect-package-manager
        run: |
          if [ -f "./yarn.lock" ]; then
            echo "manager=yarn" >> $GITHUB_OUTPUT
          elif [ -f "./package-lock.json" ]; then
            echo "manager=npm" >> $GITHUB_OUTPUT
          else
            echo "No lock file found"
            exit 1
          fi
      
      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: ${{ steps.detect-package-manager.outputs.manager }}

      - name: Install dependencies
        run: ${{ steps.detect-package-manager.outputs.manager }} install
      
      - name: Type check
        run: ${{ steps.detect-package-manager.outputs.manager }} run type-check || true
      
      - name: Lint
        run: ${{ steps.detect-package-manager.outputs.manager }} run lint || true
      
      - name: Test
        run: ${{ steps.detect-package-manager.outputs.manager }} test || true

  build:
    needs: validate
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build validation
        run: |
          if [ ! -f "Dockerfile" ]; then
            echo "No Dockerfile found, creating one..."
            cat > Dockerfile << EOF
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/public ./public

EXPOSE 3000
CMD ["npm", "start"]
EOF
          fi

  deploy:
    needs: build
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && (github.ref == 'refs/heads/main' || github.ref == 'refs/heads/staging')
    steps:
      - uses: actions/checkout@v4

      - name: Deploy to PaaS
        run: |
          curl -X POST "${PAAS_API_URL}/api/v1/build-and-push-deploy" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer ${PAAS_ACCESS_TOKEN}" \
          -d "{\"repoFullName\":\"${{ github.repository }}\",\"accessToken\":\"${{ github.token }}\"}"`


// Templates for the workflow and PR body
// const workflowTemplate = `name: Deploy to PaaS

// on:
//   push:
//     branches: [ main, staging ]
//   pull_request:
//     branches: [ main, staging ]

// env:
//   PAAS_API_URL: ${{ secrets.PAAS_API_URL }}
//   PAAS_ACCESS_TOKEN: ${{ secrets.PAAS_ACCESS_TOKEN }}

// jobs:
//   validate:
//     runs-on: ubuntu-latest
//     steps:
//       - uses: actions/checkout@v4
      
//       - name: Detect package manager
//         id: detect-package-manager
//         run: |
//           if [ -f "./yarn.lock" ]; then
//             echo "manager=yarn" >> $GITHUB_OUTPUT
//           elif [ -f "./package-lock.json" ]; then
//             echo "manager=npm" >> $GITHUB_OUTPUT
//           else
//             echo "No lock file found"
//             exit 1
//           fi
      
//       - name: Setup Node.js
//         uses: actions/setup-node@v4
//         with:
//           node-version: '20'
//           cache: ${{ steps.detect-package-manager.outputs.manager }}

//       - name: Install dependencies
//         run: ${{ steps.detect-package-manager.outputs.manager }} install
      
//       - name: Type check
//         run: ${{ steps.detect-package-manager.outputs.manager }} run type-check || true
      
//       - name: Lint
//         run: ${{ steps.detect-package-manager.outputs.manager }} run lint || true
      
//       - name: Test
//         run: ${{ steps.detect-package-manager.outputs.manager }} test || true

//   build:
//     needs: validate
//     runs-on: ubuntu-latest
//     steps:
//       - uses: actions/checkout@v4

//       - name: Build validation
//         run: |
//           if [ ! -f "Dockerfile" ]; then
//             echo "No Dockerfile found, creating one..."
//             cat > Dockerfile << 'EOF'
//             FROM node:20-alpine AS builder
//             WORKDIR /app
//             COPY package*.json ./
//             RUN npm install
//             COPY . .
//             RUN npm run build

//             FROM node:20-alpine AS runner
//             WORKDIR /app
//             COPY --from=builder /app/.next ./.next
//             COPY --from=builder /app/node_modules ./node_modules
//             COPY --from=builder /app/package.json ./package.json
//             COPY --from=builder /app/public ./public

//             EXPOSE 3000
//             CMD ["npm", "start"]
//             EOF
//           fi

//   deploy:
//     needs: build
//     runs-on: ubuntu-latest
//     if: github.event_name == 'push' && (github.ref == 'refs/heads/main' || github.ref == 'refs/heads/staging')
//     steps:
//       - uses: actions/checkout@v4

//       - name: Deploy to PaaS
//         run: |
//           curl -X POST "${PAAS_API_URL}/api/v1/build-and-push-deploy" \
//           -H "Content-Type: application/json" \
//           -H "Authorization: Bearer ${PAAS_ACCESS_TOKEN}" \
//           -d "{\"repoFullName\":\"${{ github.repository }}\",\"accessToken\":\"${{ github.token }}\"}"
// `

const prTemplate = `# 🚀 Automated CI/CD Setup for PaaS Deployment

This PR adds an automated deployment workflow to deploy your application to our PaaS platform.

## 🔧 Changes Made

1. Added GitHub Actions workflow for automated deployments
2. Included validation steps (type checking, linting, testing)
3. Added build process with Docker support
4. Configured automatic deployments to staging/production

## 🔐 Required Secrets

Please add these secrets to your repository settings:

1. **PAAS_API_URL**: The URL of the PaaS API (will be provided)
2. **PAAS_ACCESS_TOKEN**: Your PaaS authentication token (will be provided)

## 📋 Next Steps

1. Review the workflow configuration
2. Add the required secrets to your repository
3. Merge this PR to enable automated deployments

## 🤝 Support

If you need any assistance, please don't hesitate to reach out to our support team.
`
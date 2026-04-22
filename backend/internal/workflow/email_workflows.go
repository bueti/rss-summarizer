package workflow

import (
	"time"

	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	EmailPollingTaskQueue = "email-polling"
)

// EmailPollerWorkflow polls all active email sources once (scheduled by cron)
func EmailPollerWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)

	// Get active email sources
	var emailSourceIDs []uuid.UUID
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 3,
			},
		}),
		"GetActiveEmailSourcesActivity",
	).Get(ctx, &emailSourceIDs)

	if err != nil {
		logger.Error("Failed to get active email sources", "error", err)
		return err
	}

	logger.Info("Found email sources to poll", "count", len(emailSourceIDs))

	// Start child workflow for each email source
	for _, sourceID := range emailSourceIDs {
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:            "process-email-source-" + sourceID.String(),
			TaskQueue:             EmailPollingTaskQueue,
			WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			ParentClosePolicy:     enumspb.PARENT_CLOSE_POLICY_ABANDON, // Don't terminate child if parent completes
		})

		// Start the workflow asynchronously (don't wait for completion)
		childWorkflowFuture := workflow.ExecuteChildWorkflow(childCtx, ProcessEmailSourceWorkflow, sourceID)

		// Wait for child workflow to be started (not completed) to ensure it's registered in Temporal
		var childExecution workflow.Execution
		if err := childWorkflowFuture.GetChildWorkflowExecution().Get(ctx, &childExecution); err != nil {
			logger.Error("Failed to start email source processing workflow", "sourceID", sourceID, "error", err)
			continue
		}

		logger.Info("Started email source processing workflow", "sourceID", sourceID, "workflowID", childExecution.ID)
	}

	logger.Info("Email polling cycle completed", "sources_processed", len(emailSourceIDs))
	return nil
}

// ProcessEmailSourceWorkflow fetches emails from an email source and processes them
func ProcessEmailSourceWorkflow(ctx workflow.Context, emailSourceID uuid.UUID) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Processing email source", "emailSourceID", emailSourceID)

	// Fetch emails
	var fetchOutput FetchEmailsOutput
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 3 * time.Minute, // Longer timeout for email fetching
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    2 * time.Second,
				BackoffCoefficient: 2.0,
				MaximumInterval:    time.Minute,
				MaximumAttempts:    3,
			},
		}),
		"FetchEmailsActivity",
		FetchEmailsInput{EmailSourceID: emailSourceID},
	).Get(ctx, &fetchOutput)

	if err != nil {
		logger.Error("Failed to fetch emails", "emailSourceID", emailSourceID, "error", err)
		return err
	}

	logger.Info("Fetched emails", "emailSourceID", emailSourceID, "newArticles", len(fetchOutput.NewArticleIDs))

	// Process each new article (summarize them)
	for _, articleID := range fetchOutput.NewArticleIDs {
		// Start child workflow to summarize article
		// Reuse the same SummarizeArticleWorkflow used for RSS articles
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:            "summarize-article-" + articleID.String(),
			TaskQueue:             EmailPollingTaskQueue,
			WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			ParentClosePolicy:     enumspb.PARENT_CLOSE_POLICY_ABANDON, // Don't terminate child if parent completes
		})

		// Start the workflow asynchronously (don't wait for completion)
		childWorkflowFuture := workflow.ExecuteChildWorkflow(childCtx, SummarizeArticleWorkflow, articleID)

		// Wait for child workflow to be started (not completed) to ensure it's registered in Temporal
		var childExecution workflow.Execution
		if err := childWorkflowFuture.GetChildWorkflowExecution().Get(ctx, &childExecution); err != nil {
			logger.Error("Failed to start article summarization workflow", "articleID", articleID, "error", err)
			continue
		}

		logger.Info("Started article summarization workflow", "articleID", articleID, "workflowID", childExecution.ID)
	}

	return nil
}

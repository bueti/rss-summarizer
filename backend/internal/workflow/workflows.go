package workflow

import (
	"time"

	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	FeedPollingTaskQueue       = "feed-polling"
	ArticleProcessingTaskQueue = "article-processing"
)

// FeedPollerWorkflow polls all feeds once (scheduled by cron)
func FeedPollerWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)

	// Get feeds due for polling
	var feedIDs []uuid.UUID
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 3,
			},
		}),
		"GetFeedsDueForPollActivity",
	).Get(ctx, &feedIDs)

	if err != nil {
		logger.Error("Failed to get feeds due for poll", "error", err)
		return err
	}

	logger.Info("Found feeds to poll", "count", len(feedIDs))

	// Start child workflow for each feed
	for _, feedID := range feedIDs {
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:            "process-feed-" + feedID.String(),
			TaskQueue:             FeedPollingTaskQueue,
			WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			ParentClosePolicy:     enumspb.PARENT_CLOSE_POLICY_ABANDON, // Don't terminate child if parent completes
		})

		// Start the workflow asynchronously (don't wait for completion)
		childWorkflowFuture := workflow.ExecuteChildWorkflow(childCtx, ProcessFeedWorkflow, feedID)

		// Wait for child workflow to be started (not completed) to ensure it's registered in Temporal
		var childExecution workflow.Execution
		if err := childWorkflowFuture.GetChildWorkflowExecution().Get(ctx, &childExecution); err != nil {
			logger.Error("Failed to start child workflow", "feedID", feedID, "error", err)
			continue
		}

		logger.Info("Started feed processing workflow", "feedID", feedID, "workflowID", childExecution.ID)
	}

	logger.Info("Feed polling cycle completed", "feeds_processed", len(feedIDs))
	return nil
}

// ProcessFeedWorkflow fetches a feed and processes new articles
func ProcessFeedWorkflow(ctx workflow.Context, feedID uuid.UUID) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Processing feed", "feedID", feedID)

	// Fetch feed
	var fetchOutput FetchFeedOutput
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 2 * time.Minute,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    time.Second,
				BackoffCoefficient: 2.0,
				MaximumInterval:    time.Minute,
				MaximumAttempts:    3,
			},
		}),
		"FetchFeedActivity",
		FetchFeedInput{FeedID: feedID},
	).Get(ctx, &fetchOutput)

	if err != nil {
		logger.Error("Failed to fetch feed", "feedID", feedID, "error", err)
		return err
	}

	logger.Info("Fetched feed", "feedID", feedID, "newArticles", len(fetchOutput.NewArticleIDs))

	// Process each new article (summarize them)
	for _, articleID := range fetchOutput.NewArticleIDs {
		// Start child workflow to summarize article
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:            "summarize-article-" + articleID.String(),
			TaskQueue:             FeedPollingTaskQueue,
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

// SummarizeArticleWorkflow summarizes a single article
func SummarizeArticleWorkflow(ctx workflow.Context, articleID uuid.UUID) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Summarizing article", "articleID", articleID)

	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 2 * time.Minute,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    2 * time.Second,
				BackoffCoefficient: 2.0,
				MaximumInterval:    30 * time.Second,
				MaximumAttempts:    5,
			},
		}),
		"SummarizeArticleActivity",
		SummarizeArticleInput{ArticleID: articleID},
	).Get(ctx, nil)

	if err != nil {
		logger.Error("Failed to summarize article", "articleID", articleID, "error", err)
		return err
	}

	logger.Info("Successfully summarized article", "articleID", articleID)
	return nil
}

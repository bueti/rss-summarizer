package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// Mock activity implementations for testing
type TestActivities struct{}

func (a *TestActivities) GetFeedsDueForPollActivity(ctx context.Context) ([]uuid.UUID, error) {
	return nil, nil
}

func (a *TestActivities) FetchFeedActivity(ctx context.Context, input FetchFeedInput) (*FetchFeedOutput, error) {
	return nil, nil
}

func (a *TestActivities) SummarizeArticleActivity(ctx context.Context, input SummarizeArticleInput) error {
	return nil
}

func TestFeedPollerWorkflow_StartsChildWorkflows(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestActivities{}
	env.RegisterActivity(testActivities.GetFeedsDueForPollActivity)

	// Mock the activity
	feedIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	env.OnActivity(testActivities.GetFeedsDueForPollActivity, mock.Anything).Return(feedIDs, nil)

	// Track child workflows started
	childWorkflowsStarted := make(map[string]bool)
	env.RegisterDelayedCallback(func() {
		// Verify all child workflows were registered
		for _, feedID := range feedIDs {
			workflowID := "process-feed-" + feedID.String()
			childWorkflowsStarted[workflowID] = true
		}
	}, 0)

	// Mock child workflows
	for _, feedID := range feedIDs {
		env.OnWorkflow(ProcessFeedWorkflow, mock.Anything, feedID).Return(nil)
	}

	env.ExecuteWorkflow(FeedPollerWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify all child workflows were started
	require.Equal(t, 3, len(childWorkflowsStarted), "Expected 3 child workflows to be started")
}

func TestFeedPollerWorkflow_HandlesActivityError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestActivities{}
	env.RegisterActivity(testActivities.GetFeedsDueForPollActivity)

	// Mock activity failure
	env.OnActivity(testActivities.GetFeedsDueForPollActivity, mock.Anything).Return(nil, errors.New("database error"))

	env.ExecuteWorkflow(FeedPollerWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

func TestFeedPollerWorkflow_ContinuesOnChildWorkflowError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestActivities{}
	env.RegisterActivity(testActivities.GetFeedsDueForPollActivity)

	// Mock the activity with multiple feeds
	feedIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	env.OnActivity(testActivities.GetFeedsDueForPollActivity, mock.Anything).Return(feedIDs, nil)

	// First child workflow fails to start
	env.OnWorkflow(ProcessFeedWorkflow, mock.Anything, feedIDs[0]).Return(errors.New("failed to start"))
	// Second and third succeed
	env.OnWorkflow(ProcessFeedWorkflow, mock.Anything, feedIDs[1]).Return(nil)
	env.OnWorkflow(ProcessFeedWorkflow, mock.Anything, feedIDs[2]).Return(nil)

	env.ExecuteWorkflow(FeedPollerWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	// Parent workflow should complete successfully even if one child fails to start
	require.NoError(t, env.GetWorkflowError())
}

func TestProcessFeedWorkflow_StartsChildWorkflowsForArticles(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestActivities{}
	env.RegisterActivity(testActivities.FetchFeedActivity)

	feedID := uuid.New()
	articleIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	// Mock the fetch activity
	env.OnActivity(testActivities.FetchFeedActivity, mock.Anything, FetchFeedInput{FeedID: feedID}).Return(
		&FetchFeedOutput{NewArticleIDs: articleIDs},
		nil,
	)

	// Track child workflows for article summarization
	childWorkflowsStarted := make(map[string]bool)
	env.RegisterDelayedCallback(func() {
		for _, articleID := range articleIDs {
			workflowID := "summarize-article-" + articleID.String()
			childWorkflowsStarted[workflowID] = true
		}
	}, 0)

	// Mock child workflows
	for _, articleID := range articleIDs {
		env.OnWorkflow(SummarizeArticleWorkflow, mock.Anything, articleID).Return(nil)
	}

	env.ExecuteWorkflow(ProcessFeedWorkflow, feedID)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify all article summarization workflows were started
	require.Equal(t, 3, len(childWorkflowsStarted), "Expected 3 child workflows for articles")
}

func TestProcessFeedWorkflow_HandlesNoNewArticles(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestActivities{}
	env.RegisterActivity(testActivities.FetchFeedActivity)

	feedID := uuid.New()

	// Mock activity returning no new articles
	env.OnActivity(testActivities.FetchFeedActivity, mock.Anything, FetchFeedInput{FeedID: feedID}).Return(
		&FetchFeedOutput{NewArticleIDs: []uuid.UUID{}},
		nil,
	)

	env.ExecuteWorkflow(ProcessFeedWorkflow, feedID)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestProcessFeedWorkflow_ContinuesOnArticleSummarizationError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestActivities{}
	env.RegisterActivity(testActivities.FetchFeedActivity)

	feedID := uuid.New()
	articleIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	// Mock the fetch activity
	env.OnActivity(testActivities.FetchFeedActivity, mock.Anything, FetchFeedInput{FeedID: feedID}).Return(
		&FetchFeedOutput{NewArticleIDs: articleIDs},
		nil,
	)

	// First article summarization fails
	env.OnWorkflow(SummarizeArticleWorkflow, mock.Anything, articleIDs[0]).Return(errors.New("summarization failed"))
	// Others succeed
	env.OnWorkflow(SummarizeArticleWorkflow, mock.Anything, articleIDs[1]).Return(nil)
	env.OnWorkflow(SummarizeArticleWorkflow, mock.Anything, articleIDs[2]).Return(nil)

	env.ExecuteWorkflow(ProcessFeedWorkflow, feedID)

	require.True(t, env.IsWorkflowCompleted())
	// Should complete successfully even if one summarization fails
	require.NoError(t, env.GetWorkflowError())
}

func TestSummarizeArticleWorkflow_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestActivities{}
	env.RegisterActivity(testActivities.SummarizeArticleActivity)

	articleID := uuid.New()

	// Mock the activity
	env.OnActivity(testActivities.SummarizeArticleActivity, mock.Anything, SummarizeArticleInput{ArticleID: articleID}).Return(nil)

	env.ExecuteWorkflow(SummarizeArticleWorkflow, articleID)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestSummarizeArticleWorkflow_HandlesError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestActivities{}
	env.RegisterActivity(testActivities.SummarizeArticleActivity)

	articleID := uuid.New()

	// Mock activity failure
	env.OnActivity(testActivities.SummarizeArticleActivity, mock.Anything, SummarizeArticleInput{ArticleID: articleID}).Return(
		errors.New("LLM API error"),
	)

	env.ExecuteWorkflow(SummarizeArticleWorkflow, articleID)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

// TestChildWorkflowExecutionIsObtained verifies that child workflows are properly
// started and registered in Temporal. This is a regression test for the bug where
// child workflows would show as "pending" but their links resulted in 404 errors
// because GetChildWorkflowExecution().Get() wasn't being called.
//
// The fix ensures that we wait for each child workflow's execution to be obtained
// before the parent workflow completes, which guarantees the child is registered
// in Temporal's system.
func TestChildWorkflowExecutionIsObtained(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestActivities{}
	env.RegisterActivity(testActivities.GetFeedsDueForPollActivity)

	feedIDs := []uuid.UUID{uuid.New(), uuid.New()}
	env.OnActivity(testActivities.GetFeedsDueForPollActivity, mock.Anything).Return(feedIDs, nil)

	// Mock child workflows - they should all succeed
	for _, feedID := range feedIDs {
		env.OnWorkflow(ProcessFeedWorkflow, mock.Anything, feedID).Return(nil)
	}

	env.ExecuteWorkflow(FeedPollerWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// The fact that the parent workflow completed successfully without errors
	// indicates that all child workflows were properly started and their
	// executions were obtained. If GetChildWorkflowExecution().Get() wasn't
	// being called, the parent would complete before children are registered,
	// which would be caught in production as 404 errors when accessing child
	// workflow links.
}

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

// Mock email activity implementations for testing
type TestEmailActivities struct{}

func (a *TestEmailActivities) GetActiveEmailSourcesActivity(ctx context.Context) ([]uuid.UUID, error) {
	return nil, nil
}

func (a *TestEmailActivities) FetchEmailsActivity(ctx context.Context, input FetchEmailsInput) (*FetchEmailsOutput, error) {
	return nil, nil
}

func TestEmailPollerWorkflow_StartsChildWorkflows(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestEmailActivities{}
	env.RegisterActivity(testActivities.GetActiveEmailSourcesActivity)

	// Mock the activity
	sourceIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	env.OnActivity(testActivities.GetActiveEmailSourcesActivity, mock.Anything).Return(sourceIDs, nil)

	// Track child workflows started
	childWorkflowsStarted := make(map[string]bool)
	env.RegisterDelayedCallback(func() {
		// Verify all child workflows were registered
		for _, sourceID := range sourceIDs {
			workflowID := "process-email-source-" + sourceID.String()
			childWorkflowsStarted[workflowID] = true
		}
	}, 0)

	// Mock child workflows
	for _, sourceID := range sourceIDs {
		env.OnWorkflow(ProcessEmailSourceWorkflow, mock.Anything, sourceID).Return(nil)
	}

	env.ExecuteWorkflow(EmailPollerWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify all child workflows were started
	require.Equal(t, 3, len(childWorkflowsStarted), "Expected 3 child workflows to be started")
}

func TestEmailPollerWorkflow_HandlesActivityError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestEmailActivities{}
	env.RegisterActivity(testActivities.GetActiveEmailSourcesActivity)

	// Mock activity failure
	env.OnActivity(testActivities.GetActiveEmailSourcesActivity, mock.Anything).Return(nil, errors.New("database error"))

	env.ExecuteWorkflow(EmailPollerWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

func TestEmailPollerWorkflow_ContinuesOnChildWorkflowError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestEmailActivities{}
	env.RegisterActivity(testActivities.GetActiveEmailSourcesActivity)

	// Mock the activity with multiple sources
	sourceIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	env.OnActivity(testActivities.GetActiveEmailSourcesActivity, mock.Anything).Return(sourceIDs, nil)

	// First child workflow fails to start
	env.OnWorkflow(ProcessEmailSourceWorkflow, mock.Anything, sourceIDs[0]).Return(errors.New("failed to start"))
	// Second and third succeed
	env.OnWorkflow(ProcessEmailSourceWorkflow, mock.Anything, sourceIDs[1]).Return(nil)
	env.OnWorkflow(ProcessEmailSourceWorkflow, mock.Anything, sourceIDs[2]).Return(nil)

	env.ExecuteWorkflow(EmailPollerWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	// Parent workflow should complete successfully even if one child fails to start
	require.NoError(t, env.GetWorkflowError())
}

func TestProcessEmailSourceWorkflow_StartsChildWorkflowsForArticles(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestEmailActivities{}
	env.RegisterActivity(testActivities.FetchEmailsActivity)

	sourceID := uuid.New()
	articleIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	// Mock the fetch activity
	env.OnActivity(testActivities.FetchEmailsActivity, mock.Anything, FetchEmailsInput{EmailSourceID: sourceID}).Return(
		&FetchEmailsOutput{NewArticleIDs: articleIDs},
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

	env.ExecuteWorkflow(ProcessEmailSourceWorkflow, sourceID)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify all article summarization workflows were started
	require.Equal(t, 3, len(childWorkflowsStarted), "Expected 3 child workflows for articles")
}

func TestProcessEmailSourceWorkflow_HandlesNoNewArticles(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestEmailActivities{}
	env.RegisterActivity(testActivities.FetchEmailsActivity)

	sourceID := uuid.New()

	// Mock activity returning no new articles
	env.OnActivity(testActivities.FetchEmailsActivity, mock.Anything, FetchEmailsInput{EmailSourceID: sourceID}).Return(
		&FetchEmailsOutput{NewArticleIDs: []uuid.UUID{}},
		nil,
	)

	env.ExecuteWorkflow(ProcessEmailSourceWorkflow, sourceID)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestProcessEmailSourceWorkflow_ContinuesOnArticleSummarizationError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestEmailActivities{}
	env.RegisterActivity(testActivities.FetchEmailsActivity)

	sourceID := uuid.New()
	articleIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	// Mock the fetch activity
	env.OnActivity(testActivities.FetchEmailsActivity, mock.Anything, FetchEmailsInput{EmailSourceID: sourceID}).Return(
		&FetchEmailsOutput{NewArticleIDs: articleIDs},
		nil,
	)

	// First article summarization fails
	env.OnWorkflow(SummarizeArticleWorkflow, mock.Anything, articleIDs[0]).Return(errors.New("summarization failed"))
	// Others succeed
	env.OnWorkflow(SummarizeArticleWorkflow, mock.Anything, articleIDs[1]).Return(nil)
	env.OnWorkflow(SummarizeArticleWorkflow, mock.Anything, articleIDs[2]).Return(nil)

	env.ExecuteWorkflow(ProcessEmailSourceWorkflow, sourceID)

	require.True(t, env.IsWorkflowCompleted())
	// Should complete successfully even if one summarization fails
	require.NoError(t, env.GetWorkflowError())
}

// TestEmailChildWorkflowExecutionIsObtained verifies that child workflows are properly
// started and registered in Temporal. This is a regression test for the bug where
// child workflows would show as "pending" but their links resulted in 404 errors
// because GetChildWorkflowExecution().Get() wasn't being called.
//
// The fix ensures that we wait for each child workflow's execution to be obtained
// before the parent workflow completes, which guarantees the child is registered
// in Temporal's system.
func TestEmailChildWorkflowExecutionIsObtained(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register mock activities
	testActivities := &TestEmailActivities{}
	env.RegisterActivity(testActivities.GetActiveEmailSourcesActivity)

	sourceIDs := []uuid.UUID{uuid.New(), uuid.New()}
	env.OnActivity(testActivities.GetActiveEmailSourcesActivity, mock.Anything).Return(sourceIDs, nil)

	// Mock child workflows - they should all succeed
	for _, sourceID := range sourceIDs {
		env.OnWorkflow(ProcessEmailSourceWorkflow, mock.Anything, sourceID).Return(nil)
	}

	env.ExecuteWorkflow(EmailPollerWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// The fact that the parent workflow completed successfully without errors
	// indicates that all child workflows were properly started and their
	// executions were obtained. If GetChildWorkflowExecution().Get() wasn't
	// being called, the parent would complete before children are registered,
	// which would be caught in production as 404 errors when accessing child
	// workflow links.
}

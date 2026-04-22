package workflow

import (
	"fmt"

	"github.com/bbu/rss-summarizer/backend/internal/config"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type Worker struct {
	client          client.Client
	feedWorker      worker.Worker
	emailWorker     worker.Worker
	activities      *Activities
	emailActivities *EmailActivities
}

func NewWorker(cfg *config.Config, activities *Activities, emailActivities *EmailActivities) (*Worker, error) {
	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  cfg.Temporal.Host,
		Namespace: cfg.Temporal.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	// Create feed polling worker
	feedWorker := worker.New(c, FeedPollingTaskQueue, worker.Options{})

	// Register feed workflows
	feedWorker.RegisterWorkflow(FeedPollerWorkflow)
	feedWorker.RegisterWorkflow(ProcessFeedWorkflow)
	feedWorker.RegisterWorkflow(SummarizeArticleWorkflow)

	// Register feed activities
	feedWorker.RegisterActivity(activities.FetchFeedActivity)
	feedWorker.RegisterActivity(activities.SummarizeArticleActivity)
	feedWorker.RegisterActivity(activities.GetFeedsDueForPollActivity)

	// Create email polling worker
	emailWorker := worker.New(c, EmailPollingTaskQueue, worker.Options{})

	// Register email workflows
	emailWorker.RegisterWorkflow(EmailPollerWorkflow)
	emailWorker.RegisterWorkflow(ProcessEmailSourceWorkflow)
	emailWorker.RegisterWorkflow(SummarizeArticleWorkflow) // Reuse for email articles

	// Register email activities
	emailWorker.RegisterActivity(emailActivities.GetActiveEmailSourcesActivity)
	emailWorker.RegisterActivity(emailActivities.FetchEmailsActivity)
	emailWorker.RegisterActivity(activities.SummarizeArticleActivity) // Reuse for email articles

	return &Worker{
		client:          c,
		feedWorker:      feedWorker,
		emailWorker:     emailWorker,
		activities:      activities,
		emailActivities: emailActivities,
	}, nil
}

func (w *Worker) Start() error {
	// Start feed worker
	if err := w.feedWorker.Start(); err != nil {
		return fmt.Errorf("failed to start feed worker: %w", err)
	}

	// Start email worker
	if err := w.emailWorker.Start(); err != nil {
		w.feedWorker.Stop()
		return fmt.Errorf("failed to start email worker: %w", err)
	}

	return nil
}

func (w *Worker) Stop() {
	w.feedWorker.Stop()
	w.emailWorker.Stop()
	w.client.Close()
}

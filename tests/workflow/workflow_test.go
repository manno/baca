package workflow_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/manno/baca/internal/change"
	"github.com/manno/baca/internal/workflow"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWorkflow(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Workflow Suite")
}

var _ = Describe("Workflow Agent", func() {
	var (
		ctx    context.Context
		logger *slog.Logger
		agent  *workflow.Agent
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		agent = workflow.NewAgent("copilot", logger)
	})

	Context("Session Management", func() {
		It("creates a new session with initial prompt", func() {
			session := workflow.NewSession("test prompt")
			Expect(session).NotTo(BeNil())
			Expect(session.InitialPrompt).To(Equal("test prompt"))
			Expect(session.IsComplete()).To(BeFalse())
		})

		It("adds messages to conversation log", func() {
			session := workflow.NewSession("test prompt")
			session.AddMessage("agent", "What files?")
			session.AddMessage("user", "main.go")

			Expect(len(session.ConversationLog)).To(Equal(2))
			Expect(session.ConversationLog[0].Role).To(Equal("agent"))
			Expect(session.ConversationLog[1].Role).To(Equal("user"))
		})

		It("adds context to session", func() {
			session := workflow.NewSession("test prompt")
			session.AddContext(workflow.Context{
				Source:  "manual",
				Type:    "note",
				Content: "test context",
			})

			Expect(len(session.GatheredContext)).To(Equal(1))
			Expect(session.GatheredContext[0].Source).To(Equal("manual"))
		})

		It("marks session complete when refined prompt is set", func() {
			session := workflow.NewSession("test prompt")
			Expect(session.IsComplete()).To(BeFalse())

			session.SetRefinedPrompt("refined prompt")
			Expect(session.IsComplete()).To(BeTrue())
			Expect(session.RefinedPrompt).To(Equal("refined prompt"))
		})
	})

	Context("Prompt Refinement", func() {
		It("refines prompt with conversation history", func() {
			session := workflow.NewSession("Fix the bug")
			session.AddMessage("agent", "Which file?")
			session.AddMessage("user", "main.go")

			refiner := workflow.NewRefiner("copilot")
			refined, err := refiner.RefinePrompt(ctx, session)

			Expect(err).NotTo(HaveOccurred())
			Expect(refined).To(ContainSubstring("Fix the bug"))
			Expect(refined).To(ContainSubstring("main.go"))
		})

		It("includes gathered context in refined prompt", func() {
			session := workflow.NewSession("Update function")
			session.AddContext(workflow.Context{
				Source:  "github",
				Type:    "issue",
				URL:     "https://github.com/org/repo/issues/123",
				Content: "Need to handle edge case",
			})

			refiner := workflow.NewRefiner("copilot")
			refined, err := refiner.RefinePrompt(ctx, session)

			Expect(err).NotTo(HaveOccurred())
			Expect(refined).To(ContainSubstring("Update function"))
			Expect(refined).To(ContainSubstring("edge case"))
			Expect(refined).To(ContainSubstring("github"))
		})

		It("generates clarifying questions", func() {
			refiner := workflow.NewRefiner("copilot")
			questions := refiner.GenerateQuestions("Fix the bug")

			Expect(len(questions)).To(BeNumerically(">", 0))
			Expect(questions[0]).To(ContainSubstring("files"))
		})
	})

	Context("Non-Interactive Mode", func() {
		It("passes through change without modification", func() {
			ch := &change.Change{
				Kind:       "Change",
				APIVersion: "v1",
				Spec: change.ChangeSpec{
					Prompt: "original prompt",
					Repos:  []string{"https://github.com/org/repo"},
					Agent:  "copilot-cli",
				},
			}

			refined, err := agent.RunNonInteractive(ctx, ch)

			Expect(err).NotTo(HaveOccurred())
			Expect(refined).To(Equal(ch))
			Expect(refined.Spec.Prompt).To(Equal("original prompt"))
		})
	})
})

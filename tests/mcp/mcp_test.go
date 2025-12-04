package mcp_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/manno/baca/internal/mcp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMCP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MCP Suite")
}

var _ = Describe("MCP Manager", func() {
	var (
		logger  *slog.Logger
		manager *mcp.Manager
	)

	BeforeEach(func() {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		manager = mcp.NewManager(logger)
	})

	Context("Manager Operations", func() {
		It("creates a new manager", func() {
			Expect(manager).NotTo(BeNil())
		})

		It("registers clients", func() {
			ghClient := mcp.NewGitHubClient(logger)
			manager.RegisterClient(ghClient)

			client, ok := manager.GetClient(mcp.SourceGitHub)
			Expect(ok).To(BeTrue())
			Expect(client).NotTo(BeNil())
			Expect(client.Name()).To(Equal(mcp.SourceGitHub))
		})

		It("returns false for unregistered clients", func() {
			_, ok := manager.GetClient(mcp.SourceSlack)
			Expect(ok).To(BeFalse())
		})
	})

	Context("Source Parsing", func() {
		It("parses single source", func() {
			sources, err := mcp.ParseSources("github")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(sources)).To(Equal(1))
			Expect(sources[0]).To(Equal(mcp.SourceGitHub))
		})

		It("parses multiple sources", func() {
			sources, err := mcp.ParseSources("github,slack")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(sources)).To(Equal(2))
			Expect(sources).To(ContainElement(mcp.SourceGitHub))
			Expect(sources).To(ContainElement(mcp.SourceSlack))
		})

		It("handles whitespace in source list", func() {
			sources, err := mcp.ParseSources(" github , slack ")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(sources)).To(Equal(2))
		})

		It("returns error for unknown source", func() {
			_, err := mcp.ParseSources("unknown")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown MCP source"))
		})

		It("returns empty for empty string", func() {
			sources, err := mcp.ParseSources("")
			Expect(err).NotTo(HaveOccurred())
			Expect(sources).To(BeNil())
		})
	})

	Context("GitHub Client", func() {
		var ghClient *mcp.GitHubClient

		BeforeEach(func() {
			ghClient = mcp.NewGitHubClient(logger)
		})

		It("creates GitHub client", func() {
			Expect(ghClient).NotTo(BeNil())
			Expect(ghClient.Name()).To(Equal(mcp.SourceGitHub))
		})

		It("checks if gh CLI is available", func() {
			available := ghClient.IsAvailable()
			_ = available
		})
	})

	Context("Context Gathering", func() {
		It("handles empty sources list", func() {
			items, err := manager.GatherContext([]mcp.Source{}, "test query", []string{"https://github.com/example/repo"})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(items)).To(Equal(0))
		})

		It("skips unavailable clients", func() {
			ghClient := mcp.NewGitHubClient(logger)
			manager.RegisterClient(ghClient)

			items, err := manager.GatherContext(
				[]mcp.Source{mcp.SourceGitHub},
				"test query",
				[]string{"https://github.com/example/repo"},
			)
			Expect(err).NotTo(HaveOccurred())
			_ = items
		})
	})
})

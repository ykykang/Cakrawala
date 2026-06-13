package main

import (
	"cakrawala/config"
	"cakrawala/internal/fetcher"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "cakrawala",
		Short: "Automated IDX morning brief to Obsidian",
	}

	var cfgPath string
	root.PersistentFlags().StringVarP(&cfgPath, "config", "c", "config.yaml", "path to config.yaml")

	root.AddCommand(newRunCmd(&cfgPath), newStartCmd(&cfgPath))

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRunCmd(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run pipeline once",
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			dateStr, _ := cmd.Flags().GetString("date")

			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}

			target := time.Now()
			if dateStr != "" {
				target, err = time.Parse("2006-01-02", dateStr)
				if err != nil {
					return fmt.Errorf("invalid --date %q, use YYYY-MM-DD", dateStr)
				}
			}

			return runPipeline(cfg, target, target, dryRun)
		},
	}
	cmd.Flags().Bool("dry-run", false, "run pipeline without writing to vault")
	cmd.Flags().String("date", "", "fetch specific date (YYYY-MM-DD), default today")
	return cmd
}

func newStartCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start daemon — runs pipeline on cron schedule",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}

			c := cron.New(cron.WithLocation(time.FixedZone("WIB", 7*3600)))
			if _, err = c.AddFunc(cfg.CronSchedule, func() {
				log.Println("cron: pipeline start")
				today := time.Now()
				if err := runPipeline(cfg, today, today, false); err != nil {
					log.Printf("cron: pipeline error: %v", err)
				}
			}); err != nil {
				return fmt.Errorf("invalid cron schedule %q: %w", cfg.CronSchedule, err)
			}

			c.Start()
			log.Printf("daemon started — schedule: %s (WIB)", cfg.CronSchedule)

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			<-quit

			log.Println("shutting down...")
			<-c.Stop().Done()
			return nil
		},
	}
}

func runPipeline(cfg *config.Config, from, to time.Time, dryRun bool) error {
	ctx := context.Background()

	if dryRun {
		log.Println("dry-run mode — vault write skipped")
	}

	log.Printf("fetching IDX disclosures %s → %s...", from.Format("2006-01-02"), to.Format("2006-01-02"))
	disclosures, err := fetcher.NewIDXFetcher().FetchDisclosures(ctx, from, to)
	if err != nil {
		return fmt.Errorf("IDX fetch: %w", err)
	}
	log.Printf("fetched %d disclosures", len(disclosures))

	// TODO: articles, err := fetcher.NewNewsFetcher(cfg).FetchAll(ctx)
	// TODO: classified, err := classifier.New(cfg).Classify(ctx, disclosures, articles)
	// TODO: brief, err := merger.New().Merge(classified)

	if !dryRun {
		log.Println("writing to vault...")
		// TODO: err = writer.New(cfg).Write(ctx, brief)
	}

	_ = disclosures
	return nil
}

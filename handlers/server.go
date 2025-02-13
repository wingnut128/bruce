package handlers

import (
	"bruce/config"
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

func RunServer(svr_config string) error {
	log.Debug().Msg("starting server task")

	// Read server configuration
	sc := &config.ServerConfig{}
	err := config.ReadServerConfig(svr_config, sc)
	if err != nil {
		log.Error().Err(err).Msg("cannot continue without configuration data")
		os.Exit(1)
	}

	// Validate that the default action exists
	defaultFound := false
	for _, e := range sc.Execution {
		if e.Action == "default" {
			defaultFound = true
			break
		}
	}
	if !defaultFound {
		log.Error().Msg("default action not found in configuration")
		os.Exit(1)
	}

	// Separate cadence and event executions
	var eventExecutions []config.Execution

	// Channel to receive OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// WaitGroup to manage the lifecycle of goroutines
	var wg sync.WaitGroup

	// Context to manage cancellation of goroutines
	ctx, cancel := context.WithCancel(context.Background())

	log.Info().Msg("Starting Bruce in server mode")

	// Loop through the executions and handle cadence runners and event executions
	for _, e := range sc.Execution {
		// Start CadenceRunner for "cadence" type executions
		if e.Type == "cadence" {
			wg.Add(1)
			go func(e config.Execution) {
				defer wg.Done()
				CadenceRunner(ctx, e.Name, e.Target, e.PrivKey, e.Cadence)
			}(e)
		} else if e.Type == "event" {
			// Add "event" type executions to the list for the SocketRunner
			eventExecutions = append(eventExecutions, e)
		} else {
			log.Info().Msgf("Skipping invalid execution target: %s must be of type 'event' or 'cadence'", e.Name)
		}
	}

	// Start the SocketRunner only if there are event executions
	if len(eventExecutions) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SocketRunner(ctx, sc.Endpoint, sc.RunnerID, sc.Authorization, eventExecutions)
		}()
	} else {
		log.Info().Msg("No event executions found, skipping SocketRunner initialization")
	}

	// Wait for a signal to shut down
	<-sigCh
	log.Info().Msg("Shutting down server...")

	// Cancel all goroutines
	cancel()

	// Wait for all runners to finish
	wg.Wait()

	log.Info().Msg("All runners finished, server shut down successfully.")
	return nil
}

func CadenceRunner(ctx context.Context, name, propfile, key string, cadence int) {
	log.Debug().Msgf("Starting CadenceRunner[%s] with propfile: %s, every %d minutes", name, propfile, cadence)
	t, err := config.LoadConfig(propfile, key)
	if err != nil {
		log.Error().Err(err).Msgf("cannot continue without configuration data, runner %s failed", name)
		return
	}

	// Run the task at the specified cadence interval
	ticker := time.NewTicker(time.Duration(cadence) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msgf("CadenceRunner[%s] received cancellation signal, exiting...", name)
			return
		case <-ticker.C:
			log.Debug().Msgf("CadenceRunner[%s] running execution steps", name)
			err = ExecuteSteps(t)
			if err != nil {
				log.Error().Err(err).Msgf("CadenceRunner[%s] failed", name)
				return
			}
			log.Info().Msgf("CadenceRunner[%s] execution succeeded", name)
		}
	}
}

func ExecuteSteps(t *config.TemplateData) error {
	for idx, step := range t.Steps {
		if step.Action != nil {
			err := step.Action.Execute()
			if err != nil {
				log.Error().Err(err).Msgf("Error executing step [%d]", idx+1)
				return err
			}
		}
	}
	return nil
}

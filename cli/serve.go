package cli

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/stan-ley-tech/ByteForge/internal/api"
	"github.com/stan-ley-tech/ByteForge/internal/httpclient"
	"github.com/stan-ley-tech/ByteForge/internal/runner"
	"github.com/stan-ley-tech/ByteForge/internal/storage"
)

func newServeCommand() *cobra.Command {
	var addr, dbPath, staticDir string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the ByteForge API server and web UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return serve(addr, dbPath, staticDir)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":8080", "address to listen on")
	cmd.Flags().StringVar(&dbPath, "db", "byteforge.db", "path to the SQLite database file")
	cmd.Flags().StringVar(&staticDir, "static", "web/dist", "directory containing the built web UI")
	return cmd
}

func serve(addr, dbPath, staticDir string) error {
	store, err := storage.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer store.Close()

	rn := runner.New(httpclient.New(httpclient.DefaultConfig()))
	apiServer := api.NewServer(store, rn)

	mux := http.NewServeMux()
	mux.Handle("/api/", apiServer)
	mux.Handle("/healthz", apiServer)
	mux.Handle("/", spaHandler(staticDir))

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("byteforge listening", "addr", addr, "db", dbPath)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	}
}

// spaHandler serves the built web UI, falling back to index.html for any
// path that isn't a real static asset so client-side routing works on a
// hard refresh. If staticDir doesn't exist (a plain `go run` without first
// building the frontend), it reports that clearly instead of a raw 404.
func spaHandler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			http.Error(w, "web UI is not built; the API is still available under /api", http.StatusNotFound)
			return
		}

		requested := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if stat, err := os.Stat(requested); err == nil && !stat.IsDir() {
			http.ServeFile(w, r, requested)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}

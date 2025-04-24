package main

import (
	"fmt"
	"gitgud/config"
	"gitgud/git"
	"log"
	"log/slog"
	"net/http"
	"slices"
	"strings"
)

func main() {
	handler := GetRouter()
	server := http.Server{
		Addr:    "0.0.0.0:1323",
		Handler: handler,
	}

	slog.Info("Listening on port: 1323")
	err := server.ListenAndServe()
	if err != nil {
		slog.Error("Server closed", "error", err)
	}
}

func GetRouter() *http.ServeMux {
	router := http.NewServeMux()
	router.Handle("GET /{orgName}/{repositoryName}/info/refs", errorHandler(GetServiceHandler))
	router.Handle("POST /{orgName}/{repositoryName}/{service}", errorHandler(PostServiceHandler))
	return router
}

func PostServiceHandler(writer http.ResponseWriter, request *http.Request) error {
	repositoryName := request.PathValue("repositoryName")
	orgName := request.PathValue("orgName")

	if !strings.HasSuffix(repositoryName, ".git") {
		return fmt.Errorf("invalid repository name %s", repositoryName)
	}

	repositoryName = strings.ReplaceAll(repositoryName, ".git", "")

	remoteRepo, err := git.NewRemoteRepository(config.Settings.BaseURL, orgName, repositoryName)
	if err != nil {
		return err
	}

	logWriter := LogWriter{writer}

	service := request.PathValue("service")
	if !slices.Contains([]string{"git-upload-pack", "git-receive-pack"}, service) {
		return fmt.Errorf("unexpected service: %s", service)
	}

	writer.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-result", service))

	command := remoteRepo.CallService(service, false)

	// Passing the body from the request into the git service command
	command.Stdin = request.Body
	command.Stdout = logWriter

	var stdErr strings.Builder
	command.Stderr = &stdErr

	err = command.Run()

	if err != nil {
		return fmt.Errorf("failure calling service %s: %w", service, err)
	}

	return err
}

func GetServiceHandler(writer http.ResponseWriter, request *http.Request) error {
	repositoryName := request.PathValue("repositoryName")
	orgName := request.PathValue("orgName")

	if !strings.HasSuffix(repositoryName, ".git") {
		return fmt.Errorf("invalid repository name %s", repositoryName)
	}

	repositoryName = strings.ReplaceAll(repositoryName, ".git", "")

	remoteRepo, err := git.NewRemoteRepository(config.Settings.BaseURL, orgName, repositoryName)
	if err != nil {
		return err
	}

	logWriter := LogWriter{writer}

	service := request.URL.Query().Get("service")
	if !slices.Contains([]string{"git-upload-pack", "git-receive-pack"}, service) {
		return fmt.Errorf("unexpected service: %s", service)
	}

	writer.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))

	// Write the service when advertising
	fmt.Fprintf(logWriter, "%04x# service=%s\n", len("# service="+service)+5, service)
	logWriter.Write([]byte("0000"))

	command := remoteRepo.CallService(service, true)

	var stdErr strings.Builder
	command.Stdout = logWriter
	command.Stderr = &stdErr

	err = command.Run()

	if err != nil {
		return fmt.Errorf("failure calling service %s: %w", service, err)
	}

	return err
}

type errorHandler func(http.ResponseWriter, *http.Request) error

func (fn errorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := fn(w, r)
	if err != nil {
		slog.Error("Unexpected error in ServeHTTP", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type LogWriter struct {
	ResponseWriter http.ResponseWriter
}

func (w LogWriter) Write(p []byte) (n int, err error) {
	log.Println("writing: ", string(p))
	return w.ResponseWriter.Write(p)
}

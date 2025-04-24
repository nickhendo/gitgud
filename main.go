package main

import (
	"fmt"
	"gitgud/config"
	"log"
	"log/slog"
	"net/http"
	"os/exec"
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
	router.Handle("GET /{orgName}/{repositoryName}/info/refs", errorHandler(GetRepoHandler))
	router.Handle("POST /{orgName}/{repositoryName}/{service}", errorHandler(PostServiceHandler))
	return router
}

func PostServiceHandler(writer http.ResponseWriter, request *http.Request) error {
	service := request.PathValue("service")
	repositoryName := request.PathValue("repositoryName")
	orgName := request.PathValue("orgName")
	repoPath := fmt.Sprintf("%s/%s/%s", config.Settings.RepositoriesLocation, orgName, repositoryName)

	var cmd *exec.Cmd
	logWriter := LogWriter{writer}

	switch service {
	case "git-upload-pack":
		writer.Header().Set("Content-Type", "application/x-git-upload-pack-result")
		cmd = exec.Command("git", "upload-pack", "--stateless-rpc", "--strict", repoPath)
	case "git-receive-pack":
		writer.Header().Set("Content-Type", "application/x-git-receive-pack-result")
		writer.WriteHeader(http.StatusOK)
		cmd = exec.Command("git", "receive-pack", "--stateless-rpc", repoPath)
	default:
		return fmt.Errorf("unexpected service: %s", service)
	}

	cmd.Env = append(cmd.Env, "GIT_PROTOCOL=version=2")

	cmd.Stdin = request.Body
	cmd.Stdout = logWriter

	var stdErr strings.Builder
	cmd.Stderr = &stdErr

	err := cmd.Run()
	if err != nil {
		log.Printf("%s error: %v, stderr: %s", service, err, stdErr.String())
	}

	writer.WriteHeader(http.StatusOK)
	return err
}

func GetRepoHandler(writer http.ResponseWriter, request *http.Request) error {
	fmt.Printf("request.Method: %v\n", request.Method)
	repositoryName := request.PathValue("repositoryName")
	orgName := request.PathValue("orgName")
	service := request.URL.Query().Get("service")
	repoPath := fmt.Sprintf("%s/%s/%s", config.Settings.RepositoriesLocation, orgName, repositoryName)

	var cmd *exec.Cmd

	customWriter := LogWriter{writer}

	switch service {
	case "git-upload-pack":
		writer.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		cmd = exec.Command("git", "upload-pack", "--stateless-rpc", "--advertise-refs", repoPath)
	case "git-receive-pack":
		writer.Header().Set("Content-Type", "application/x-git-receive-pack-advertisement")
		cmd = exec.Command("git", "receive-pack", "--stateless-rpc", "--advertise-refs", repoPath)
	default:
		return fmt.Errorf("unexpected service: %s", service)
	}
	cmd.Env = append(cmd.Env, "GIT_PROTOCOL=version=2")

	// Write the "# service=git-upload-pack" header in pkt-line format
	fmt.Fprintf(customWriter, "%04x# service=%s\n", len("# service="+service)+5, service)
	customWriter.Write([]byte("0000"))

	cmd.Stdout = customWriter

	var stdErr strings.Builder
	cmd.Stderr = &stdErr

	err := cmd.Run()
	if err != nil {
		log.Printf("%s error: %v, stderr: %s", service, err, stdErr.String())
	}

	writer.WriteHeader(http.StatusOK)
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

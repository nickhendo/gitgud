package main

import (
	"fmt"
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

type errorHandler func(http.ResponseWriter, *http.Request) error

func (fn errorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := fn(w, r)
	if err != nil {
		slog.Error("Unexpected error in ServeHTTP", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func GetRouter() *http.ServeMux {
	router := http.NewServeMux()
	router.Handle("GET /repos/test/{repositoryName}/info/refs", errorHandler(GetRepoHandler))
	router.Handle("POST /repos/test/{repositoryName}/{service}", errorHandler(PostServiceHandler))
	return router
}

func PostServiceHandler(writer http.ResponseWriter, request *http.Request) error {
	service := request.PathValue("service")
	repositoryName := request.PathValue("repositoryName")
	repoLocation := "repositories"
	repoPath := fmt.Sprintf("%s/%s", repoLocation, repositoryName)

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
	service := request.URL.Query().Get("service")
	repoLocation := "repositories"
	repoPath := fmt.Sprintf("%s/%s", repoLocation, repositoryName)

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

func CreateRepo(repoName string) error {
	if !strings.HasSuffix(repoName, ".git") {
		return fmt.Errorf("bare repository name must end with '.git'")
	}

	if strings.Contains(repoName, " ") {
		return fmt.Errorf("bare repository name must not contain spaces")
	}
	defaultBranchName := "main"
	repoLocation := "repositories"

	repoPath := strings.Join([]string{repoLocation, repoName}, "/")
	slog.Debug("creating repository at", "path", repoPath)
	command := exec.Command("git", "init", repoPath, "--bare")
	slog.Debug("command.Args", "value", strings.Join(command.Args, ","))

	var output strings.Builder
	command.Stdout = &output
	err := command.Run()
	slog.Debug(output.String())
	if err != nil {
		return err
	}

	command2 := exec.Command(
		"git",
		"symbolic-ref",
		"HEAD",
		fmt.Sprintf("refs/heads/%s", defaultBranchName),
	)
	command2.Dir = repoPath
	slog.Debug("command2", "value", command2.String())
	var output2 strings.Builder
	command2.Stdout = &output2
	err = command2.Run()
	slog.Debug("output2", "value", output2.String())

	return err
}

func DeleteRepo(repoName string) error {
	repoLocation := "repositories"

	input := strings.Join([]string{repoLocation, repoName}, "/")
	slog.Debug("attempting to delete repository", "path", input)
	command := exec.Command("rm", "-rf", input)
	var output strings.Builder
	command.Stdout = &output
	err := command.Run()
	slog.Debug("command", "output", output.String())
	return err
}

type LogWriter struct {
	ResponseWriter http.ResponseWriter
}

func (w LogWriter) Write(p []byte) (n int, err error) {
	log.Println("writing: ", string(p))
	return w.ResponseWriter.Write(p)
}

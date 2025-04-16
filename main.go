package main

import (
	"log"
	"os/exec"
	"strings"
)

func CreateRepo(repoName string) error {
	defaultBranchName := "main"
	repoLocation := "repositories"

	input := strings.Join([]string{repoLocation, repoName}, "/")
	log.Println(input)
	command := exec.Command("git", "init", "-b", defaultBranchName, input)
	var output strings.Builder
	command.Stdout = &output
	err := command.Run()
	log.Println(output.String())
	return err
}

func DeleteRepo(repoName string) error {
	repoLocation := "repositories"

	input := strings.Join([]string{repoLocation, repoName}, "/")
	log.Println(input)
	command := exec.Command("rm", "-rf", input)
	var output strings.Builder
	command.Stdout = &output
	err := command.Run()
	log.Println(output.String())
	return err
}

func main() {
	err := CreateRepo("test_repo")
	if err != nil {
		log.Fatal(err)
	}
}

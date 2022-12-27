package mycopy

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/julieqiu/derrors"
)

func Run(newRepo, newDir, oldRepo, oldDir string) error {
	if err := validateRequest(oldRepo, oldDir); err != nil {
		return err
	}
	return copyAndEdit(newRepo, newDir, oldRepo, oldDir)
}

// validateRequest checks if repo and dir is a valid Go project repository and
// directory.
func validateRequest(repo, dir string) error {
	url := goDirectoryHeadURL(repo, dir)
	resp, err := http.Head(url)
	if err != nil {
		return fmt.Errorf("http.Get(%q): %v", url, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("Failed to find %q.\n", url)
		return fmt.Errorf("http.Get(%q) returned %d (%q)", url, resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return nil
}

func goRepoURL(repo string) string {
	return fmt.Sprintf("https://go.googlesource.com/%s", repo)
}

func goDirectoryHeadURL(repo, dir string) string {
	return fmt.Sprintf("%s/+/refs/heads/master/%s", goRepoURL(repo), dir)
}

func goDirectoryCommitURL(repo, dir, commit string) string {
	return fmt.Sprintf("%s/+/%s/%s", goRepoURL(repo), commit, dir)
}

func importPath(suffix string) string {
	return fmt.Sprintf("golang.org/x/%s", suffix)
}

// copyAndEdit makes a copy of the package at
// https://go.googlesource.com/<repo>/+/refs/heads/master/<dir> into <dir>.
func copyAndEdit(newRepo, newDir, oldRepo, oldDir string) (err error) {
	tempDir, err := ioutil.TempDir("", "go_")
	if err != nil {
		return err
	}
	defer func() {
		rerr := os.RemoveAll(tempDir)
		if err == nil {
			err = rerr
		}
	}()

	cmd := exec.Command("git", "clone", goRepoURL(oldRepo), tempDir)
	log.Println(strings.Join(cmd.Args, " "))
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("cmd.Run(%q): %v", strings.Join(cmd.Args, " "), err)
	}

	// Copy package into <dir>.
	cmd = exec.Command("cp", "-r", fmt.Sprintf("%s/%s", tempDir, oldDir), newDir)
	log.Println(strings.Join(cmd.Args, " "))
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("cmd.Run(%q): %v", strings.Join(cmd.Args, " "), err)
	}

	// Edit files to replace all "internal" paths and add header about origin.
	commit := commitForMaster(oldRepo)
	return filepath.Walk(oldDir, func(filename string, info os.FileInfo, err error) error {
		fileInfo, err := os.Stat(filename)
		if err != nil {
			log.Fatalf("os.Stat(%q): %v", filename, err)
		}
		if !fileInfo.IsDir() && filepath.Ext(filename) == ".go" {
			if err := editFile(filename, newRepo, newDir, oldRepo, oldDir, commit); err != nil {
				return err
			}
		}
		return nil
	})
}

// editFile prepends the file with a message to warn users not
// make any edits to the file. It also finds occurrences of "<repo>/internal/<dir>"
// and replaces them with "<new-repo>/internal/<dir>".
func editFile(filename, newRepo, newDir, oldRepo, oldDir, commit string) (err error) {
	defer derrors.Wrap(&err, "editFile(%q, %q, %q, %q, %q, %q)", filename, newRepo, newDir, oldRepo, oldDir, commit)
	log.Printf("Editing: %s", filename)

	// Create a temporary file for writing. This will be renamed at the end of the function.
	wf, err := os.OpenFile(fmt.Sprintf("%s_tmp", filename), os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer wf.Close()

	writer := bufio.NewWriter(wf)
	fmt.Fprintln(writer, "// DO NOT EDIT. This file was copied from")
	fmt.Fprintln(writer, fmt.Sprintf("// %s", goDirectoryCommitURL(oldRepo, oldDir, commit)))
	fmt.Fprintln(writer, "")

	contents, err := readLines(filename)
	if err != nil {
		return err
	}
	pkgToDownload := map[string]bool{}
	internalPath := importPath(oldRepo + "/internal")
	for _, line := range contents {
		if strings.Contains(line, internalPath) {
			pkgToDownload[strings.Fields(line)[0]] = true
			line = replaceImportPath(line, importPath(newRepo+"/"+newDir), internalPath)
			fmt.Println(line)
		}
		fmt.Fprintln(writer, fmt.Sprintf("%s", line))
	}

	if err := writer.Flush(); err != nil {
		return err
	}
	return os.Rename(fmt.Sprintf("%s_tmp", filename), filename)
}

func replaceImportPath(line, newPath, oldPath string) string {
	return strings.ReplaceAll(line, oldPath, newPath)
}

// readLines reads the contents of filename and returns each line of the file.
func readLines(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var contents []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		contents = append(contents, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner.Err: %v", err)
	}
	return contents, nil
}

// commitForMaster returns the commit hash of the current master at origin.
func commitForMaster(repo string) string {
	cmd := exec.Command("git", "ls-remote", goRepoURL(repo), "rev-parse", "HEAD")
	log.Println(strings.Join(cmd.Args, " "))
	out, err := cmd.Output()
	log.Println(string(out))
	if err != nil {
		log.Fatalf("cmd.Run(%q): %v", strings.Join(cmd.Args, " "), err)
	}
	parts := strings.Fields(string(out))
	if len(parts) != 2 {
		log.Fatalf("Unexpected output: %q", string(out))
	}
	return parts[0][0:8]
}

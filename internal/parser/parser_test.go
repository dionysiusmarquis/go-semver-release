package parser

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v2/internal/rule"
)

var fakeLogger = zerolog.New(io.Discard)

func TestParser_CommitTypeRegex(t *testing.T) {
	assert := assert.New(t)

	type test struct {
		commit     string
		commitType string
	}

	matrix := []test{
		{"feat: implemented foo", "feat"},
		{"fix(foo.js): fixed foo", "fix"},
		{"chore(api): fixed doc typos", "chore"},
		{"test(../tests/): implemented unit tests", "test"},
		{"ci(ci.yaml): added stages to pipeline", "ci"},
	}

	for _, item := range matrix {
		got := conventionalCommitRegex.FindStringSubmatch(item.commit)[1]

		assert.Equal(item.commitType, got, "commit type should be equal")
	}
}

func TestParser_BreakingChangeRegex(t *testing.T) {
	assert := assert.New(t)

	type test struct {
		commit     string
		isBreaking bool
	}

	matrix := []test{
		{"feat: implemented foo", false},
		{"fix(foo.js)!: fixed foo", true},
		{"chore(docs): fixed doc typos BREAKING CHANGE: delete some APIs", true},
	}

	for _, item := range matrix {
		submatch := conventionalCommitRegex.FindStringSubmatch(item.commit)
		got := strings.Contains(submatch[3], "!") || strings.Contains(submatch[0], "BREAKING CHANGE")

		assert.Equal(item.isBreaking, got, "breaking change should be equal")
	}
}

func TestParser_FetchLatestSemverTagWithNoTag(t *testing.T) {
	assert := assert.New(t)

	r, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	assert.NoError(err, "should have been able to create Git repository")

	defer func(path string) {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "should have been able to remove Git repository")
	}(repositoryPath)

	rules, err := rule.Init()
	assert.NoError(err, "should have been able to create rule reader")

	parser := New(fakeLogger, rules)

	latest, err := parser.fetchLatestSemverTag(r)
	assert.NoError(err, "should have been able to fetch latest semver tag")

	assert.Nil(latest, "latest semver tag should be nil")
}

func TestParser_FetchLatestSemverTagWithOneTag(t *testing.T) {
	assert := assert.New(t)

	r, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	assert.NoError(err, "should have been able to create Git repository")

	defer func(path string) {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "should have been able to remove Git repository")
	}(repositoryPath)

	h, err := r.Head()
	assert.NoError(err, "should have been able to get HEAD")

	tag := "1.0.0"

	_, err = r.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
		Message: tag,
		Tagger: &object.Signature{
			Name:  "Go Semver Release",
			Email: "ci@ci.ci",
			When:  time.Now(),
		},
	})
	assert.NoError(err, "should have been able to create tag")

	rules, err := rule.Init()
	assert.NoError(err, "should have been able to parse rule")

	parser := New(fakeLogger, rules)

	latest, err := parser.fetchLatestSemverTag(r)
	assert.NoError(err, "should have been able to fetch latest semver tag")

	assert.Equal(tag, latest.Name, "latest semver tag should be equal")
}

func TestParser_FetchLatestSemverTagWithMultipleTags(t *testing.T) {
	assert := assert.New(t)

	r, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	assert.NoError(err, "should have been able to create Git repository")

	defer func(path string) {
		err := os.RemoveAll(path)
		assert.NoError(err, "should have been able to remove Git repository")
	}(repositoryPath)

	h, err := r.Head()
	assert.NoError(err, "should have been able to get HEAD")

	tags := []string{"2.0.0", "2.0.1", "3.0.0", "2.5.0", "0.0.2", "0.0.1", "0.1.0", "1.0.0"}

	for i, tag := range tags {
		_, err := r.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
			Message: tag,
			Tagger: &object.Signature{
				Name:  "Go Semver Release",
				Email: "ci@ci.ci",
				When:  time.Now().Add(time.Duration(i) * time.Hour),
			},
		})
		assert.NoError(err, "should have been able to create tag")
	}

	rules, err := rule.Init()
	assert.NoError(err, "should have been able to parse rule")

	commitAnalyzer := New(fakeLogger, rules)

	latest, err := commitAnalyzer.fetchLatestSemverTag(r)
	assert.NoError(err, "should have been able to fetch latest semver tag")

	want := "3.0.0"
	assert.Equal(want, latest.Name, "latest semver tag should be equal")
}

func TestParser_ComputeNewSemverNumberWithUntaggedRepositoryWithoutNewRelease(t *testing.T) {
	assert := assert.New(t)

	r, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	assert.NoError(err, "should have been able to create Git repository")

	defer func(path string) {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "should have able to remove Git repository")
	}(repositoryPath)

	rules, err := rule.Init()
	assert.NoError(err, "should have been able to parse rule")

	parser := New(fakeLogger, rules)

	version, _, err := parser.ComputeNewSemver(r)
	assert.NoError(err, "should have been able to compute newsemver")

	want := "0.0.0"

	assert.Equal(want, version.String(), "version should be equal")
}

func TestParser_ComputeNewSemverNumberWithUntaggedRepositoryWitPatchRelease(t *testing.T) {
	assert := assert.New(t)

	r, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	assert.NoError(err, "should have been able to create git repository")

	defer func(path string) {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "should have able to remove git repository")
	}(repositoryPath)

	rules, err := rule.Init()
	assert.NoError(err, "should have been able to parse rule")

	parser := New(fakeLogger, rules)

	version, _, err := parser.ComputeNewSemver(r)
	assert.NoError(err, "should have been able to compute newsemver")

	want := "0.0.1"
	assert.Equal(want, version.String(), "version should be equal")
}

func TestParser_UnknownReleaseType(t *testing.T) {
	assert := assert.New(t)

	r, repositoryPath, err := createGitRepository("fix: commit that trigger an unknown release")
	assert.NoError(err, "should have been able to create git repository")

	defer func(path string) {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "should have able to remove git repository")
	}(repositoryPath)

	rules := rule.ReleaseRules{
		Rules: []rule.ReleaseRule{
			{CommitType: "fix", ReleaseType: "unknown"},
		},
	}

	parser := New(fakeLogger, rules)

	_, _, err = parser.ComputeNewSemver(r)
	assert.Error(err, "should have been failed trying to compute semver")
}

func TestParser_ComputeNewSemverNumberOnUntaggedRepositoryWitMinorRelease(t *testing.T) {
	assert := assert.New(t)

	r, repositoryPath, err := createGitRepository("feat: commit that triggers a minor release")
	assert.NoError(err, "should have been able to create git repository")

	defer func(path string) {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "should have able to remove git repository")
	}(repositoryPath)

	rules, err := rule.Init()
	assert.NoError(err, "should have been able to parse rule")

	parser := New(fakeLogger, rules)

	version, _, err := parser.ComputeNewSemver(r)
	assert.NoError(err, "should have been able to compute newsemver")

	want := "0.1.0"
	assert.Equal(want, version.String(), "version should be equal")
}

func TestParser_ComputeNewSemverNumberOnUntaggedRepositoryWithMajorRelease(t *testing.T) {
	assert := assert.New(t)

	r, repositoryPath, err := createGitRepository("feat!: commit that triggers a major release")
	assert.NoError(err, "should have been able to create git repository")

	defer func(path string) {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "should have able to remove git repository")
	}(repositoryPath)

	err = addCommit(r, "fix: added hello feature")
	assert.NoError(err, "should have able to add git commit")

	rules, err := rule.Init()
	assert.NoError(err, "should have been able to parse rule")

	parser := New(fakeLogger, rules)

	version, newRelease, err := parser.ComputeNewSemver(r)
	assert.NoError(err, "should have been able to compute newsemver")

	want := "1.0.1"

	assert.Equal(want, version.String(), "version should be equal")

	assert.Equal(true, newRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemverOnUninitializedRepository(t *testing.T) {
	assert := assert.New(t)

	dir, err := os.MkdirTemp("", "parser-*")
	if !assert.NoError(err, "failed to create temp. dir.") {
		return
	}

	defer func() {
		err = os.RemoveAll(dir)
		if !assert.NoError(err, "failed to remove temp. dir.") {
			return
		}
	}()

	repository, err := git.PlainInit(dir, false)
	if !assert.NoError(err, "failed to initialize Git repository") {
		return
	}

	rules, err := rule.Init()
	if !assert.NoError(err, "failed to initialize rule") {
		return
	}

	parser := New(fakeLogger, rules)

	_, _, err = parser.ComputeNewSemver(repository)
	assert.ErrorContains(err, "failed to fetch head", "should have been failed trying to fetch latest semver tag from uninitialized repository")
}

func TestParser_ComputeNewSemverOnRepositoryWithNoHead(t *testing.T) {
	assert := assert.New(t)

	tempDirPath, err := os.MkdirTemp("", "tag-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	repository, err := git.PlainInit(tempDirPath, false)
	if err != nil {
		t.Fatalf("failed to init repository: %v", err)
	}

	defaultRules, err := rule.Init()
	if err != nil {
		t.Fatalf("failed to initialize rule: %v", err)
	}

	parser := New(fakeLogger, defaultRules)

	_, _, err = parser.ComputeNewSemver(repository)
	assert.Error(err, "should have been failed trying to compute new semver from repository with no HEAD")
}

func TestParser_ComputeNewSemverWithBuildMetadata(t *testing.T) {

	assert := assert.New(t)

	r, repositoryPath, err := createGitRepository("feat!: commit that triggers a major release")
	assert.NoError(err, "should have been able to create git repository")

	defer func(path string) {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "should have able to remove git repository")
	}(repositoryPath)

	err = addCommit(r, "fix: added hello feature")
	assert.NoError(err, "should have able to add git commit")

	rules, err := rule.Init()
	assert.NoError(err, "should have been able to parse rule")

	parser := New(fakeLogger, rules, WithBuildMetadata("metadata"))

	version, newRelease, err := parser.ComputeNewSemver(r)
	assert.NoError(err, "should have been able to compute newsemver")

	want := "1.0.1+metadata"

	assert.Equal(want, version.String(), "version should be equal")

	assert.Equal(true, newRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemverWithPrerelease(t *testing.T) {

	assert := assert.New(t)

	r, repositoryPath, err := createGitRepository("feat!: commit that triggers a major release")
	assert.NoError(err, "should have been able to create git repository")

	defer func(path string) {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "should have able to remove git repository")
	}(repositoryPath)

	err = addCommit(r, "fix: added hello feature")
	assert.NoError(err, "should have able to add git commit")

	rules, err := rule.Init()
	assert.NoError(err, "should have been able to parse rule")

	parser := New(fakeLogger, rules, WithPrereleaseMode(true), WithPrereleaseSuffix("rc"))

	version, newRelease, err := parser.ComputeNewSemver(r)
	assert.NoError(err, "should have been able to compute newsemver")

	want := "1.0.1-rc"

	assert.Equal(want, version.String(), "version should be equal")

	assert.Equal(true, newRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemverWithPrereleaseAndBuildMetadata(t *testing.T) {

	assert := assert.New(t)

	r, repositoryPath, err := createGitRepository("feat!: commit that triggers a major release")
	assert.NoError(err, "should have been able to create git repository")

	defer func(path string) {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "should have able to remove git repository")
	}(repositoryPath)

	err = addCommit(r, "fix: added hello feature")
	assert.NoError(err, "should have able to add git commit")

	rules, err := rule.Init()
	assert.NoError(err, "should have been able to parse rule")

	parser := New(fakeLogger, rules, WithPrereleaseMode(true), WithPrereleaseSuffix("rc"), WithBuildMetadata("metadata"))

	version, newRelease, err := parser.ComputeNewSemver(r)
	assert.NoError(err, "should have been able to compute newsemver")

	want := "1.0.1-rc+metadata"

	assert.Equal(want, version.String(), "version should be equal")

	assert.Equal(true, newRelease, "boolean should be equal")
}

func TestParser_ShortMessage(t *testing.T) {
	assert := assert.New(t)

	msg := "This is a very long commit message that is over fifty character"
	short := shortenMessage(msg)

	expected := "This is a very long commit message that is over..."

	assert.Equal(expected, short, "short message should be equal")
}

func createGitRepository(firstCommitMessage string) (repository *git.Repository, tempDirPath string, err error) {
	tempDirPath, err = os.MkdirTemp("", "parser-*")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	r, err := git.PlainInit(tempDirPath, false)
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize git repository: %s", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get worktree: %s", err)
	}

	tempFileName := "temp"
	tempFilePath := filepath.Join(tempDirPath, tempFileName)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp file: %s", err)
	}

	defer func() {
		err = tempFile.Close()
		if err != nil {
			return
		}
	}()

	err = os.WriteFile(tempFilePath, []byte("Hello world"), 0o644)
	if err != nil {
		return nil, "", fmt.Errorf("failed to write to temp file: %s", err)
	}

	_, err = w.Add(tempFileName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to add temp file to worktree: %s", err)
	}

	commit, err := w.Commit(firstCommitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver-release@ci.go",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create commit object %s", err)
	}

	_, err = r.CommitObject(commit)
	if err != nil {
		return nil, "", fmt.Errorf("failed to commit object %s", err)
	}

	return r, tempDirPath, nil
}

func addCommit(r *git.Repository, message string) (err error) {
	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("could not get worktree: %w", err)
	}

	tempDirPath, err := os.MkdirTemp("", "commit-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	defer func(path string) {
		err = os.RemoveAll(tempDirPath)
	}(tempDirPath)

	tempFileName := "temp"
	tempFilePath := filepath.Join(tempDirPath, tempFileName)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	defer func() {
		err = tempFile.Close()
	}()

	err = os.WriteFile(tempFilePath, []byte("Hello world"), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	_, err = w.Add(tempFileName)
	if err != nil {
		return fmt.Errorf("failed to add temp file to worktree: %w", err)
	}

	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver-release@ci.go",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	return
}

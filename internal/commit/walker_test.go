package commit

import (
	"os"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/s0ders/go-semver-release/v6/internal/gittest"
	assertion "github.com/stretchr/testify/assert"
)

func TestWalker_SingleBranch(t *testing.T) {
	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = os.RemoveAll(testRepository.Path)
		checkErr(t, err, "removing repository")
	}()

	type e = []string
	steps := []gittest.Step{
		gittest.NewCommitStep("main", "main-1"),
		gittest.NewCallbackStep("", e{
			"main-1",
			"First commit",
		}),

		gittest.NewCommitStep("beta", "beta-1"),
		gittest.NewCommitStep("main", "main-2"),
		gittest.NewCommitStep("main", "main-3"),
		gittest.NewCommitStep("main", "main-4"),
		gittest.NewCommitStep("main", "main-5"),
		gittest.NewCommitStep("main", "main-6"),

		gittest.NewCallbackStep("main", e{
			"main-6",
			"main-5",
			"main-4",
			"main-3",
			"main-2",
			"main-1",
			"First commit",
		}),
	}

	err = gittest.ExecuteSteps(testRepository, steps, func(expected e) error {
		startCommit, err := testRepository.LatestCommit()
		if err != nil {
			return nil
		}
		w := NewWalker(startCommit)
		checkCommits(t, w, expected)
		return nil
	})
	checkErr(t, err, "execute test steps")
}

func TestWalker_MultiBranch_WithoutMerges(t *testing.T) {
	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = os.RemoveAll(testRepository.Path)
		checkErr(t, err, "removing repository")
	}()

	type e = []string
	steps := []gittest.Step{
		gittest.NewCommitStep("main", "main-1"),
		gittest.NewCallbackStep("", e{
			"main-1",
			"First commit",
		}),

		// create beta branch and add commits
		gittest.NewCommitStep("beta", "beta-1"),
		gittest.NewCommitStep("main", "main-2"),
		gittest.NewCommitStep("beta", "beta-2"),
		gittest.NewCommitStep("beta", "beta-3"),
		gittest.NewCallbackStep("beta", e{
			"beta-3",
			"beta-2",
			"beta-1",
			"main-1",
			"First commit",
		}),

		gittest.NewCommitStep("main", "main-3"),
		gittest.NewCommitStep("beta", "beta-4"),

		// create alpha branch and add commits
		gittest.NewCommitStep("alpha", "alpha-1"),
		gittest.NewCommitStep("alpha", "alpha-2"),
		gittest.NewCommitStep("main", "main-4"),
		gittest.NewCommitStep("beta", "beta-5"),
		gittest.NewCommitStep("alpha", "alpha-3"),

		// check all branches
		gittest.NewCallbackStep("main", e{
			"main-4",
			"main-3",
			"main-2",
			"main-1",
			"First commit",
		}),
		gittest.NewCallbackStep("beta", e{
			"beta-5",
			"beta-4",
			"beta-3",
			"beta-2",
			"beta-1",
			"main-1",
			"First commit",
		}),
		gittest.NewCallbackStep("alpha", e{
			"alpha-3",
			"alpha-2",
			"alpha-1",
			"beta-4",
			"beta-3",
			"beta-2",
			"beta-1",
			"main-1",
			"First commit",
		}),
	}

	err = gittest.ExecuteSteps(testRepository, steps, func(expected e) error {
		startCommit, err := testRepository.LatestCommit()
		if err != nil {
			return nil
		}
		w := NewWalker(startCommit)
		checkCommits(t, w, expected)
		return nil
	})
	checkErr(t, err, "execute test steps")
}

func TestWalker_MultiBranch_WithMerges(t *testing.T) {
	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = os.RemoveAll(testRepository.Path)
		checkErr(t, err, "removing repository")
	}()

	type e = []string
	steps := []gittest.Step{
		gittest.NewCommitStep("main", "main-1"),
		gittest.NewCallbackStep("", e{
			"main-1",
			"First commit",
		}),

		// create beta branch and add commits
		gittest.NewCommitStep("beta", "beta-1"),
		gittest.NewCommitStep("main", "main-2"),
		gittest.NewCommitStep("beta", "beta-2"),
		gittest.NewCommitStep("beta", "beta-3"),
		gittest.NewCallbackStep("main", e{
			"main-2",
			"main-1",
			"First commit",
		}),
		gittest.NewCallbackStep("beta", e{
			"beta-3",
			"beta-2",
			"beta-1",
			"main-1",
			"First commit",
		}),

		gittest.NewCommitStep("main", "main-3"),
		gittest.NewCommitStep("beta", "beta-4"),

		// create alpha branch and add commits
		gittest.NewCommitStep("alpha", "alpha-1"),
		gittest.NewCommitStep("alpha", "alpha-2"),
		gittest.NewCommitStep("main", "main-4"),
		gittest.NewCommitStep("beta", "beta-5"),
		gittest.NewCommitStep("alpha", "alpha-3"),
		gittest.NewCallbackStep("alpha", e{
			"alpha-3",
			"alpha-2",
			"alpha-1",
			"beta-4",
			"beta-3",
			"beta-2",
			"beta-1",
			"main-1",
			"First commit",
		}),

		// merge alpha into beta
		gittest.NewMergeStep("beta", "alpha", false),

		gittest.NewCommitStep("main", "main-5"),
		gittest.NewCommitStep("beta", "beta-6"),

		// merge beta into main
		gittest.NewMergeStep("main", "beta", false),

		gittest.NewCommitStep("main", "main-6"),
		gittest.NewCommitStep("main", "main-7"),

		// check main branch
		gittest.NewCallbackStep("main", e{
			"main-7",
			"main-6",
			"Merge branch 'beta'\n",
			"beta-6",
			"Merge branch 'alpha' into beta\n",
			"alpha-3",
			"alpha-2",
			"alpha-1",
			"beta-5",
			"beta-4",
			"beta-3",
			"beta-2",
			"beta-1",
			"main-5",
			"main-4",
			"main-3",
			"main-2",
			"main-1",
			"First commit",
		}),
	}

	err = gittest.ExecuteSteps(testRepository, steps, func(expected e) error {
		startCommit, err := testRepository.LatestCommit()
		if err != nil {
			return nil
		}
		w := NewWalker(startCommit)
		checkCommits(t, w, expected)
		return nil
	})
	checkErr(t, err, "execute test steps")
}

func TestWalker_MultiBranch_SingleMergeBase_WithoutMerges(t *testing.T) {
	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = os.RemoveAll(testRepository.Path)
		checkErr(t, err, "removing repository")
	}()

	type e = []string
	steps := []gittest.Step{
		gittest.NewCommitStep("main", "main-1"),

		// create beta and alpha branch and add commits
		gittest.NewCheckoutStep("main", "beta", true),
		gittest.NewCheckoutStep("main", "alpha", true),
		gittest.NewCommitStep("beta", "beta-1"),
		gittest.NewCommitStep("main", "main-2"),
		gittest.NewCommitStep("alpha", "alpha-1"),
		gittest.NewCommitStep("beta", "beta-2"),
		gittest.NewCommitStep("alpha", "alpha-2"),

		// check all branches
		gittest.NewCallbackStep("main", e{
			"main-2",
			"main-1",
			"First commit",
		}),
		gittest.NewCallbackStep("beta", e{
			"beta-2",
			"beta-1",
			"main-1",
			"First commit",
		}),
		gittest.NewCallbackStep("alpha", e{
			"alpha-2",
			"alpha-1",
			"main-1",
			"First commit",
		}),
	}

	err = gittest.ExecuteSteps(testRepository, steps, func(expected e) error {
		startCommit, err := testRepository.LatestCommit()
		if err != nil {
			return nil
		}
		w := NewWalker(startCommit)
		checkCommits(t, w, expected)
		return nil
	})
	checkErr(t, err, "execute test steps")
}

func TestWalker_MultiBranch_SingleMergeBase_WithMerges(t *testing.T) {
	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = os.RemoveAll(testRepository.Path)
		checkErr(t, err, "removing repository")
	}()

	type e = []string
	steps := []gittest.Step{
		gittest.NewCommitStep("main", "main-1"),

		// create beta and alpha branch and add commits
		gittest.NewCheckoutStep("main", "beta", true),
		gittest.NewCheckoutStep("main", "alpha", true),
		gittest.NewCommitStep("beta", "beta-1"),
		gittest.NewCommitStep("main", "main-2"),
		gittest.NewCommitStep("alpha", "alpha-1"),
		gittest.NewCommitStep("alpha", "alpha-2"),

		// merge alpha into beta
		gittest.NewMergeStep("beta", "alpha", false),
		gittest.NewCommitStep("beta", "beta-2"),

		// merge beta into alpha
		gittest.NewMergeStep("main", "beta", false),

		// check main branch
		gittest.NewCallbackStep("main", e{
			"Merge branch 'beta'\n",
			"beta-2",
			"Merge branch 'alpha' into beta\n",
			"alpha-2",
			"alpha-1",
			"beta-1",
			"main-2",
			"main-1",
			"First commit",
		}),
	}

	err = gittest.ExecuteSteps(testRepository, steps, func(expected e) error {
		startCommit, err := testRepository.LatestCommit()
		if err != nil {
			return nil
		}
		w := NewWalker(startCommit)
		checkCommits(t, w, expected)
		return nil
	})
	checkErr(t, err, "execute test steps")
}

func TestWalker_MultiBranch_OctopusMerge(t *testing.T) {
	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = os.RemoveAll(testRepository.Path)
		checkErr(t, err, "removing repository")
	}()

	type e = []string
	steps := []gittest.Step{
		gittest.NewCommitStep("main", "main-1"),

		// create beta branch and add commits
		gittest.NewCommitWithFileStep("beta", "beta-1", "./beta-1.txt"),
		gittest.NewCommitWithFileStep("main", "main-2", "./main-2.txt"),
		gittest.NewCommitWithFileStep("beta", "beta-2", "./beta-2.txt"),

		// create alpha branch and add commits
		gittest.NewCommitWithFileStep("alpha", "alpha-1", "./alpha-1.txt"),
		gittest.NewCommitWithFileStep("alpha", "alpha-2", "./alpha-2.txt"),
		gittest.NewCommitWithFileStep("main", "main-3", "./main-3.txt"),
		gittest.NewCommitWithFileStep("beta", "beta-3", "./beta-3.txt"),
		gittest.NewCommitWithFileStep("alpha", "alpha-3", "./alpha-3.txt"),

		// merge alpha and beta into beta
		gittest.NewOctopusMergeStep("main", []string{"alpha", "beta"}),

		gittest.NewCommitStep("main", "main-5"),
		gittest.NewCommitStep("beta", "beta-6"),

		// check main branch
		gittest.NewCallbackStep("main", e{
			"main-5",
			"Merge branches 'beta' and 'alpha'\n",
			"alpha-3",
			"alpha-2",
			"alpha-1",
			"beta-3",
			"beta-2",
			"beta-1",
			"main-3",
			"main-2",
			"main-1",
			"First commit",
		}),
	}

	err = gittest.ExecuteSteps(testRepository, steps, func(expected e) error {
		startCommit, err := testRepository.LatestCommit()
		if err != nil {
			return nil
		}
		w := NewWalker(startCommit)
		checkCommits(t, w, expected)
		return nil
	})
	checkErr(t, err, "execute test steps")
}

func checkCommits(t *testing.T, walker object.CommitIter, expected []string) {
	assert := assertion.New(t)

	actual := []string{}
	err := walker.ForEach(func(c *object.Commit) error {
		actual = append(actual, strings.Split(c.Message, ":")[0])
		return nil
	})
	checkErr(t, err, "traverse commits")
	assert.Equal(expected, actual)
}

func checkErr(t *testing.T, err error, message string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %s", message, err)
	}
}

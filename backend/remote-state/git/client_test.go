package git

import (
	"testing"
)

func getRemoteClient(t *testing.T) *RemoteClient {
	b := getBackend(t)

	_, rc, err := b.StateMgr()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return rc
}

func TestRemoteClientGetPath(t *testing.T) {
	rc := getRemoteClient(t)

	data, err := rc.getPath("README.md")
	if err != nil {
		t.Fatalf("error getting README.md: %s", err)
	}

	if len(data) == 0 {
		t.Fatal("README.md returned zero bytes")
	}

	_, err = rc.getPath("no-git-has-a-file-with-this-name.txt")
	if err == nil {
		t.Fatal("expected file not found error")
	}
}

func TestRemoteClientCommit(t *testing.T) {
	rc := getRemoteClient(t)

	err := rc.commit("Hello Git World this is terraform!")
	if err != nil {
		t.Fatal("failed to commit: ", err)
	}
}

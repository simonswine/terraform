package git

import (
	"testing"
	"time"

	gitplumbing "srcd.works/go-git.v4/plumbing"
	gitobject "srcd.works/go-git.v4/plumbing/object"
)

func TestCommitEncode(t *testing.T) {

	signature := gitobject.Signature{
		Name:  "Mr Commiter",
		Email: "email@commiter.org",
		When:  time.Now(),
	}

	c := Commit{
		Parents: []gitplumbing.Hash{gitplumbing.NewHash("abcd3b19e793491b1c6ea0fd8b46cd9f32e592fc")},
	}
	c.Message = "Test commit"
	c.Author = signature
	c.Committer = signature

	mem := gitplumbing.EncodedObject(&gitplumbing.MemoryObject{})
	err := c.Encode(mem)
	if err != nil {
		t.Fatal("error encoding commit: ", err)
	}

	if mem.Size() < 10 {
		t.Errorf("object is too small")
	}

	if mem.Hash().String() == "0000000000000000000000000000000000000000" {
		t.Errorf("hash is zero")
	}

}

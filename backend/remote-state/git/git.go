package git

import (
	"bytes"
	"fmt"

	"srcd.works/go-git.v4/plumbing"
	"srcd.works/go-git.v4/plumbing/object"
	"srcd.works/go-git.v4/utils/ioutil"
)

type Commit struct {
	object.Commit
	Tree    plumbing.Hash
	Parents []plumbing.Hash
}

// Encode transforms a Commit into a plumbing.EncodedObject.
func (b *Commit) Encode(o plumbing.EncodedObject) error {
	var err error
	w := bytes.NewBuffer([]byte{})
	if _, err = fmt.Fprintf(w, "tree %s\n", b.Tree.String()); err != nil {
		return err
	}
	for _, parent := range b.Parents {
		if _, err = fmt.Fprintf(w, "parent %s\n", parent.String()); err != nil {
			return err
		}
	}
	if _, err = fmt.Fprint(w, "author "); err != nil {
		return err
	}
	if err = b.Author.Encode(w); err != nil {
		return err
	}
	if _, err = fmt.Fprint(w, "\ncommitter "); err != nil {
		return err
	}
	if err = b.Committer.Encode(w); err != nil {
		return err
	}
	if _, err = fmt.Fprintf(w, "\n\n%s", b.Message); err != nil {
		return err
	}

	o.SetType(plumbing.CommitObject)
	w2, err := o.Writer()
	if err != nil {
		return err
	}
	defer ioutil.CheckClose(w2, &err)
	size, err := w.WriteTo(w2)
	o.SetSize(size)
	return err
}

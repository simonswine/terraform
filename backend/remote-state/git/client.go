package git

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"

	"srcd.works/go-git.v4"
	gitconfig "srcd.works/go-git.v4/config"
	gitplumbing "srcd.works/go-git.v4/plumbing"
	gitobject "srcd.works/go-git.v4/plumbing/object"
	"srcd.works/go-git.v4/storage/memory"
)

const (
	lockSuffix     = "/.lock"
	lockInfoSuffix = "/.lockinfo"
)

// RemoteClient is a remote client that stores data in Git.
type RemoteClient struct {
	Options *git.CloneOptions
	Path    string
	Branch  string

	repositoryCache *git.Repository
	storage         *memory.Storage

	lockCh <-chan struct{}
}

func (c *RemoteClient) Repository() (*git.Repository, error) {
	if c.repositoryCache == nil {
		c.storage = memory.NewStorage()
		repo, err := git.Clone(c.storage, nil, c.Options)
		if err != nil {
			return nil, err
		}
		c.repositoryCache = repo
	}
	return c.repositoryCache, nil
}

func (c *RemoteClient) commit(message string) error {
	// get branch head
	repo, err := c.Repository()
	if err != nil {
		return err
	}

	branch, err := repo.Head()
	if err != nil {
		return err
	}

	head, err := repo.Commit(branch.Hash())
	if err != nil {
		return err
	}

	tree, err := head.Tree()
	if err != nil {
		return err
	}

	signature := gitobject.Signature{
		Name:  "Mr Commiter",
		Email: "email@commiter.org",
		When:  time.Now(),
	}

	commit := Commit{
		Parents: []gitplumbing.Hash{head.ID()},
		Tree:    tree.ID(),
	}
	commit.Message = message
	commit.Author = signature
	commit.Committer = signature

	obj := c.storage.NewEncodedObject()
	err = commit.Encode(obj)
	if err != nil {
		return err
	}

	_, err = c.storage.SetEncodedObject(obj)
	if err != nil {
		return err
	}

	ref := gitplumbing.NewHashReference(
		c.Options.ReferenceName,
		obj.Hash(),
	)

	//err = c.storage.SetReference(ref)
	//if err != nil {
	//	return err
	//}

	refspec := gitconfig.RefSpec(
		fmt.Sprintf(
			"refs/heads/%s:refs/remotes/%s/%s",
			c.Branch,
			DefaultRemoteName,
			c.Branch,
		),
	)
	return fmt.Errorf("%+v\n%+v", refspec, ref)

	pushOptions := &git.PushOptions{
		Auth:       c.Options.Auth,
		RemoteName: DefaultRemoteName,
		RefSpecs:   []gitconfig.RefSpec{refspec},
	}

	return repo.Push(pushOptions)
}

func (c *RemoteClient) getPath(path string) ([]byte, error) {
	repo, err := c.Repository()
	if err != nil {
		return nil, err
	}

	branch, err := repo.Head()
	if err != nil {
		return nil, err
	}

	commit, err := repo.Commit(branch.Hash())
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	file, err := tree.File(path)
	if err != nil {
		return nil, err
	}

	reader, err := file.Blob.Reader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	data, err := c.getPath(c.Path)
	if err != nil {
		return nil, err
	}

	md5 := md5.Sum(data)

	return &remote.Payload{
		Data: data,
		MD5:  md5[:],
	}, nil
}

func (c *RemoteClient) Put(data []byte) error {
	return fmt.Errorf("implement me: %s", "PUT")
}

func (c *RemoteClient) Delete() error {
	return fmt.Errorf("implement me: %s", "DELETE")
}

func (c *RemoteClient) putLockInfo(info *state.LockInfo) error {
	info.Path = c.Path
	info.Created = time.Now().UTC()

	// Key:   c.Path + lockInfoSuffix,
	// Value: info.Marshal(),

	return fmt.Errorf("implement me: %s", "PUTLOCKINFO")
}

func (c *RemoteClient) getLockInfo() (*state.LockInfo, error) {
	path := c.Path + lockInfoSuffix
	data, err := c.getPath(path)
	if err != nil {
		return nil, err
	}

	li := &state.LockInfo{}
	err = json.Unmarshal(data, li)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling lock info: %s", err)
	}

	return li, nil
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	select {
	case <-c.lockCh:
		// We had a lock, but lost it.
		// Since we typically only call lock once, we shouldn't ever see this.
		return "", errors.New("lost git lock")
	default:
		if c.lockCh != nil {
			// we have an active lock already
			return "", nil
		}
	}

	return "", fmt.Errorf("implement me: %s", "LOCK")
}

func (c *RemoteClient) Unlock(id string) error {
	return fmt.Errorf("implement me: %s", "UNLOCK")
}

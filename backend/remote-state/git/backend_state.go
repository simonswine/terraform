package git

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

const (
	keyEnvPrefix = "-env:"
)

func (b *Backend) States() ([]string, error) {
	return []string{}, fmt.Errorf("Unimplemented: States")
}

func (b *Backend) DeleteState(name string) error {
	return fmt.Errorf("Unimplemented: Delete State")
}

func (b *Backend) StateMgr() (*remote.State, *RemoteClient, error) {
	// Get the Consul API client
	opts, err := b.clientRaw()
	if err != nil {
		return nil, nil, err
	}

	// Determine the brach of the repo
	branch := b.configData.Get("branch").(string)

	// Build the state client
	client := &RemoteClient{
		Options: opts,
		Branch:  branch,
	}
	return &remote.State{
		Client: client,
	}, client, nil
}

func (b *Backend) State(name string) (state.State, error) {
	stateMgr, client, err := b.StateMgr()
	if err != nil {
		return nil, err
	}
	client.Path = name

	// Grab a lock, we use this to write an empty state if one doesn't
	// exist already. We have to write an empty state as a sentinel value
	// so States() knows it exists.
	lockInfo := state.NewLockInfo()
	lockInfo.Operation = "init"
	lockId, err := stateMgr.Lock(lockInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to lock state in Git: %s", err)
	}

	// Local helper function so we can call it multiple places
	lockUnlock := func(parent error) error {
		if err := stateMgr.Unlock(lockId); err != nil {
			return fmt.Errorf(strings.TrimSpace(errStateUnlock), lockId, err)
		}

		return parent
	}

	// Grab the value
	if err := stateMgr.RefreshState(); err != nil {
		err = lockUnlock(err)
		return nil, err
	}

	// If we have no state, we have to create an empty state
	if v := stateMgr.State(); v == nil {
		if err := stateMgr.WriteState(terraform.NewState()); err != nil {
			err = lockUnlock(err)
			return nil, err
		}
		if err := stateMgr.PersistState(); err != nil {
			err = lockUnlock(err)
			return nil, err
		}
	}

	// Unlock, the state should now be initialized
	if err := lockUnlock(nil); err != nil {
		return nil, err
	}

	return stateMgr, nil
}

func (b *Backend) path(name string) string {
	path := b.configData.Get("path").(string)
	if name != backend.DefaultStateName {
		path += fmt.Sprintf("%s%s", keyEnvPrefix, name)
	}

	return path
}

const errStateUnlock = `
Error unlocking Git state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
`

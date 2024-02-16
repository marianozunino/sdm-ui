package program

import (
	"bytes"
	"fmt"
)

func (p *Program) executeSync() error {
	fmt.Printf("[sync] Account: %s\n", p.account)
	fmt.Println("[sync] Syncing...")
	statusesBuffer := new(bytes.Buffer)

	if err := p.retryCommand(func() error {
		statusesBuffer.Reset()
		return p.sdmWrapper.Status(statusesBuffer)
	}); err != nil {
		fmt.Println("[sync] Failed to sync with SDM")
		return err
	}

	dataSources := parseDataSources(statusesBuffer.String())
	p.db.StoreServers(p.account, dataSources)

	servers := parseServers(statusesBuffer.String())
	p.db.StoreServers(p.account, servers)

	return nil
}

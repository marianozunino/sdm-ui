package program

import (
	"fmt"
	"io"
)

func (p *Program) executeList(w io.Writer) error {
	dataSources, err := p.db.RetrieveDatasources(p.account)
	if err != nil {
		return err
	}

	if len(dataSources) == 0 {
		fmt.Printf("[list] No data sources found, syncing...\n")
		if err := p.executeSync(); err != nil {
			return err
		}

		dataSources, err = p.db.RetrieveDatasources(p.account)
		if err != nil {
			return err
		}
	}

	printDataSources(dataSources, w)
	return nil
}

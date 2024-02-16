package program

import "io"

func (p *Program) executeList(w io.Writer) error {
	dataSources, err := p.db.RetrieveDatasources(p.account)
	if err != nil {
		return err
	}

	printDataSources(dataSources, w)
	return nil
}

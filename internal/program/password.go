package program

import (
	"fmt"

	"github.com/ncruces/zenity"
)

func (p *Program) retrievePassword() (string, error) {
	password, err := p.keyring.GetSecret(p.account)
	if err != nil {
		password, err = p.askForPassword()
		if err != nil {
			return "", err
		}
		if password == "" {
			return "", fmt.Errorf("Please provide a password")
		}
		if err := p.keyring.SetSecret(p.account, password); err != nil {
			return "", err
		}
	}
	return password, nil
}

func (p *Program) askForPassword() (string, error) {
	fmt.Println("[login] Prompting for password...")
	_, pwd, err := zenity.Password(zenity.Title("Type your SDM password"))
	return pwd, err
}

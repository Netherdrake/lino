package model

import (
	types "github.com/lino-network/lino/types"
)

type Developer struct {
	Username       types.AccountKey `json:"username"`
	Deposit        types.Coin       `json:"deposit"`
	AppConsumption types.Coin       `json:"app_consumption"`
}

type DeveloperList struct {
	AllDevelopers []types.AccountKey `json:"all_developers"`
}
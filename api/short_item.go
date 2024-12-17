package api

type ShortItem struct {
	ItemName string `json:"item_name" alias:"niinlookup.niin"`
	Niin     string `json:"niin" alias:"niinlookup.niin"`
	Fsc      string `json:"fsc" alias:"niinlookup.fsc"`
	HasAmdf  bool   `json:"has_amdf" alias:"niinlookup.has_amdf"`
	HasFlis  bool   `json:"has_flis" alias:"niinlookup.has_flis"`
}

package model

type ShortItem struct {
	ItemName    string `json:"item_name"`
	Niin        string `json:"niin"`
	Fsc         string `json:"fsc"`
	HasAmdfData bool   `json:"has_amdf_data"`
	HasFlisData bool   `json:"has_flis_data"`
}

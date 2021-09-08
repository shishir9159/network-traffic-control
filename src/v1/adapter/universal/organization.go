package universal


type Organization struct {
	BaseEntity
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Company      Company      `json:"company"`
	GitDirectory GitDirectory `json:"gitDirectory"`
}

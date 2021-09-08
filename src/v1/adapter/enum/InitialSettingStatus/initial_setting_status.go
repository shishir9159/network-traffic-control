package InitialSettingStatus

type InitialSettingStatus string

const (
	PENDING   = InitialSettingStatus("PENDING")
	SKIPPED   = InitialSettingStatus("SKIPPED")
	COMPLETED = InitialSettingStatus("COMPLETED")
)

var IsInitialSettingStatus = map[string]bool{
	"PENDING":   true,
	"SKIPPED":   true,
	"COMPLETED": true,
}

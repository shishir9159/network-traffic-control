package UserType

type UserType string

const (
	REGULAR       = UserType("REGULAR")
	AGENT_SERVICE = UserType("AGENT_SERVICE")
	EXTERNAL      = UserType("EXTERNAL")
	GIT           = UserType("GIT")
)

var IsUserType = map[string]bool{
	"REGULAR":       true,
	"AGENT_SERVICE": true,
	"EXTERNAL":      true,
	"GIT":           true,
}

var GetUserTypeAsString = map[UserType]string{
	REGULAR:       "REGULAR",
	AGENT_SERVICE: "AGENT_SERVICE",
	EXTERNAL:      "EXTERNAL",
	GIT:           "GIT",
}

func GetUserType(userType string) UserType {
	return UserType(userType)
}

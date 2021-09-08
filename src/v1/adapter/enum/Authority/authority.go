package Authority

type Authority string

const (
	ROLE_DEV           = Authority("ROLE_DEV")
	ROLE_OPS           = Authority("ROLE_OPS")
	ROLE_ADMIN         = Authority("ROLE_ADMIN")
	ROLE_SUPER_ADMIN   = Authority("ROLE_SUPER_ADMIN")
	ROLE_AGENT_SERVICE = Authority("ROLE_AGENT_SERVICE")
	ROLE_GIT_AGENT     = Authority("ROLE_GIT_AGENT")
	ROLE_SNAPSHOTTER   = Authority("ROLE_SNAPSHOTTER")
	ANONYMOUS          = Authority("ANONYMOUS")
	ROLE_SYSTEM          = Authority("ROLE_SYSTEM")
)

var IsAuthority = map[string]bool{
	"ROLE_DEV":           true,
	"ROLE_OPS":           true,
	"ROLE_ADMIN":         true,
	"ROLE_SUPER_ADMIN":   true,
	"ROLE_AGENT_SERVICE": true,
	"ROLE_GIT_AGENT":     true,
	"ROLE_SNAPSHOTTER":   true,
	"ANONYMOUS":          true,
	"ROLE_SYSTEM": true,
}

var GetAuthorityString = map[Authority]string{
	ROLE_DEV:           "ROLE_DEV",
	ROLE_OPS:           "ROLE_OPS",
	ROLE_ADMIN:         "ROLE_ADMIN",
	ROLE_SUPER_ADMIN:   "ROLE_SUPER_ADMIN",
	ROLE_AGENT_SERVICE: "ROLE_AGENT_SERVICE",
	ROLE_GIT_AGENT:     "ROLE_GIT_AGENT",
	ROLE_SNAPSHOTTER:   "ROLE_SNAPSHOTTER",
	ANONYMOUS:          "ANONYMOUS",
	ROLE_SYSTEM: "ROLE_SYSTEM",
}

var GetAuthorityEnum = map[string]Authority{
	"ROLE_DEV":           ROLE_DEV,
	"ROLE_OPS":           ROLE_OPS,
	"ROLE_ADMIN":         ROLE_ADMIN,
	"ROLE_SUPER_ADMIN":   ROLE_SUPER_ADMIN,
	"ROLE_AGENT_SERVICE": ROLE_AGENT_SERVICE,
	"ROLE_GIT_AGENT":     ROLE_GIT_AGENT,
	"ROLE_SNAPSHOTTER":   ROLE_SNAPSHOTTER,
	"ANONYMOUS":          ANONYMOUS,
	"ROLE_SYSTEM": ROLE_SYSTEM,
}

func GetAuthority(authority string) Authority {
	return GetAuthorityEnum[authority]
}

func ContainsAnyAuthority(authorities []Authority, authority ...Authority) bool {
	for _, auth := range authority {
		if ContainsAuthority(authorities, auth) {
			return true
		}
	}
	return false
}

func ContainsAuthority(authorities []Authority, authority Authority) bool {
	for _, auth := range authorities {
		if auth == authority {
			return true
		}
	}
	return false
}

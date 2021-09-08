package AccountType

type AccountType string

const (
	COMPANY    = AccountType("COMPANY")
	INDIVIDUAL = AccountType("INDIVIDUAL")
)

var IsAccountType = map[string]bool{
	"COMPANY":    true,
	"INDIVIDUAL": true,
}

var GetUserTypeAsEnum = map[string]AccountType{
	"COMPANY":    COMPANY,
	"INDIVIDUAL": INDIVIDUAL,
}

var GetUserTypeAsString = map[AccountType]string{
	COMPANY:    "COMPANY",
	INDIVIDUAL: "INDIVIDUAL",
}

func GetAccountType(accountType string) AccountType {
	return GetUserTypeAsEnum[accountType]
}

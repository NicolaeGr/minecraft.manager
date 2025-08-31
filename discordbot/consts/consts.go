package consts

var AdminIDs = []string{
	"695996468762378252",
}

func IsAdmin(userID string) bool {
	for _, id := range AdminIDs {
		if userID == id {
			return true
		}
	}
	return false
}

type ContextKey string

const ManagerKey ContextKey = "mgr"

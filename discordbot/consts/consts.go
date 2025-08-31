package consts

var AdminIDs = []string{
	"123456789012345678", // Replace with actual admin Discord user IDs
}

func IsAdmin(userID string) bool {
	for _, id := range AdminIDs {
		if userID == id {
			return true
		}
	}
	return false
}

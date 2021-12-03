package model

type Members struct {
	ServiceAccounts []ServiceAccountUUID
	Users           []UserUUID
	Groups          []GroupUUID
}

type MemberNotation struct {
	Type string `json:"type"`
	UUID string `json:"uuid"`
}

// FixMembers remove from members invalid links, if some removed, returns true
func FixMembers(members *[]MemberNotation, users []UserUUID, groups []GroupUUID,
	serviceAccounts []ServiceAccountUUID) bool {
	if len(*members) == len(users)+len(groups)+len(serviceAccounts) {
		return false
	}
	buildSet := func(uuids []string) map[string]struct{} {
		result := map[string]struct{}{}
		for _, uuid := range uuids {
			result[uuid] = struct{}{}
		}
		return result
	}
	membersSuperSet := map[string]map[string]struct{}{
		UserType:           buildSet(users),
		ServiceAccountType: buildSet(serviceAccounts),
		GroupType:          buildSet(groups),
	}
	newMembers := make([]MemberNotation, 0, len(*members))

	fixed := false
	for _, m := range *members {
		if _, ok := membersSuperSet[m.Type][m.UUID]; ok {
			newMembers = append(newMembers, m)
		} else {
			fixed = true
		}
	}
	*members = newMembers
	return fixed
}

package internal

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/rolebinding-watcher/pkg"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authz"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type UserEffectiveRoleProcessor interface {
	ProceedUserEffectiveRole(userEffectiveRoles pkg.UserEffectiveRoles) error
}

type ChangesProcessor struct {
	userEffectiveRoleProcessor UserEffectiveRoleProcessor
	Logger                     hclog.Logger
}

func (c *ChangesProcessor) UpdateUserEffectiveRoles(futureDB *io.MemoryStoreTxn, users map[pkg.UserUUID]struct{}, roles map[pkg.RoleName]struct{}) error {
	//c.Logger.Info(fmt.Sprintf("users: %v", users)) // TODO REMOVE
	//c.Logger.Info(fmt.Sprintf("roles: %v", roles)) // TODO REMOVE
	service := pkg.UserEffectiveRolesService(futureDB)
	for userUUID := range users {
		newUsersEffectiveRoleResults, err := c.CalculateNewUserEffectiveRoles(futureDB, userUUID, makeSlice(roles))
		c.Logger.Info(fmt.Sprintf("newUsersEffectiveRoleResults: %v", newUsersEffectiveRoleResults))
		if err != nil {
			return err
		}
		for _, newUsersEffectiveRoleResult := range newUsersEffectiveRoleResults {
			key, newUsersEffectiveRoles := buildEffectiveRoles(userUUID, newUsersEffectiveRoleResult)
			oldUsersEffectiveRoles, err := service.GetByKey(key)
			if err != nil {
				return fmt.Errorf("service.GetByKey: %w", err)
			}
			if newUsersEffectiveRoles.Equal(oldUsersEffectiveRoles) {
				continue
			}
			err = c.userEffectiveRoleProcessor.ProceedUserEffectiveRole(newUsersEffectiveRoles)
			if err != nil {
				return fmt.Errorf("userEffectiveRoleProcessor.ProceedUserEffectiveRole: %w", err)
			}
			err = service.Update(&newUsersEffectiveRoles)
			if err != nil {
				return fmt.Errorf("service.Update: %w", err)
			}
		}
	}
	return nil
}

func buildEffectiveRoles(userID pkg.UserUUID, userEffectiveRoleResult authz.EffectiveRoleResult) (pkg.UserEffectiveRolesKey, pkg.UserEffectiveRoles) {
	return pkg.UserEffectiveRolesKey{
			UserUUID: userID,
			RoleName: userEffectiveRoleResult.Role,
		},
		pkg.UserEffectiveRoles{
			UserUUID: userID,
			RoleName: userEffectiveRoleResult.Role,
			Tenants:  userEffectiveRoleResult.Tenants,
		}
}

func (c *ChangesProcessor) CalculateNewUserEffectiveRoles(futureDB *io.MemoryStoreTxn, userUUID pkg.UserUUID, roles []pkg.RoleName) ([]authz.EffectiveRoleResult, error) {
	roleChecker := authz.NewEffectiveRoleChecker(futureDB)
	return roleChecker.CheckEffectiveRoles(model.Subject{
		Type:       iam_model.UserType,
		UUID:       userUUID,
		TenantUUID: "",
	}, roles)
}

func makeSlice(set map[string]struct{}) []string {
	slice := make([]string, 0, len(set))
	for k := range set {
		slice = append(slice, k)
	}
	sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
	return slice
}

type PrintProceeder struct {
	Logger hclog.Logger
}

func (c PrintProceeder) ProceedUserEffectiveRole(newUsersEffectiveRoles pkg.UserEffectiveRoles) error {
	data, err := json.MarshalIndent(newUsersEffectiveRoles, "", "\t")
	if err != nil {
		return err
	}
	c.Logger.Info("\n" + string(data) + "\n")
	return nil
}

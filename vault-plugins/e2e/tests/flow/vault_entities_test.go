package flow

import (
	"github.com/flant/negentropy/vault-plugins/shared/io"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func assertEntity(fullID string){
	entityID, err := identityApi.EntityApi().GetID(fullID)
	Expect(err).ToNot(HaveOccurred())
	Expect(entityID).ToNot(BeEmpty())
}

func assertEntityAliases(o io.MemoryStorableObject){
	for _, s := range sources {
		eaName := s.ExpectedEaName(o)
		if eaName != "" {
			aliasId, err := identityApi.AliasApi().FindAliasIDByName(eaName, mountAccessorId)
			Expect(err).ToNot(HaveOccurred())
			Expect(aliasId).ToNot(BeEmpty())
		}
	}
}

var _ = Describe("Entity and entity aliases", func() {
	Context("creating user", func() {
		It("creates one entity and entity aliases for sources", func() {
			user := createUser()

			assertEntity(user.FullIdentifier)
			assertEntityAliases(user)
		})
	})

	Context("creating service account", func() {
		FIt("creates one entity and entity aliases for sources", func() {
			sa := createServiceAccount()

			assertEntity(sa.FullIdentifier)
			assertEntityAliases(sa)
		})
	})
})

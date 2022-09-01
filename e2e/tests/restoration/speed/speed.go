package main

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/restoration/common"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
)

var feedingMultipliers = []int{10, 20}

// var feedingMultipliers = []int{1_000, 10_000, 50_000, 100_000}

func main() {
	println(111)
	// to use e2e test libs
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	result := []common.RestorationDurationResult{}
	s := common.Suite{}
	s.BeforeSuite()
	println(222)
	feedingAmmount := 0
	for _, multiplier := range feedingMultipliers {
		toFeed := multiplier - feedingAmmount
		feed(toFeed)
		s.RestartVaults()
		result = append(result, s.CollectMetrics(multiplier))
		feedingAmmount += toFeed
	}
	fmt.Printf("         N     RootIAMFactory    RootAUTHFactory    AuthAUTHFactory\n")
	for _, r := range result {
		fmt.Printf("%10d %18s %18s %18s\n", r.FeedMultiplier, r.RootIAMFactory.String(), r.RootAUTHFactory.String(), r.AuthAUTHFactory.String())
	}
}

func feed(n int) {
	rootClient := lib.NewConfiguredIamVaultClient()
	tenantApi := lib.NewTenantAPI(rootClient)
	userApi := lib.NewUserAPI(rootClient)
	tenant := specs.CreateRandomTenant(tenantApi)
	for i := 0; i < n; i++ {
		if i%10 == 0 {
			fmt.Printf("%d/%d\n", n, i)
		}
		specs.CreateRandomUser(userApi, tenant.UUID)
	}
}

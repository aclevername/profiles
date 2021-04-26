package catalog_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"github.com/weaveworks/profiles/pkg/catalog"
)

var _ = Describe("Catalog", func() {
	var (
		c       *catalog.Catalog
		catName string
	)

	BeforeEach(func() {
		c = catalog.New()
		catName = "whiskers"
	})

	It("manages an in memory list of profiles", func() {
		By("adding profiles to the list")
		profiles := []profilesv1.ProfileDescription{{Name: "foo"}, {Name: "bar"}, {Name: "alsofoo"}}
		c.Add(catName, profiles...)

		By("returning all matching profiles based on query string")
		Expect(c.Search("foo")).To(ConsistOf(
			profilesv1.ProfileDescription{Name: "foo", Catalog: catName},
			profilesv1.ProfileDescription{Name: "alsofoo", Catalog: catName},
		))

		By("getting details for a specific named profile in a catalog")
		exists, profile := c.Get(catName, "foo")
		Expect(exists).To(BeTrue())
		Expect(profile).To(Equal(
			profilesv1.ProfileDescription{Name: "foo", Catalog: catName},
		))

		By("returns empty list when no matching profile exist")
		Expect(c.Search("dontexist")).To(HaveLen(0))

		By("returns false when the profile doesn't exist")
		exists, _ = c.Get(catName, "dontexist")
		Expect(exists).To(BeFalse())
	})
})

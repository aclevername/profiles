package version_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/profiles/pkg/version"
)

var _ = Describe("Version", func() {
	Context("GetVersion", func() {
		It("should return the version", func() {
			Expect(version.GetVersion()).To(Equal("v0.0.1"))
		})
	})
})

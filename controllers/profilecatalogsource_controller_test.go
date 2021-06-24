package controllers_test

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ProfileCatalogSourceController", func() {
	var (
		namespace string
		ctx       = context.Background()
	)

	BeforeEach(func() {
		namespace = uuid.New().String()
		nsp := v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(context.Background(), &nsp)).To(Succeed())
	})

	Context("Create, update and delete", func() {
		It("syncs the in-memory list when a ProfileCatalogSource is added or deleted", func() {
			By("creating a new ProfileCatalogSource")
			catalogSource := &profilesv1.ProfileCatalogSource{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ProfileCatalogSource",
					APIVersion: "profile.weave.works/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "catalog",
					Namespace: namespace,
				},
				Spec: profilesv1.ProfileCatalogSourceSpec{
					Profiles: []profilesv1.ProfileCatalogEntry{
						{
							ProfileDescription: profilesv1.ProfileDescription{
								Name:        "foo",
								Description: "bar",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, catalogSource)).Should(Succeed())

			By("searching for a profile")
			query := func() []profilesv1.ProfileCatalogEntry {
				return catalogReconciler.Profiles.Search("foo")
			}
			Eventually(query, 2*time.Second).Should(ContainElement(profilesv1.ProfileCatalogEntry{ProfileDescription: profilesv1.ProfileDescription{Name: "foo", Description: "bar"}, CatalogSource: "catalog"}))
			Expect(k8sClient.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: "catalog"}, catalogSource)).To(Succeed())

			By("adding more items to ProfileCatalogSource")
			pName := fmt.Sprintf("new-profile-%s", uuid.New().String())
			catalogSource.Spec.Profiles = append(catalogSource.Spec.Profiles, profilesv1.ProfileCatalogEntry{
				ProfileDescription: profilesv1.ProfileDescription{
					Name:        pName,
					Description: "I am new here",
				},
			})
			Expect(k8sClient.Update(context.Background(), catalogSource)).To(Succeed())

			Eventually(func() []profilesv1.ProfileCatalogEntry {
				return catalogReconciler.Profiles.Search(pName)
			}, 2*time.Second).Should(ConsistOf(profilesv1.ProfileCatalogEntry{
				ProfileDescription: profilesv1.ProfileDescription{
					Name:        pName,
					Description: "I am new here",
				},
				CatalogSource: "catalog",
			}))

			By("deleting the ProfileCatalogSource")
			Expect(k8sClient.Delete(ctx, catalogSource)).To(Succeed())
			Eventually(query, 2*time.Second).Should(BeEmpty())
			Expect(catalogReconciler.Profiles.Search(pName)).To(BeEmpty())
		})
	})
})

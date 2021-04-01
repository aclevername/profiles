package controllers_test

import (
	"context"
	"fmt"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/profiles/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ProfileController", func() {
	const nginxProfileURL = "https://github.com/weaveworks/nginx-profile"

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

	Context("Create", func() {
		DescribeTable("Applying a Profile creates the correct resources", func(pSubSpec v1alpha1.ProfileSubscriptionSpec) {
			subscriptionName := "foo"
			branch := "main"

			pSub := v1alpha1.ProfileSubscription{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ProfileSubscription",
					APIVersion: "profilesubscriptions.weave.works/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      subscriptionName,
					Namespace: namespace,
				},
			}
			pSub.Spec = pSubSpec
			Expect(k8sClient.Create(ctx, &pSub)).Should(Succeed())

			By("creating a GitRepository resource")
			profileRepoName := "nginx-profile"
			gitRepoName := fmt.Sprintf("%s-%s-%s", subscriptionName, profileRepoName, branch)
			gitRepo := sourcev1.GitRepository{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: gitRepoName, Namespace: namespace}, &gitRepo)
			}, 10*time.Second).ShouldNot(HaveOccurred())
			Expect(gitRepo.Spec.URL).To(Equal(nginxProfileURL))
			Expect(gitRepo.Spec.Reference.Branch).To(Equal(branch))

			By("creating a HelmRelease resource")
			profileName := "nginx"
			chartName := "nginx-server"
			helmReleaseName := fmt.Sprintf("%s-%s-%s", subscriptionName, profileName, chartName)
			helmRelease := helmv2.HelmRelease{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: helmReleaseName, Namespace: namespace}, &helmRelease)
			}, 10*time.Second).ShouldNot(HaveOccurred())
			Expect(helmRelease.Spec.Chart.Spec.Chart).To(Equal("nginx/chart"))
			Expect(helmRelease.Spec.Chart.Spec.SourceRef).To(Equal(
				helmv2.CrossNamespaceObjectReference{
					Kind:      "GitRepository",
					Name:      gitRepoName,
					Namespace: namespace,
				},
			))
			if pSub.Spec.Values != nil {
				Expect(helmRelease.Spec.Values).To(Equal(pSub.Spec.Values))
			}
			if pSub.Spec.ValuesFrom != nil {
				Expect(helmRelease.Spec.ValuesFrom).To(Equal(pSub.Spec.ValuesFrom))
			}

			By("updating the status")
			profile := v1alpha1.ProfileSubscription{}
			Eventually(func() string {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: subscriptionName, Namespace: namespace}, &profile)).To(Succeed())
				return profile.Status.State
			}, 10*time.Second).Should(Equal("running"))
		},
			Entry("a single Helm chart with no supplied values", v1alpha1.ProfileSubscriptionSpec{
				ProfileURL: nginxProfileURL,
			}),
			Entry("a single Helm chart with supplied values", v1alpha1.ProfileSubscriptionSpec{
				ProfileURL: nginxProfileURL,
				Values: &apiextensionsv1.JSON{
					Raw: []byte(`{"replicaCount": 3,"service":{"port":8081}}`),
				},
			}),
			Entry("a single Helm chart with values supplied via valuesFrom", v1alpha1.ProfileSubscriptionSpec{
				ProfileURL: nginxProfileURL,
				ValuesFrom: []helmv2.ValuesReference{
					{
						Name:     "nginx-values",
						Kind:     "Secret",
						Optional: true,
					},
				},
			}),
		)

		When("retrieving the Profile Definition fails", func() {
			It("updates the status", func() {
				subscriptionName := "fetch-definition-error"
				profileURL := "https://github.com/does-not/exist"

				pSub := v1alpha1.ProfileSubscription{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ProfileSubscription",
						APIVersion: "profilesubscriptions.weave.works/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      subscriptionName,
						Namespace: namespace,
					},
					Spec: v1alpha1.ProfileSubscriptionSpec{
						ProfileURL: profileURL,
					},
				}
				Expect(k8sClient.Create(ctx, &pSub)).Should(Succeed())

				profile := v1alpha1.ProfileSubscription{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, client.ObjectKey{Name: subscriptionName, Namespace: namespace}, &profile)
					return err == nil && profile.Status != v1alpha1.ProfileSubscriptionStatus{}
				}, 10*time.Second, 1*time.Second).Should(BeTrue())

				Expect(profile.Status.Message).To(Equal("error when fetching profile definition"))
				Expect(profile.Status.State).To(Equal("failing"))
			})
		})

		When("creating Profile artifacts fail", func() {
			It("updates the status", func() {
				subscriptionName := "git-resource-already-exists-error"
				profileURL := nginxProfileURL

				gitRefName := fmt.Sprintf("%s-%s-%s", subscriptionName, "nginx-profile", "main")
				gitRepo := sourcev1.GitRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      gitRefName,
						Namespace: namespace,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "GitRepository",
						APIVersion: "source.toolkit.fluxcd.io/v1beta1",
					},
					Spec: sourcev1.GitRepositorySpec{
						URL: profileURL,
						Reference: &sourcev1.GitRepositoryRef{
							Branch: "main",
						},
					},
				}
				Expect(k8sClient.Create(ctx, &gitRepo)).Should(Succeed())

				pSub := v1alpha1.ProfileSubscription{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ProfileSubscription",
						APIVersion: "profilesubscriptions.weave.works/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      subscriptionName,
						Namespace: namespace,
					},
					Spec: v1alpha1.ProfileSubscriptionSpec{
						ProfileURL: profileURL,
					},
				}
				Expect(k8sClient.Create(ctx, &pSub)).Should(Succeed())

				profile := v1alpha1.ProfileSubscription{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, client.ObjectKey{Name: subscriptionName, Namespace: namespace}, &profile)
					return err == nil && profile.Status != v1alpha1.ProfileSubscriptionStatus{}
				}, 10*time.Second, 1*time.Second).Should(BeTrue())

				Expect(profile.Status.Message).To(Equal("error when creating profile artifacts"))
				Expect(profile.Status.State).To(Equal("failing"))
			})
		})
	})
})

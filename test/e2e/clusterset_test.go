package e2e

import (
	"context"
	"fmt"

	clustersetutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var updatedSubjectAdmin = rbacv1.Subject{
	APIGroup: "rbac.authorization.k8s.io",
	Kind:     "User",
	Name:     "admin2",
}
var updatedSubjectView = rbacv1.Subject{
	APIGroup: "rbac.authorization.k8s.io",
	Kind:     "User",
	Name:     "view2",
}

var (
	adminUser1               = "admin1"
	viewUser1                = "view1"
	clusterRoleBindingAdmin1 = "clusterSetRoleBindingAdmin1"
	clusterRoleBindingView1  = "clusterSetRoleBindingView1"
)

var _ = ginkgo.Describe("Testing ManagedClusterSet", func() {
	var (
		managedCluster              string
		managedClusterSet           string
		adminClusterSetRole         string
		viewClusterSetRole          string
		adminRoleBindingName        string
		viewRoleBindingName         string
		adminClusterRoleBindingName string
		viewClusterRoleBindingName  string
		err                         error
	)

	ginkgo.BeforeEach(func() {
		managedCluster = util.RandomName()
		managedClusterSet = util.RandomName()
		adminClusterSetRole = utils.GenerateClustersetClusterroleName(managedClusterSet, "admin")
		viewClusterSetRole = utils.GenerateClustersetClusterroleName(managedClusterSet, "view")
		adminRoleBindingName = utils.GenerateClustersetResourceRoleBindingName("admin")
		viewRoleBindingName = utils.GenerateClustersetResourceRoleBindingName("view")
		adminClusterRoleBindingName = utils.GenerateClustersetClusterRoleBindingName(managedCluster, "admin")
		viewClusterRoleBindingName = utils.GenerateClustersetClusterRoleBindingName(managedCluster, "view")

		err = util.ImportManagedCluster(clusterClient, managedCluster)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.CreateManagedClusterSet(clusterClient, managedClusterSet)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// set managedClusterSet for managedCluster
		clusterSetLabel := map[string]string{
			clustersetutils.ClusterSetLabel: managedClusterSet,
		}
		err = util.UpdateManagedClusterLabels(clusterClient, managedCluster, clusterSetLabel)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		err = util.CreateClusterRoleBindingForUser(kubeClient, clusterRoleBindingAdmin1, adminClusterSetRole, adminUser1)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		err = util.CreateClusterRoleBindingForUser(kubeClient, clusterRoleBindingView1, viewClusterSetRole, viewUser1)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
		err = util.DeleteManagedClusterSet(clusterClient, managedClusterSet)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.CleanManagedCluster(clusterClient, managedCluster)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.DeleteClusterRoleBinding(kubeClient, clusterRoleBindingAdmin1)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.DeleteClusterRoleBinding(kubeClient, clusterRoleBindingView1)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.Context("managedClusterSet admin/view clusterRole should be created/deleted automatically.", func() {
		ginkgo.It("managedClusterSet admin/view clusterRole should be created/deleted automatically", func() {
			ginkgo.By("managedClusterSet admin clusterRole should be created")
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), adminClusterSetRole, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("managedClusterSet view clusterRole should be created")
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), viewClusterSetRole, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("should delete the managedClusterSet")
			err = util.DeleteManagedClusterSet(clusterClient, managedClusterSet)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("managedClusterSet admin clusterRole should be deleted")
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), adminClusterSetRole, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

			ginkgo.By("managedClusterSet view clusterRole should be deleted")
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), viewClusterSetRole, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("managedCluster clusterRoleBinding should be created/updated automatically.", func() {
		ginkgo.It("managedCluster clusterRoleBinding should be updated successfully", func() {
			ginkgo.By("clusterSet admin clusterRoleBinding should be auto created")
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.Background(), adminClusterRoleBindingName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("update clusterSet admin clusterRoleBinding subject")
			updateAdminClusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), clusterRoleBindingAdmin1, metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			updatedAdminSubject := append(updateAdminClusterRoleBinding.Subjects, updatedSubjectAdmin)
			updateAdminClusterRoleBinding.Subjects = updatedAdminSubject

			_, err = kubeClient.RbacV1().ClusterRoleBindings().Update(context.Background(), updateAdminClusterRoleBinding, metav1.UpdateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("clusterSet admin clusterRoleBinding should be updated")
			gomega.Eventually(func() bool {
				generatedClusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.Background(), adminClusterRoleBindingName, metav1.GetOptions{})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				for _, subject := range generatedClusterRoleBinding.Subjects {
					if subject.Kind == updatedSubjectAdmin.Kind &&
						subject.Name == updatedSubjectAdmin.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			ginkgo.By("clusterSet view clusterRoleBinding should be created automatically")
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.Background(), viewClusterRoleBindingName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("update clusterSet view clusterRoleBinding subject")
			updateClusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), clusterRoleBindingView1, metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			updatedSubject := append(updateClusterRoleBinding.Subjects, updatedSubjectView)
			updateClusterRoleBinding.Subjects = updatedSubject

			_, err = kubeClient.RbacV1().ClusterRoleBindings().Update(context.Background(), updateClusterRoleBinding, metav1.UpdateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("clusterSet view clusterRoleBinding should be updated")
			gomega.Eventually(func() bool {
				generatedClusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.Background(), viewClusterRoleBindingName, metav1.GetOptions{})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				for _, subject := range generatedClusterRoleBinding.Subjects {
					if subject.Kind == updatedSubjectView.Kind &&
						subject.Name == updatedSubjectView.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

		})
	})

	ginkgo.Context("managedCluster namespace roleBinding should be created/updated automatically.", func() {
		ginkgo.It("managedCluster namespace roleBinding should be auto created successfully", func() {
			ginkgo.By("clusterSet admin roleBinding should be created")
			gomega.Eventually(func() error {
				_, err = kubeClient.RbacV1().RoleBindings(managedCluster).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("update clusterSet admin clusterRoleBinding subject")
			updateAdminClusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), clusterRoleBindingAdmin1, metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			updatedAdminSubject := append(updateAdminClusterRoleBinding.Subjects, updatedSubjectAdmin)
			updateAdminClusterRoleBinding.Subjects = updatedAdminSubject

			_, err = kubeClient.RbacV1().ClusterRoleBindings().Update(context.Background(), updateAdminClusterRoleBinding, metav1.UpdateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("clusterSet admin roleBinding should be updated")
			gomega.Eventually(func() bool {
				generatedRoleBinding, err := kubeClient.RbacV1().RoleBindings(managedCluster).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				for _, subject := range generatedRoleBinding.Subjects {
					if subject.Kind == updatedSubjectAdmin.Kind &&
						subject.Name == updatedSubjectAdmin.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			ginkgo.By("clusterSet view roleBinding should be created")
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(managedCluster).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("update clusterSet view clusterRoleBinding subject")
			updateClusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), clusterRoleBindingView1, metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			updatedSubject := append(updateClusterRoleBinding.Subjects, updatedSubjectView)
			updateClusterRoleBinding.Subjects = updatedSubject

			_, err = kubeClient.RbacV1().ClusterRoleBindings().Update(context.Background(), updateClusterRoleBinding, metav1.UpdateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("clusterSet view roleBinding should be updated")
			gomega.Eventually(func() bool {
				generatedRoleBinding, err := kubeClient.RbacV1().RoleBindings(managedCluster).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				for _, subject := range generatedRoleBinding.Subjects {
					if subject.Kind == updatedSubjectView.Kind &&
						subject.Name == updatedSubjectView.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("clusterClaim and clusterDeployment should be added into managedClusterSet automatically.", func() {
		var (
			clusterPoolNamespace string
			clusterPool          string
			clusterClaim         string
			clusterDeployment    string
		)
		ginkgo.BeforeEach(func() {
			clusterPoolNamespace = util.RandomName()
			clusterPool = util.RandomName()
			clusterClaim = util.RandomName()
			clusterDeployment = util.RandomName()
			err = util.CreateNamespace(clusterPoolNamespace)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.CreateNamespace(clusterDeployment)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			clusterSetLabel := map[string]string{"cluster.open-cluster-management.io/clusterset": managedClusterSet}
			err = util.CreateClusterPool(hiveClient, clusterPool, clusterPoolNamespace, clusterSetLabel)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.CreateClusterClaim(hiveClient, clusterClaim, clusterPoolNamespace, clusterPool)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.CreateClusterDeployment(hiveClient, clusterDeployment, clusterDeployment, clusterPool, clusterPoolNamespace)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.ClaimCluster(hiveClient, clusterDeployment, clusterDeployment, clusterClaim)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})
		ginkgo.AfterEach(func() {
			err = hiveClient.HiveV1().ClusterDeployments(clusterDeployment).Delete(context.TODO(), clusterDeployment, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = hiveClient.HiveV1().ClusterClaims(clusterPoolNamespace).Delete(context.TODO(), clusterClaim, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = hiveClient.HiveV1().ClusterPools(clusterPoolNamespace).Delete(context.TODO(), clusterPool, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.DeleteNamespace(clusterDeployment)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.DeleteNamespace(clusterPoolNamespace)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})
		ginkgo.It("clusterClaim and clusterDeployment should be added into managedClusterSet automatically.", func() {
			ginkgo.By("clusterSet admin roleBinding in clusterPool namespace is created")
			gomega.Eventually(func() error {
				_, err = kubeClient.RbacV1().RoleBindings(clusterPoolNamespace).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("clusterSet view roleBinding in clusterPool namespace is created")
			gomega.Eventually(func() error {
				_, err = kubeClient.RbacV1().RoleBindings(clusterPoolNamespace).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("clusterSet admin roleBinding in clusterDeployment namespace is created")
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(clusterDeployment).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("clusterSet view roleBinding in clusterDeployment namespace is created")
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(clusterDeployment).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("update clusterSet admin clusterRoleBinding subject")
			updateAdminClusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), clusterRoleBindingAdmin1, metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			updatedAdminSubject := append(updateAdminClusterRoleBinding.Subjects, updatedSubjectAdmin)
			updateAdminClusterRoleBinding.Subjects = updatedAdminSubject

			_, err = kubeClient.RbacV1().ClusterRoleBindings().Update(context.Background(), updateAdminClusterRoleBinding, metav1.UpdateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("clusterSet admin roleBinding should be updated")
			gomega.Eventually(func() bool {
				generatedRoleBinding, err := kubeClient.RbacV1().RoleBindings(clusterDeployment).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				for _, subject := range generatedRoleBinding.Subjects {
					if subject.Kind == updatedSubjectAdmin.Kind &&
						subject.Name == updatedSubjectAdmin.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			ginkgo.By("update clusterSet view clusterRoleBinding subject")
			updateClusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), clusterRoleBindingView1, metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			updatedSubject := append(updateClusterRoleBinding.Subjects, updatedSubjectView)
			updateClusterRoleBinding.Subjects = updatedSubject

			_, err = kubeClient.RbacV1().ClusterRoleBindings().Update(context.Background(), updateClusterRoleBinding, metav1.UpdateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("clusterSet view roleBinding should be updated")
			gomega.Eventually(func() bool {
				generatedRoleBinding, err := kubeClient.RbacV1().RoleBindings(clusterDeployment).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				for _, subject := range generatedRoleBinding.Subjects {
					if subject.Kind == updatedSubjectView.Kind &&
						subject.Name == updatedSubjectView.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("managedCluster clusterRoleBinding and namespace roleBinding should be deleted successfully after managedClusterSet is deleted", func() {
		var (
			clusterPoolNamespace string
			clusterDeployment    string
			clusterPool          string
			clusterClaim         string
		)
		ginkgo.BeforeEach(func() {
			clusterPoolNamespace = util.RandomName()
			clusterDeployment = util.RandomName()
			clusterPool = util.RandomName()
			clusterClaim = util.RandomName()
			err = util.CreateNamespace(clusterPoolNamespace)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.CreateNamespace(clusterDeployment)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			clusterSetLabel := map[string]string{"cluster.open-cluster-management.io/clusterset": managedClusterSet}
			err = util.CreateClusterPool(hiveClient, clusterPool, clusterPoolNamespace, clusterSetLabel)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.CreateClusterClaim(hiveClient, clusterClaim, clusterPoolNamespace, clusterPool)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.CreateClusterDeployment(hiveClient, clusterDeployment, clusterDeployment, clusterPool, clusterPoolNamespace)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.ClaimCluster(hiveClient, clusterDeployment, clusterDeployment, clusterClaim)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})
		ginkgo.AfterEach(func() {
			err = hiveClient.HiveV1().ClusterDeployments(clusterDeployment).Delete(context.TODO(), clusterDeployment, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = hiveClient.HiveV1().ClusterClaims(clusterPoolNamespace).Delete(context.TODO(), clusterClaim, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = hiveClient.HiveV1().ClusterPools(clusterPoolNamespace).Delete(context.TODO(), clusterPool, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.DeleteNamespace(clusterDeployment)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.DeleteNamespace(clusterPoolNamespace)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("managedCluster clusterRoleBinding and namespace roleBinding should be deleted successfully after managedClusterSet is deleted", func() {
			ginkgo.By("delete managedClusterSet")
			err = clusterClient.ClusterV1beta1().ManagedClusterSets().Delete(context.Background(), managedClusterSet, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("managedCluster admin clusterRoleBinding should be deleted")
			gomega.Eventually(func() bool {
				_, err = kubeClient.RbacV1().ClusterRoleBindings().Get(context.Background(), adminClusterRoleBindingName, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			ginkgo.By("managedCluster view clusterRoleBinding should be deleted")
			gomega.Eventually(func() bool {
				_, err = kubeClient.RbacV1().ClusterRoleBindings().Get(context.Background(), viewClusterRoleBindingName, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			ginkgo.By("managedCluster namespace admin roleBinding should be deleted")
			gomega.Eventually(func() bool {
				_, err = kubeClient.RbacV1().RoleBindings(managedCluster).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			ginkgo.By("managedCluster namespace view roleBinding should be deleted")
			gomega.Eventually(func() bool {
				_, err = kubeClient.RbacV1().RoleBindings(managedCluster).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			ginkgo.By("clusterPool namespace admin roleBinding should be deleted")
			gomega.Eventually(func() error {
				_, err = kubeClient.RbacV1().RoleBindings(clusterPoolNamespace).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

			ginkgo.By("clusterPool namespace view roleBinding should be deleted")
			gomega.Eventually(func() error {
				_, err = kubeClient.RbacV1().RoleBindings(clusterPoolNamespace).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

			ginkgo.By("clusterDeployment namespace admin roleBinding should be deleted")
			gomega.Eventually(func() error {
				_, err = kubeClient.RbacV1().RoleBindings(clusterDeployment).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

			ginkgo.By("clusterDeployment namespace view roleBinding should be deleted")
			gomega.Eventually(func() error {
				_, err = kubeClient.RbacV1().RoleBindings(clusterDeployment).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("clusterdeployment clusterset should be synced with managedcluster.", func() {
		ginkgo.BeforeEach(func() {
			err = util.CreateClusterDeployment(hiveClient, managedCluster, managedCluster, "", "")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})
		ginkgo.AfterEach(func() {
			err = hiveClient.HiveV1().ClusterDeployments(managedCluster).Delete(context.TODO(), managedCluster, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("clusterdeployment clusterset should be synced with managedcluster automatically.", func() {
			ginkgo.By("clusterdeployment is created")
			gomega.Eventually(func() error {
				clusterDeployment, err := hiveClient.HiveV1().ClusterDeployments(managedCluster).Get(context.Background(), managedCluster, metav1.GetOptions{})
				if err != nil {
					return err
				}
				clusterDeploymentSet := clusterDeployment.Labels[clustersetutils.ClusterSetLabel]
				if clusterDeploymentSet == managedClusterSet {
					return nil
				}
				return fmt.Errorf("Failed to sync clusterdeployment's clusterset.")
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("update managedcluster clusterset, clusterdeployment clusterset should be synced automatically.", func() {
			ginkgo.By("update managedcluster clusterset")
			// set managedClusterSet for managedCluster
			managedClusterSet1 := util.RandomName()
			clusterSetLabel := map[string]string{
				clustersetutils.ClusterSetLabel: managedClusterSet1,
			}
			err = util.UpdateManagedClusterLabels(clusterClient, managedCluster, clusterSetLabel)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				clusterDeployment, err := hiveClient.HiveV1().ClusterDeployments(managedCluster).Get(context.Background(), managedCluster, metav1.GetOptions{})
				if err != nil {
					return err
				}
				clusterDeploymentSet := clusterDeployment.Labels[clustersetutils.ClusterSetLabel]
				if clusterDeploymentSet == managedClusterSet1 {
					return nil
				}
				return fmt.Errorf("Failed to sync clusterdeployment's clusterset.")
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("it should fail when updating clusterpool and managedcluster clusterset.", func() {
		var (
			clusterPoolNamespace string
			clusterPool          string
			clusterClaim         string
		)
		ginkgo.BeforeEach(func() {
			clusterPoolNamespace = util.RandomName()
			clusterPool = util.RandomName()
			clusterClaim = util.RandomName()
			err = util.CreateNamespace(clusterPoolNamespace)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			clusterSetLabel := map[string]string{"cluster.open-cluster-management.io/clusterset": managedClusterSet}
			err = util.CreateClusterPool(hiveClient, clusterPool, clusterPoolNamespace, clusterSetLabel)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.CreateClusterClaim(hiveClient, clusterClaim, clusterPoolNamespace, clusterPool)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.CreateClusterDeployment(hiveClient, managedCluster, managedCluster, clusterPool, clusterPoolNamespace)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.ClaimCluster(hiveClient, managedCluster, managedCluster, clusterClaim)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.AfterEach(func() {
			err = hiveClient.HiveV1().ClusterDeployments(managedCluster).Delete(context.TODO(), managedCluster, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = hiveClient.HiveV1().ClusterClaims(clusterPoolNamespace).Delete(context.TODO(), clusterClaim, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = hiveClient.HiveV1().ClusterPools(clusterPoolNamespace).Delete(context.TODO(), clusterPool, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = util.DeleteNamespace(clusterPoolNamespace)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("try to update clusterpool clusterset label, and it should fail.", func() {
			ginkgo.By("Try to update clusterpool clusterset, and it should fail")
			managedClusterSet1 := util.RandomName()
			clusterSetLabel := map[string]string{
				clustersetutils.ClusterSetLabel: managedClusterSet1,
			}
			err = util.UpdateClusterPoolLabel(hiveClient, clusterPool, clusterPoolNamespace, clusterSetLabel)
			gomega.Expect(err).Should(gomega.HaveOccurred())
		})

		ginkgo.It("try to update managedcluster clusterset label, and it should fail.", func() {
			ginkgo.By("Try to update claimed managedcluster clusterset, and it should fail")
			managedClusterSet1 := util.RandomName()
			clusterSetLabel := map[string]string{
				clustersetutils.ClusterSetLabel: managedClusterSet1,
			}
			err = util.UpdateManagedClusterLabels(clusterClient, managedCluster, clusterSetLabel)
			gomega.Expect(err).Should(gomega.HaveOccurred())
		})
	})

})
